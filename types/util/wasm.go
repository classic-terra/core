package util

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// stripLenPrefixAddrOnly safely strips length prefix only for known address keys
// Returns the stripped address and whether stripping occurred
func stripLenPrefixAddrOnly(b []byte) (out []byte, stripped bool) {
	if len(b) == 21 && int(b[0]) == 20 {
		out = make([]byte, 20)
		copy(out, b[1:21])
		return out, true
	}
	out = make([]byte, len(b))
	copy(out, b)
	return out, false
}

// ReadContractHistoryWithFallback reads contract history with fallback to old prefix for backward compatibility
// This function handles the case where some contract history data might not have been migrated yet
func ReadContractHistoryWithFallback(store sdk.KVStore, contractAddr sdk.AccAddress) ([]byte, bool) {
	// First try to read from the new prefix (0x05) - migrated data
	newPrefix := []byte{0x05}
	newKey := make([]byte, 0, len(newPrefix)+len(contractAddr))
	newKey = append(newKey, newPrefix...)
	newKey = append(newKey, contractAddr...)
	value := store.Get(newKey)

	if value != nil {
		return value, true
	}

	// If not found in new location, try the old prefix (0x06) - unmigrated data
	oldPrefix := []byte{0x06}

	// Handle potential length-prefixed addresses in old data
	// Try with original address first
	oldKey := make([]byte, 0, len(oldPrefix)+len(contractAddr))
	oldKey = append(oldKey, oldPrefix...)
	oldKey = append(oldKey, contractAddr...)
	value = store.Get(oldKey)

	if value != nil {
		return value, true
	}

	// Also try with length-prefixed address for old data
	lengthPrefixedAddr := make([]byte, 0, 1+len(contractAddr))
	lengthPrefixedAddr = append(lengthPrefixedAddr, byte(len(contractAddr)))
	lengthPrefixedAddr = append(lengthPrefixedAddr, contractAddr...)

	oldKeyWithPrefix := make([]byte, 0, len(oldPrefix)+len(lengthPrefixedAddr))
	oldKeyWithPrefix = append(oldKeyWithPrefix, oldPrefix...)
	oldKeyWithPrefix = append(oldKeyWithPrefix, lengthPrefixedAddr...)
	value = store.Get(oldKeyWithPrefix)

	if value != nil {
		return value, true
	}

	return nil, false
}

// IterateContractHistoryWithFallback iterates over contract history with fallback support
// This function handles both migrated (0x05) and unmigrated (0x06) contract history entries
func IterateContractHistoryWithFallback(store sdk.KVStore, cb func(contractAddr []byte, history []byte) bool) {
	// First iterate over new prefix (0x05) - migrated data
	newPrefix := []byte{0x05}
	newStore := prefix.NewStore(store, newPrefix)
	newIter := newStore.Iterator(nil, nil)
	defer newIter.Close()

	processedContracts := make(map[string]bool)

	for ; newIter.Valid(); newIter.Next() {
		contractAddr := newIter.Key()
		history := newIter.Value()

		// Mark this contract as processed to avoid duplicates
		processedContracts[string(contractAddr)] = true

		if !cb(contractAddr, history) {
			return
		}
	}

	// Then iterate over old prefix (0x06) - unmigrated data
	// Only process contracts that haven't been processed yet
	oldPrefix := []byte{0x06}
	oldStore := prefix.NewStore(store, oldPrefix)
	oldIter := oldStore.Iterator(nil, nil)
	defer oldIter.Close()

	for ; oldIter.Valid(); oldIter.Next() {
		contractAddr := oldIter.Key()
		history := oldIter.Value()

		// Remove length prefix if present for comparison
		unprefixedAddr, _ := stripLenPrefixAddrOnly(contractAddr)

		// Skip if we already processed this contract from the new prefix
		if processedContracts[string(unprefixedAddr)] {
			continue
		}

		// Use the unprefixed address for consistency
		if !cb(unprefixedAddr, history) {
			return
		}
	}
}

// ReadContractInfoWithFallback reads contract info with fallback to old prefix
func ReadContractInfoWithFallback(store sdk.KVStore, contractAddr sdk.AccAddress) ([]byte, bool) {
	// New prefix for contract info after migration: 0x02
	newPrefix := []byte{0x02}
	newKey := make([]byte, 0, len(newPrefix)+len(contractAddr))
	newKey = append(newKey, newPrefix...)
	newKey = append(newKey, contractAddr...)
	if v := store.Get(newKey); v != nil {
		return v, true
	}

	// Old prefix before migration: 0x04 (may include length-prefixed address)
	oldPrefix := []byte{0x04}
	oldKey := make([]byte, 0, len(oldPrefix)+len(contractAddr))
	oldKey = append(oldKey, oldPrefix...)
	oldKey = append(oldKey, contractAddr...)
	if v := store.Get(oldKey); v != nil {
		return v, true
	}

	// Try length-prefixed variant
	lengthPrefixedAddr := make([]byte, 0, 1+len(contractAddr))
	lengthPrefixedAddr = append(lengthPrefixedAddr, byte(len(contractAddr)))
	lengthPrefixedAddr = append(lengthPrefixedAddr, contractAddr...)

	oldKeyWithPrefix := make([]byte, 0, len(oldPrefix)+len(lengthPrefixedAddr))
	oldKeyWithPrefix = append(oldKeyWithPrefix, oldPrefix...)
	oldKeyWithPrefix = append(oldKeyWithPrefix, lengthPrefixedAddr...)
	if v := store.Get(oldKeyWithPrefix); v != nil {
		return v, true
	}

	return nil, false
}

