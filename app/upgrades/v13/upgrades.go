//nolint:revive
package v13

import (
	"bytes"
	"fmt"
	"sort"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV13UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Perform wasm key migration
		wasmStoreKey := keepers.GetKey(wasmtypes.StoreKey)
		if err := migrateWasmKeys(ctx, keepers.WasmKeeper, wasmStoreKey); err != nil {
			return nil, err
		}
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

func migrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	store := ctx.KVStore(wasmStoreKey)

	if isMigrationCompleted(store) {
		ctx.Logger().Info("WASM key migration already completed, skipping")
		return nil
	}

	ctx.Logger().Info("Starting WASM key migration from forked to original format")

	// Collect contract addresses before migration
	contractAddresses := collectContractAddresses(store)
	ctx.Logger().Info(fmt.Sprintf("Found %d contracts for migration", len(contractAddresses)))

	if len(contractAddresses) == 0 {
		ctx.Logger().Info("No contracts found for migration, this might indicate an issue")
	}

	// Save sequence keys for later migration
	sequenceKeys := saveSequenceKeys(store, ctx)

	// Perform migrations in order
	if err := migrateContractKeys(store); err != nil {
		return fmt.Errorf("failed to migrate contract keys: %w", err)
	}

	if err := migrateSequenceKeys(store, sequenceKeys, ctx); err != nil {
		return fmt.Errorf("failed to migrate sequence keys: %w", err)
	}

	if err := migrateCodeKeys(store, contractAddresses); err != nil {
		return fmt.Errorf("failed to migrate code keys: %w", err)
	}

	if err := migrateContractStoreKeys(store, contractAddresses); err != nil {
		return fmt.Errorf("failed to migrate contract store keys: %w", err)
	}

	if err := migratePrefix(store, []byte{0x06}, []byte{0x05}, "contractHistoryKey"); err != nil {
		return fmt.Errorf("failed to migrate contract history keys: %w", err)
	}

	if err := migrateSecondaryIndexKeys(store); err != nil {
		return fmt.Errorf("failed to migrate secondary index keys: %w", err)
	}

	if err := migrateParamsKey(store); err != nil {
		return fmt.Errorf("failed to migrate params key: %w", err)
	}

	markMigrationCompleted(store)
	ctx.Logger().Info("WASM key migration completed successfully")

	return nil
}

// Helper functions for migration state
func isMigrationCompleted(store sdk.KVStore) bool {
	migrationMarker := []byte("v13_wasm_migrated")
	return store.Has(migrationMarker)
}

func markMigrationCompleted(store sdk.KVStore) {
	migrationMarker := []byte("v13_wasm_migrated")
	store.Set(migrationMarker, []byte("true"))
}

// Helper for saving sequence keys
type sequenceKeys struct {
	codeIDValue     []byte
	instanceIDValue []byte
}

func saveSequenceKeys(store sdk.KVStore, ctx sdk.Context) sequenceKeys {
	oldCodeIDKey := []byte{0x01}
	oldInstanceIDKey := []byte{0x02}

	oldCodeIDValue := store.Get(oldCodeIDKey)
	oldInstanceIDValue := store.Get(oldInstanceIDKey)

	logSequenceKeyInfo(ctx, oldCodeIDValue, oldInstanceIDValue)

	var seq sequenceKeys
	if oldCodeIDValue != nil {
		seq.codeIDValue = make([]byte, len(oldCodeIDValue))
		copy(seq.codeIDValue, oldCodeIDValue)
	}

	if oldInstanceIDValue != nil {
		seq.instanceIDValue = make([]byte, len(oldInstanceIDValue))
		copy(seq.instanceIDValue, oldInstanceIDValue)
	}

	return seq
}

func logSequenceKeyInfo(ctx sdk.Context, codeIDValue, instanceIDValue []byte) {
	if codeIDValue != nil {
		ctx.Logger().Info(fmt.Sprintf("Found code ID sequence: %v", codeIDValue))
	} else {
		ctx.Logger().Info("No code ID sequence found at key 0x01")
	}

	if instanceIDValue != nil {
		ctx.Logger().Info(fmt.Sprintf("Found instance ID sequence: %v", instanceIDValue))
	} else {
		ctx.Logger().Info("No instance ID sequence found at key 0x02")
	}
}

func migrateSequenceKeys(store sdk.KVStore, seq sequenceKeys, ctx sdk.Context) error {
	oldCodeIDKey := []byte{0x01}
	oldInstanceIDKey := []byte{0x02}

	if seq.codeIDValue != nil {
		newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
		store.Set(newCodeIDKey, seq.codeIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated code ID sequence from 0x01 to %X", newCodeIDKey))
		store.Delete(oldCodeIDKey)
	}

	if seq.instanceIDValue != nil {
		newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
		store.Set(newInstanceIDKey, seq.instanceIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated instance ID sequence from 0x02 to %X", newInstanceIDKey))
		store.Delete(oldInstanceIDKey)
	}

	return nil
}

// Generic prefix migration without collision checking
func migratePrefix(store sdk.KVStore, oldPrefix, newPrefix []byte, name string) error {
	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := copyBytes(iterator.Key())
		originalValue := copyBytes(iterator.Value())

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, originalKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated %s, migratedCount %d\n", name, migratedCount)
	return nil
}

func migrateSecondaryIndexKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x10}
	newPrefix := []byte{0x06}

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var skippedRootKey int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := copyBytes(iterator.Key())
		originalValue := copyBytes(iterator.Value())

		// Skip the root key (empty key) which could be params data
		if len(originalKey) == 0 {
			fmt.Printf("Skipping root key at 0x10 (likely params data): %s\n", string(originalValue))
			skippedRootKey++
			continue
		}

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, originalKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated secondaryIndexKey, migratedCount %d, skippedRootKey %d\n",
		migratedCount, skippedRootKey)
	return nil
}

func migrateContractKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x04}
	newPrefix := []byte{0x02}

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var lengthPrefixRemovedCount int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := copyBytes(iterator.Key())
		originalValue := copyBytes(iterator.Value())

		unprefixedKey, stripped := removeLengthPrefixIfNeeded(originalKey)
		if stripped {
			lengthPrefixRemovedCount++
			fmt.Printf("Removed length prefix from contract key: %X -> %X\n", originalKey, unprefixedKey)
		}

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, unprefixedKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated contractKey, migratedCount %d, lengthPrefixRemovedCount %d\n",
		migratedCount, lengthPrefixRemovedCount)

	return nil
}

func migrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x05}
	newPrefix := []byte{0x03}

	fmt.Printf("Using %d pre-collected contracts to migrate storage\n", len(contractAddresses))

	totalMigrated := migrateContractSpecificKeys(store, oldPrefix, newPrefix, contractAddresses)
	directMigrated := migrateDirectContractStoreKeys(store, oldPrefix, newPrefix)

	fmt.Printf("Total migrated contract store keys: %d\n", totalMigrated+directMigrated)
	return nil
}

func migrateContractSpecificKeys(store sdk.KVStore, oldPrefix, newPrefix []byte, contractAddresses [][]byte) int {
	var totalMigrated int

	for i, originalContractAddr := range contractAddresses {
		if originalContractAddr == nil {
			fmt.Printf("Warning: Skipping nil contract address at index %d\n", i)
			continue
		}

		contractAddr := copyBytes(originalContractAddr)
		unprefixedAddr, stripped := removeLengthPrefixIfNeeded(contractAddr)
		if stripped {
			fmt.Printf("Stripped contract address: %X -> %X\n", contractAddr, unprefixedAddr)
		}

		oldContractPrefix := append(oldPrefix, contractAddr...)
		newContractPrefix := append(newPrefix, unprefixedAddr...)

		contractKeyCount := migrateContractStorage(store, oldContractPrefix, newContractPrefix)
		totalMigrated += contractKeyCount
		fmt.Printf("Migrated %d keys for contract %X\n", contractKeyCount, unprefixedAddr)
	}

	return totalMigrated
}

func migrateContractStorage(store sdk.KVStore, oldContractPrefix, newContractPrefix []byte) int {
	oldContractStore := prefix.NewStore(store, oldContractPrefix)
	oldContractIter := oldContractStore.Iterator(nil, nil)
	defer oldContractIter.Close()

	var contractKeyCount int
	for ; oldContractIter.Valid(); oldContractIter.Next() {
		originalKey := copyBytes(oldContractIter.Key())
		originalValue := copyBytes(oldContractIter.Value())

		if len(originalKey) == 0 {
			continue
		}

		oldFullKey := append(oldContractPrefix, originalKey...)
		newFullKey := append(newContractPrefix, originalKey...)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		contractKeyCount++
	}

	return contractKeyCount
}

func migrateDirectContractStoreKeys(store sdk.KVStore, oldPrefix, newPrefix []byte) int {
	directOldStore := prefix.NewStore(store, oldPrefix)
	directOldIter := directOldStore.Iterator(nil, nil)
	defer directOldIter.Close()

	var directMigrated int
	for ; directOldIter.Valid(); directOldIter.Next() {
		originalKey := copyBytes(directOldIter.Key())
		originalValue := copyBytes(directOldIter.Value())

		if originalKey == nil || originalValue == nil {
			continue
		}

		rebuiltKey := rebuildCompositeKey(originalKey)
		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, rebuiltKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		directMigrated++
	}

	fmt.Printf("Additionally migrated %d direct contract store keys\n", directMigrated)
	return directMigrated
}

func rebuildCompositeKey(originalKey []byte) []byte {
	if len(originalKey) > 1 {
		candidateLen := int(originalKey[0]) + 1
		if candidateLen <= len(originalKey) {
			head := originalKey[:candidateLen]
			tail := originalKey[candidateLen:]

			if unprefHead, stripped := removeLengthPrefixIfNeeded(head); stripped {
				rebuiltKey := append([]byte{}, unprefHead...)
				rebuiltKey = append(rebuiltKey, tail...)
				fmt.Printf("Stripped composite key: %X -> %X\n", originalKey, rebuiltKey)
				return rebuiltKey
			}
		}
	}
	return originalKey
}

