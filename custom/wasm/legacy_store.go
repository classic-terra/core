package wasm

import (
	"bytes"
	"io"

	coretypes "github.com/classic-terra/core/v3/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// getAddressLengthPrefix determines the correct length prefix (0x14 or 0x20) for a contract address
// by checking what was actually stored in the old database format.
//
// The challenge: When we have body = address + storage_key, we need to determine if the address
// is 20 or 32 bytes. We can't just validate the first 20 or 32 bytes because the wasmd verifier
// accepts both lengths, so it would incorrectly validate (address + partial_storage_key) as a
// valid 32-byte address.
//
// Solution: Check the old database to see what length prefix was actually used for this contract.
// We try both possibilities and see which one exists in the DB.
func (s *legacyWasmStore) getAddressLengthPrefix(body []byte) (byte, bool) {
	if len(body) < 20 {
		return 0, false // Too short to contain a contract address
	}

	// Extract potential addresses
	addr20 := body[:20]

	// Try 20-byte address first (more common)
	// Build old key with 0x14 prefix: 0x05 + 0x14 + addr20 + storage_key
	key20 := append([]byte{0x05, 0x14}, addr20...)

	// Check if any keys with this prefix exist in the DB
	iter20 := s.parent.Iterator(key20, storetypes.PrefixEndBytes(key20))
	defer iter20.Close()
	if iter20.Valid() {
		return 0x14, true
	}

	// Try 32-byte address if we have enough bytes
	if len(body) >= 32 {
		addr32 := body[:32]
		key32 := append([]byte{0x05, 0x20}, addr32...)

		iter32 := s.parent.Iterator(key32, storetypes.PrefixEndBytes(key32))
		defer iter32.Close()
		if iter32.Valid() {
			return 0x20, true
		}
	}

	// Default to 20-byte if nothing found (first page query with just address, no storage key yet)
	return 0x14, true
}

const (
	wasmMigrationHeightMainnet int64 = 25619230
	wasmMigrationHeightTestnet int64 = 26888496
	wasmMigrationHeightLocal   int64 = 200
)

func isPreWasmKeyMigration(chainID string, height int64) bool {
	if height <= 0 {
		return false
	}
	switch chainID {
	case coretypes.ColumbusChainID:
		return height < wasmMigrationHeightMainnet
	case coretypes.RebelChainID:
		return height < wasmMigrationHeightTestnet
	case "localterra", "localterra-legacy":
		return height < wasmMigrationHeightLocal
	default:
		return false
	}
}

type legacyWasmStore struct{ parent storetypes.KVStore }

var _ storetypes.KVStore = (*legacyWasmStore)(nil)

// --- Centralized mapping helpers -------------------------------------------------

var (
	// simplePrefixReverse maps new single-byte prefixes back to old (excluding specials)
	newToOldSimple = map[byte]byte{
		0x01: 0x03, // code
		0x02: 0x04, // contract
		0x03: 0x05, // contract store
		0x05: 0x06, // history
		0x06: 0x10, // secondary index
	}
	// simplePrefixForward is the opposite (old -> new) for iterator direction
	oldToNewSimple = map[byte]byte{
		0x03: 0x01,
		0x04: 0x02,
		0x05: 0x03,
		0x06: 0x05,
		0x10: 0x06,
	}
	seqPrefixNew      = byte(0x04) // new composite prefix for sequence keys
	lastCodeIDKey     = []byte("lastCodeId")
	lastContractIDKey = []byte("lastContractId")
	legacyParamsOld   = byte(0x11)
	legacyParamsNew   = byte(0x10) // new params key (empty suffix)
)

// buildLenVariants returns variants with optional length prefixes (20 & 32) for address based keys.
func buildLenVariants(basePrefix byte, addrOrRest []byte) [][]byte {
	variants := [][]byte{append([]byte{basePrefix}, addrOrRest...)}
	if l := len(addrOrRest); l >= 20 { // attempt 20-byte variant
		pref20 := append([]byte{basePrefix, 20}, addrOrRest[:20]...)
		pref20 = append(pref20, addrOrRest[20:]...)
		variants = append(variants, pref20)
	}
	if len(addrOrRest) >= 32 { // attempt 32-byte variant
		pref32 := append([]byte{basePrefix, 32}, addrOrRest[:32]...)
		pref32 = append(pref32, addrOrRest[32:]...)
		variants = append(variants, pref32)
	}
	return variants
}

// translateNewToOld maps a new-format key into one or more candidate old-format keys.
// Multi-candidate output is required for address length prefix ambiguity.
func translateNewToOld(newKey []byte) [][]byte {
	if len(newKey) == 0 {
		return [][]byte{newKey}
	}
	// Sequence keys (new composite form -> single-byte old)
	if newKey[0] == seqPrefixNew {
		if bytes.Equal(newKey, append([]byte{seqPrefixNew}, lastCodeIDKey...)) {
			return [][]byte{{0x01}}
		}
		if bytes.Equal(newKey, append([]byte{seqPrefixNew}, lastContractIDKey...)) {
			return [][]byte{{0x02}}
		}
	}
	// Params (single byte new -> single byte old)
	if newKey[0] == legacyParamsNew && len(newKey) == 1 {
		return [][]byte{{legacyParamsOld}}
	}
	// Simple reversible prefixes
	if oldPref, ok := newToOldSimple[newKey[0]]; ok {
		body := newKey[1:]
		// For contract(0x02) and contract store(0x03) we provide variants with optional length prefixes.
		if newKey[0] == 0x02 || newKey[0] == 0x03 {
			return buildLenVariants(oldPref, body)
		}
		return [][]byte{append([]byte{oldPref}, body...)}
	}
	// Fallback: no translation (should not normally happen)
	return [][]byte{newKey}
}

// mapOldToNew converts an old-format key to new-format; returns nil if not a wasm key we care about.
func mapOldToNew(old []byte) []byte {
	if len(old) == 0 {
		return nil
	}
	// Sequence
	if old[0] == 0x01 && len(old) == 1 {
		return append([]byte{seqPrefixNew}, lastCodeIDKey...)
	}
	if old[0] == 0x02 && len(old) == 1 {
		return append([]byte{seqPrefixNew}, lastContractIDKey...)
	}
	// Params
	if old[0] == legacyParamsOld && len(old) == 1 {
		return []byte{legacyParamsNew}
	}
	// Simple prefixes
	if newPref, ok := oldToNewSimple[old[0]]; ok {
		body := old[1:]
		switch old[0] {
		case 0x04: // contract: possible length prefix to strip
			body = stripLegacyLenPrefix(body)
		case 0x05: // contract store: body may contain addr(+lenprefix)+suffix
			body = rebuildContractStoreBody(body)
		}
		return append([]byte{newPref}, body...)
	}
	// history (0x06) and secondary index (0x10) are covered above; if not matched return nil
	return nil
}

// rebuildContractStoreBody reconstructs (addr+suffix) with any legacy len prefix removed.
func rebuildContractStoreBody(rest []byte) []byte {
	// If length prefixed (20 or 32) we drop that single length byte.
	if len(rest) > 0 && (rest[0] == 20 || rest[0] == 32) {
		// We cannot reliably know addr length if suffix appended; assume first byte declares address length.
		ln := int(rest[0])
		if len(rest) >= 1+ln { // minimal safety check
			// Skip the length prefix byte, keep address + suffix
			return rest[1:]
		}
	}
	return rest
}

func stripLegacyLenPrefix(b []byte) []byte {
	if len(b) >= 21 && b[0] == 20 && int(b[0]) == len(b)-1 { // 20-byte address prefixed
		return b[1:]
	}
	return b
}

func (s *legacyWasmStore) Get(key []byte) []byte {
	for _, cand := range translateNewToOld(key) {
		if bz := s.parent.Get(cand); bz != nil {
			return bz
		}
	}
	return nil
}

func (s *legacyWasmStore) Has(key []byte) bool {
	for _, c := range translateNewToOld(key) {
		if s.parent.Has(c) {
			return true
		}
	}
	return false
}

func (s *legacyWasmStore) Set(_, _ []byte) {
	// Set is a no-op in the legacy store (queries are read-only)
}

func (s *legacyWasmStore) Delete(_ []byte) {
	// Delete is a no-op in the legacy store (queries are read-only)
}

func (s *legacyWasmStore) Iterator(start, end []byte) storetypes.Iterator {
	// Translate bounds to old format for efficient iteration
	oldStart, oldEnd := s.translateBoundsForIteration(start, end)
	return newLegacyIterator(s.parent.Iterator(oldStart, oldEnd), start, end)
}

func (s *legacyWasmStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	oldStart, oldEnd := s.translateBoundsForIteration(start, end)
	return newLegacyIterator(s.parent.ReverseIterator(oldStart, oldEnd), start, end)
}

// translateBoundsForIteration converts new-format bounds to old-format for the underlying iterator
func (s *legacyWasmStore) translateBoundsForIteration(start, end []byte) ([]byte, []byte) {
	if len(start) == 0 && len(end) == 0 {
		return nil, nil
	}

	// For contract store queries (prefix 0x03), translate to old format (prefix 0x05)
	if len(start) > 0 && start[0] == 0x03 {
		// Old format: 0x05 + length_prefix + address + storage_key
		// New format: 0x03 + address + storage_key
		// Determine correct length prefix (0x14 for 20-byte or 0x20 for 32-byte addresses)

		body := start[1:] // address + storage_key
		var oldStart []byte

		if lenPrefix, ok := s.getAddressLengthPrefix(body); ok {
			oldStart = append([]byte{0x05, lenPrefix}, body...)
		} else {
			// Invalid address in query bounds - return empty range to prevent full DB scan.
			// Using [0x05, 0xff] creates an impossible range (0xff > valid length prefixes 0x14/0x20)
			// that immediately returns zero results instead of scanning the entire database.
			// This protects against DoS attacks using malformed pagination queries.
			return []byte{0x05, 0xff}, []byte{0x05, 0xff}
		}

		var oldEnd []byte
		if len(end) > 0 && end[0] == 0x03 {
			bodyEnd := end[1:]
			if lenPrefix, ok := s.getAddressLengthPrefix(bodyEnd); ok {
				oldEnd = append([]byte{0x05, lenPrefix}, bodyEnd...)
			} else {
				// Invalid address in end bound - use start as both bounds to create empty range.
				// This prevents nil bounds which would trigger full DB scan from beginning.
				return oldStart, oldStart
			}
		}

		return oldStart, oldEnd
	}

	// For other prefixes, use the translateNewToOld logic
	var oldStart, oldEnd []byte
	if len(start) > 0 {
		candidates := translateNewToOld(start)
		if len(candidates) > 0 {
			oldStart = candidates[0]
			for _, c := range candidates[1:] {
				if bytes.Compare(c, oldStart) < 0 {
					oldStart = c
				}
			}
		}
	}
	if len(end) > 0 {
		candidates := translateNewToOld(end)
		if len(candidates) > 0 {
			oldEnd = candidates[0]
			for _, c := range candidates[1:] {
				if bytes.Compare(c, oldEnd) > 0 {
					oldEnd = c
				}
			}
		}
	}
	return oldStart, oldEnd
}

func (s *legacyWasmStore) GetStoreType() storetypes.StoreType {
	if gt, ok := s.parent.(interface{ GetStoreType() storetypes.StoreType }); ok {
		return gt.GetStoreType()
	}
	return storetypes.StoreTypeIAVL
}

// iterator translating old keys to new keys
type legacyIterator struct {
	under storetypes.Iterator
	start []byte
	end   []byte
	valid bool
	key   []byte
	val   []byte
}

func newLegacyIterator(under storetypes.Iterator, start, end []byte) *legacyIterator {
	it := &legacyIterator{under: under, start: start, end: end}
	it.advance()
	return it
}
func (it *legacyIterator) Domain() ([]byte, []byte) { return it.start, it.end }
func (it *legacyIterator) Valid() bool              { return it.valid }
func (it *legacyIterator) Key() []byte              { return it.key }
func (it *legacyIterator) Value() []byte            { return it.val }
func (it *legacyIterator) Next()                    { it.advance() }
func (it *legacyIterator) Close() error             { it.under.Close(); return nil }
func (it *legacyIterator) Error() error             { return nil }

func (it *legacyIterator) advance() {
	for ; it.under.Valid(); it.under.Next() {
		oldKey := it.under.Key()
		newKey := mapOldToNew(oldKey)
		if newKey == nil {
			continue
		}
		if !rangeOK(newKey, it.start, it.end) {
			continue
		}
		it.key = newKey
		it.val = it.under.Value()
		it.valid = true
		it.under.Next() // Advance before returning, since post-statement won't run
		return
	}
	it.valid = false
}

func rangeOK(k, start, end []byte) bool {
	if start != nil && bytes.Compare(k, start) < 0 {
		return false
	}
	if end != nil && bytes.Compare(k, end) >= 0 {
		return false
	}
	return true
}

// CacheWrap just delegates (historic queries are read-only, but we delegate for compatibility)
func (s *legacyWasmStore) CacheWrap() storetypes.CacheWrap { return s.parent.CacheWrap() }

func (s *legacyWasmStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	// pass through to underlying store; translation occurs on outer layer
	if cw, ok := s.parent.(interface {
		CacheWrapWithTrace(io.Writer, storetypes.TraceContext) storetypes.CacheWrap
	}); ok {
		return cw.CacheWrapWithTrace(w, tc)
	}
	return s.parent.CacheWrap()
}

type legacyMultiStore struct {
	storetypes.MultiStore
	wasmKey storetypes.StoreKey
	legacy  storetypes.KVStore
}

func (l legacyMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	if key.Name() == l.wasmKey.Name() {
		return l.legacy
	}
	return l.MultiStore.GetKVStore(key)
}

// prepareLegacyWasmContext wraps the wasm KVStore with a translating legacy store.
// The real mounted wasm store key is injected (dependency injection) instead of
// being discovered via reflection/unsafe.
func prepareLegacyWasmContext(ctx sdk.Context, wasmKey storetypes.StoreKey) (sdk.Context, bool) {
	if wasmKey == nil || !isPreWasmKeyMigration(ctx.ChainID(), ctx.BlockHeight()) {
		return ctx, false
	}
	legacyStore := &legacyWasmStore{parent: ctx.KVStore(wasmKey)}
	wrapped := legacyMultiStore{MultiStore: ctx.MultiStore(), wasmKey: wasmKey, legacy: legacyStore}
	newCtx := sdk.NewContext(wrapped, ctx.BlockHeader(), ctx.IsCheckTx(), ctx.Logger())
	newCtx = newCtx.WithGasMeter(ctx.GasMeter()).WithEventManager(ctx.EventManager()).WithChainID(ctx.ChainID())
	return newCtx, true
}