// ReadRawContractStateWithFallback reads a single contract store entry by key with fallback
func ReadRawContractStateWithFallback(store sdk.KVStore, contractAddr []byte, key []byte) ([]byte, bool) {
	// New: 0x03 | addr | key
	newPrefix := []byte{0x03}
	newKey := make([]byte, 0, len(newPrefix)+len(contractAddr)+len(key))
	newKey = append(newKey, newPrefix...)
	newKey = append(newKey, contractAddr...)
	newKey = append(newKey, key...)
	if v := store.Get(newKey); v != nil {
		return v, true
	}

	// Old: 0x05 | addr | key
	oldPrefix := []byte{0x05}
	oldKey := make([]byte, 0, len(oldPrefix)+len(contractAddr)+len(key))
	oldKey = append(oldKey, oldPrefix...)
	oldKey = append(oldKey, contractAddr...)
	oldKey = append(oldKey, key...)
	if v := store.Get(oldKey); v != nil {
		return v, true
	}

	// Old with length-prefixed address
	lengthPrefixedAddr := make([]byte, 0, 1+len(contractAddr))
	lengthPrefixedAddr = append(lengthPrefixedAddr, byte(len(contractAddr)))
	lengthPrefixedAddr = append(lengthPrefixedAddr, contractAddr...)

	oldKeyWithPrefix := make([]byte, 0, len(oldPrefix)+len(lengthPrefixedAddr)+len(key))
	oldKeyWithPrefix = append(oldKeyWithPrefix, oldPrefix...)
	oldKeyWithPrefix = append(oldKeyWithPrefix, lengthPrefixedAddr...)
	oldKeyWithPrefix = append(oldKeyWithPrefix, key...)
	if v := store.Get(oldKeyWithPrefix); v != nil {
		return v, true
	}

	return nil, false
}

// IterateAllContractStateWithFallback iterates all contract store entries for a contract with fallback
func IterateAllContractStateWithFallback(store sdk.KVStore, contractAddr []byte, cb func(key, value []byte) bool) {
	visited := make(map[string]bool)

	// New prefix first: 0x03 | addr | ...
	newContractPrefix := make([]byte, 0, 1+len(contractAddr))
	newContractPrefix = append(newContractPrefix, 0x03)
	newContractPrefix = append(newContractPrefix, contractAddr...)

	newStore := prefix.NewStore(store, newContractPrefix)
	newIter := newStore.Iterator(nil, nil)
	defer newIter.Close()
	for ; newIter.Valid(); newIter.Next() {
		k := append([]byte{}, newIter.Key()...)
		v := append([]byte{}, newIter.Value()...)
		visited[string(k)] = true
		if !cb(k, v) {
			return
		}
	}

	// Old prefix without length prefix: 0x05 | addr | ...
	oldContractPrefix := make([]byte, 0, 1+len(contractAddr))
	oldContractPrefix = append(oldContractPrefix, 0x05)
	oldContractPrefix = append(oldContractPrefix, contractAddr...)

	oldStore := prefix.NewStore(store, oldContractPrefix)
	oldIter := oldStore.Iterator(nil, nil)
	for ; oldIter.Valid(); oldIter.Next() {
		k := append([]byte{}, oldIter.Key()...)
		if visited[string(k)] {
			continue
		}
		v := append([]byte{}, oldIter.Value()...)
		if !cb(k, v) {
			oldIter.Close()
			return
		}
	}
	oldIter.Close()

	// Old prefix with length-prefixed address: 0x05 | [len]|addr | ...
	lengthPrefixedAddr := make([]byte, 0, 1+len(contractAddr))
	lengthPrefixedAddr = append(lengthPrefixedAddr, byte(len(contractAddr)))
	lengthPrefixedAddr = append(lengthPrefixedAddr, contractAddr...)

	oldPrefixedContractPrefix := make([]byte, 0, 1+len(lengthPrefixedAddr))
	oldPrefixedContractPrefix = append(oldPrefixedContractPrefix, 0x05)
	oldPrefixedContractPrefix = append(oldPrefixedContractPrefix, lengthPrefixedAddr...)

	oldPrefixedStore := prefix.NewStore(store, oldPrefixedContractPrefix)
	oldPrefIter := oldPrefixedStore.Iterator(nil, nil)
	defer oldPrefIter.Close()
	for ; oldPrefIter.Valid(); oldPrefIter.Next() {
		k := append([]byte{}, oldPrefIter.Key()...)
		if visited[string(k)] {
			continue
		}
		v := append([]byte{}, oldPrefIter.Value()...)
		if !cb(k, v) {
			return
		}
	}
}