func migrateParamsKey(store sdk.KVStore) error {
	oldKey := []byte{0x11}
	newKey := []byte{0x10}

	value := store.Get(oldKey)
	if value == nil {
		return nil
	}

	tmpValue := copyBytes(value)
	store.Set(newKey, tmpValue)
	store.Delete(oldKey)
	return nil
}

func migrateCodeKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x03}
	newPrefix := []byte{0x01}

	known := buildKnownLegitimateAddresses(contractAddresses)

	oldStore := prefix.NewStore(store, oldPrefix)
	iter := oldStore.Iterator(nil, nil)
	defer iter.Close()

	migratedCount := 0
	skippedAsContractStore := 0

	for ; iter.Valid(); iter.Next() {
		originalKey := copyBytes(iter.Key())
		originalValue := copyBytes(iter.Value())

		// If this looks like a contract store key (address + subkey), leave it alone.
		if looksLikeContractStoreKey(originalKey, known) {
			skippedAsContractStore++
			continue
		}

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, originalKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated codeKey, migratedCount %d (skipped as contractStore: %d)\n",
		migratedCount, skippedAsContractStore)
	return nil
}

// Helper utility functions
func copyBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func buildFullKey(prefix, key []byte) []byte {
	fullKey := make([]byte, 0, len(prefix)+len(key))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, key...)
	return fullKey
}

func removeLengthPrefixIfNeeded(b []byte) (out []byte, stripped bool) {
	// Check for length prefix pattern first: [len|payload]
	if len(b) > 1 && int(b[0]) == len(b)-1 {
		payload := b[1:]
		// Verify the payload is a valid address
		if err := sdk.VerifyAddressFormat(payload); err == nil {
			return bytes.Clone(payload), true
		}
	}

	// If not length-prefixed, check if already a valid address
	if err := sdk.VerifyAddressFormat(b); err == nil {
		return bytes.Clone(b), false
	}

	// Not an address format we recognize -> don't touch
	return bytes.Clone(b), false
}

func buildKnownLegitimateAddresses(contractAddresses [][]byte) []string {
	tempMap := make(map[string]bool)
	var result []string

	// First pass: identify addresses that don't need stripping
	for _, addr := range contractAddresses {
		if err := sdk.VerifyAddressFormat(addr); err == nil {
			addrStr := string(addr)
			if !tempMap[addrStr] {
				tempMap[addrStr] = true
				result = append(result, addrStr)
			}
		}
	}

	// Second pass: for potential length-prefixed addresses
	for _, addr := range contractAddresses {
		if err := sdk.VerifyAddressFormat(addr); err != nil {
			if stripped, wasStripped := removeLengthPrefixIfNeeded(addr); wasStripped {
				strippedStr := string(stripped)
				if !tempMap[strippedStr] {
					tempMap[strippedStr] = true
					result = append(result, strippedStr)
				} else {
					fmt.Printf("WARNING: Collision detected for address %X, will preserve original format\n", addr)
				}
			}
		}
	}

	sort.Strings(result)
	return result
}

func collectContractAddresses(store sdk.KVStore) [][]byte {
	contractInfoPrefix := []byte{0x04}
	contractInfoStore := prefix.NewStore(store, contractInfoPrefix)
	contractInfoIter := contractInfoStore.Iterator(nil, nil)
	defer contractInfoIter.Close()

	var contractAddresses [][]byte
	for ; contractInfoIter.Valid(); contractInfoIter.Next() {
		addr := contractInfoIter.Key()
		contractAddresses = append(contractAddresses, addr)

		fmt.Printf("Found contract address: %X (length: %d)\n", addr, len(addr))

		unprefixedAddr, stripped := removeLengthPrefixIfNeeded(addr)
		if stripped {
			fmt.Printf("  - Would be unprefixed to: %X (length: %d)\n", unprefixedAddr, len(unprefixedAddr))
		}
	}

	return contractAddresses
}

func looksLikeContractStoreKey(k []byte, knownAddrs []string) bool {
	// Case A: 1-byte length-prefixed address + subkey
	if len(k) > 1 {
		ln := int(k[0])
		if ln > 0 && 1+ln <= len(k) {
			addr := string(k[1 : 1+ln])
			for _, known := range knownAddrs {
				if known == addr && 1+ln < len(k) {
					return true
				}
			}
		}
	}

	// Case B: unprefixed address at the front + subkey
	for _, addr := range knownAddrs {
		ab := []byte(addr)
		if len(k) > len(ab) && bytes.Equal(k[:len(ab)], ab) {
			return true
		}
	}
	return false
}

// Exported functions for testing
func MigrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	return migrateWasmKeys(ctx, wasmKeeper, wasmStoreKey)
}

func RemoveLengthPrefixIfNeeded(bz []byte) ([]byte, bool) {
	return removeLengthPrefixIfNeeded(bz)
}

func CollectContractAddresses(store sdk.KVStore) [][]byte {
	return collectContractAddresses(store)
}

func MigrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	return migrateContractStoreKeys(store, contractAddresses)
}

func MigrateContractKeys(store sdk.KVStore) error {
	return migrateContractKeys(store)
}
