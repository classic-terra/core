//nolint:revive
package v13

import (
	"fmt"

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

	ctx.Logger().Info("Starting WASM key migration from forked to original format")

	// First, collect all contract addresses before any migration
	contractAddresses := collectContractAddresses(store)
	ctx.Logger().Info(fmt.Sprintf("Found %d contracts for migration", len(contractAddresses)))

	// Add validation of collected addresses
	if len(contractAddresses) == 0 {
		ctx.Logger().Info("No contracts found for migration, this might indicate an issue")
	}

	// 1. Save sequence keys (0x01, 0x02) to temporary variables
	oldCodeIDKey := []byte{0x01}
	oldCodeIDValue := store.Get(oldCodeIDKey)

	oldInstanceIDKey := []byte{0x02}
	oldInstanceIDValue := store.Get(oldInstanceIDKey)

	if oldCodeIDValue != nil {
		ctx.Logger().Info(fmt.Sprintf("Found code ID sequence: %v", oldCodeIDValue))
	} else {
		ctx.Logger().Info("No code ID sequence found at key 0x01")
	}

	if oldInstanceIDValue != nil {
		ctx.Logger().Info(fmt.Sprintf("Found instance ID sequence: %v", oldInstanceIDValue))
	} else {
		ctx.Logger().Info("No instance ID sequence found at key 0x02")
	}

	// Make copies to avoid any issues with shared memory
	var codeIDValue, instanceIDValue []byte
	if oldCodeIDValue != nil {
		codeIDValue = make([]byte, len(oldCodeIDValue))
		copy(codeIDValue, oldCodeIDValue)
	}

	if oldInstanceIDValue != nil {
		instanceIDValue = make([]byte, len(oldInstanceIDValue))
		copy(instanceIDValue, oldInstanceIDValue)
	}

	// 2.1 Migrate contract keys (0x04 -> 0x02)
	// This needs to happen before we write to 0x02 with sequence keys
	if err := migrateContractKeys(store); err != nil {
		return fmt.Errorf("failed to migrate contract keys: %w", err)
	}

	// 2.2. Now that 0x04 is free, manually migrate sequence keys from our saved copies
	if codeIDValue != nil {
		newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
		store.Set(newCodeIDKey, codeIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated code ID sequence from 0x01 to %X", newCodeIDKey))
		store.Delete(oldCodeIDKey)
	}

	if instanceIDValue != nil {
		newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
		store.Set(newInstanceIDKey, instanceIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated instance ID sequence from 0x02 to %X", newInstanceIDKey))
		store.Delete(oldInstanceIDKey)
	}

	// 3. Migrate code keys (0x03 -> 0x01)
	// This can only be done after sequence keys are migrated away from 0x01
	if err := migrateCodeKeys(store); err != nil {
		return err
	}

	// 4. Migrate contract store keys (0x05 -> 0x03)
	// This needs to happen before contract history keys migration
	if err := migrateContractStoreKeys(store, contractAddresses); err != nil {
		return err
	}

	// 5. Migrate contract history keys (0x06 -> 0x05)
	// This can only be done after contract store keys are migrated away from 0x05
	if err := migrateContractHistoryKeys(store); err != nil {
		return err
	}

	// 6. Migrate secondary index keys (0x10 -> 0x06)
	// This needs to happen before params key migration to free up 0x10
	if err := migrateSecondaryIndexKeys(store); err != nil {
		return err
	}

	// 7. Migrate params key (0x11 -> 0x10)
	// Now that 0x10 is free, we can safely migrate params
	if err := migrateParamsKey(store); err != nil {
		return err
	}

	ctx.Logger().Info("WASM key migration completed successfully")

	return nil
}

// migrateCodeKeys migrates code keys from 0x03 to 0x01
func migrateCodeKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x03}
	newPrefix := []byte{0x01}
	return migratePrefix(store, oldPrefix, newPrefix, "codeKey")
}

// removeLengthPrefixIfNeeded checks if a key has a length prefix and removes it if present
func removeLengthPrefixIfNeeded(bz []byte) []byte {
	if len(bz) == 0 {
		return bz
	}

	// Check if this looks like a length-prefixed address
	// The first byte should indicate the length of the remaining bytes
	prefixLen := int(bz[0])

	// Validate that the prefix length makes sense:
	// 1. It should be positive
	// 2. It should be less than the total length minus 1 (for the prefix byte itself)
	// 3. For Cosmos addresses, it's typically 20 bytes
	if prefixLen > 0 && prefixLen <= len(bz)-1 && prefixLen == len(bz)-1 {
		// This is likely a length-prefixed address
		fmt.Printf("Found length prefix: original %X, unprefixed %X\n", bz, bz[1:])
		return bz[1:] // Remove the first byte (length prefix)
	}

	// If the key is longer than 20 bytes and starts with a length prefix, try to remove it
	if len(bz) > 20 && prefixLen == 20 {
		fmt.Printf("Found potential length prefix in long key: original %X, unprefixed %X\n", bz, bz[1:])
		return bz[1:] // Remove the first byte (length prefix)
	}

	fmt.Printf("No length prefix found, returning original: %X\n", bz)
	return bz // Return as is if not length-prefixed
}

// migrateContractHistoryKeys migrates contract history keys from 0x06 to 0x05
func migrateContractHistoryKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x06}
	newPrefix := []byte{0x05}
	return migratePrefix(store, oldPrefix, newPrefix, "contractHistoryKey")
}

// migrateSecondaryIndexKeys migrates secondary index keys from 0x10 to 0x06
func migrateSecondaryIndexKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x10}
	newPrefix := []byte{0x06}
	return migratePrefix(store, oldPrefix, newPrefix, "secondaryIndexKey")
}

// migrateParamsKey migrates params key from 0x11 to 0x10
func migrateParamsKey(store sdk.KVStore) error {
	oldKey := []byte{0x11}
	newKey := []byte{0x10}

	value := store.Get(oldKey)
	if value != nil {
		tmpValue := make([]byte, len(value))
		copy(tmpValue, value)
		store.Set(newKey, tmpValue)
		store.Delete(oldKey)
	}

	return nil
}

// migrateContractKeys migrates contract keys from 0x04 to 0x02
// and removes length prefixes from addresses
func migrateContractKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x04}
	newPrefix := []byte{0x02}

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var lengthPrefixRemovedCount int

	for ; iterator.Valid(); iterator.Next() {
		// Copy the key and value to avoid issues with shared memory
		originalKey := make([]byte, len(iterator.Key()))
		copy(originalKey, iterator.Key())

		originalValue := make([]byte, len(iterator.Value()))
		copy(originalValue, iterator.Value())

		// The key is the contract address with potential length prefix
		// We need to check if it has a length prefix and remove it
		unprefixedKey := removeLengthPrefixIfNeeded(originalKey)

		// Track if we removed a length prefix
		if len(unprefixedKey) != len(originalKey) {
			lengthPrefixRemovedCount++
			fmt.Printf("Removed length prefix from contract key: %X -> %X\n",
				originalKey, unprefixedKey)
		}

		// Construct full keys
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, unprefixedKey...)

		// Set with new prefix and delete old
		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)

		migratedCount++
	}

	fmt.Printf("migrated contractKey, migratedCount %d, lengthPrefixRemovedCount %d\n",
		migratedCount, lengthPrefixRemovedCount)

	return nil
}

// migrateContractStoreKeys migrates contract store keys from 0x05 to 0x03
// and removes length prefixes from addresses in the keys
func migrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x05}
	newPrefix := []byte{0x03}

	fmt.Printf("Using %d pre-collected contracts to migrate storage\n", len(contractAddresses))

	// Now migrate each contract's storage
	var totalMigrated int
	for i, originalContractAddr := range contractAddresses {
		// Skip nil addresses if any
		if originalContractAddr == nil {
			fmt.Printf("Warning: Skipping nil contract address at index %d\n", i)
			continue
		}

		// Copy the contract address to avoid issues with shared memory
		contractAddr := make([]byte, len(originalContractAddr))
		copy(contractAddr, originalContractAddr)

		// Remove length prefix from contract address if needed
		unprefixedAddr := removeLengthPrefixIfNeeded(contractAddr)

		// Construct the old and new prefixes for this specific contract
		oldContractPrefix := append([]byte{0x05}, contractAddr...)   // Original key with potential length prefix
		newContractPrefix := append([]byte{0x03}, unprefixedAddr...) // New key without length prefix

		// Create iterator for this contract's storage
		oldContractStore := prefix.NewStore(store, oldContractPrefix)
		oldContractIter := oldContractStore.Iterator(nil, nil)

		var contractKeyCount int
		for ; oldContractIter.Valid(); oldContractIter.Next() {
			// Copy the key and value to avoid issues with shared memory
			originalKey := make([]byte, len(oldContractIter.Key()))
			copy(originalKey, oldContractIter.Key())

			originalValue := make([]byte, len(oldContractIter.Value()))
			copy(originalValue, oldContractIter.Value())

			// Skip empty keys or values
			if len(originalKey) == 0 || len(originalValue) == 0 {
				continue
			}

			// Construct full keys - create new slices to avoid modifying the original prefixes
			oldFullKey := append([]byte{}, oldContractPrefix...)
			oldFullKey = append(oldFullKey, originalKey...)

			newFullKey := append([]byte{}, newContractPrefix...)
			newFullKey = append(newFullKey, originalKey...)

			// Set with new prefix and delete old
			store.Set(newFullKey, originalValue)
			store.Delete(oldFullKey)

			contractKeyCount++
			totalMigrated++
		}
		oldContractIter.Close()

		fmt.Printf("Migrated %d keys for contract %X\n", contractKeyCount, unprefixedAddr)
	}

	// Also handle any direct contract store keys that might not be associated with a contract
	// (this is a fallback to ensure we don't miss anything)
	directOldStore := prefix.NewStore(store, oldPrefix)
	directOldIter := directOldStore.Iterator(nil, nil)

	var directMigrated int
	for ; directOldIter.Valid(); directOldIter.Next() {
		// Copy the key and value to avoid issues with shared memory
		originalKey := make([]byte, len(directOldIter.Key()))
		copy(originalKey, directOldIter.Key())

		originalValue := make([]byte, len(directOldIter.Value()))
		copy(originalValue, directOldIter.Value())

		// Skip empty keys or values
		if len(originalKey) == 0 || len(originalValue) == 0 {
			continue
		}

		// Check if the key starts with a length prefix and remove it
		unprefixedKey := removeLengthPrefixIfNeeded(originalKey)

		// Construct full keys - create new slices to avoid modifying the original prefixes
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, unprefixedKey...)

		// Set with new prefix and delete old
		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)

		directMigrated++
	}
	directOldIter.Close()

	fmt.Printf("Additionally migrated %d direct contract store keys\n", directMigrated)
	fmt.Printf("Total migrated contract store keys: %d\n", totalMigrated+directMigrated)

	return nil
}

// migratePrefix is a helper function to migrate all keys with a given prefix
func migratePrefix(store sdk.KVStore, oldPrefix, newPrefix []byte, name string) error {
	oldStore := prefix.NewStore(store, oldPrefix)
	newStore := prefix.NewStore(store, newPrefix)

	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int

	for ; iterator.Valid(); iterator.Next() {
		// Copy the key and value to avoid issues with shared memory
		originalKey := make([]byte, len(iterator.Key()))
		copy(originalKey, iterator.Key())

		originalValue := make([]byte, len(iterator.Value()))
		copy(originalValue, iterator.Value())

		newStore.Set(originalKey, originalValue)
		oldStore.Delete(originalKey)
		migratedCount++
	}

	fmt.Printf("migrated name %s, migratedCount %d\n", name, migratedCount)

	return nil
}

// collectContractAddresses gets all contract addresses before any migration
func collectContractAddresses(store sdk.KVStore) [][]byte {
	// Contract addresses are stored with prefix 0x04 before migration
	contractInfoPrefix := []byte{0x04}
	contractInfoStore := prefix.NewStore(store, contractInfoPrefix)
	contractInfoIter := contractInfoStore.Iterator(nil, nil)
	defer contractInfoIter.Close()

	var contractAddresses [][]byte
	for ; contractInfoIter.Valid(); contractInfoIter.Next() {
		// The key is the contract address (potentially with length prefix)
		addr := contractInfoIter.Key()
		contractAddresses = append(contractAddresses, addr)

		// Log each contract address for debugging
		fmt.Printf("Found contract address: %X (length: %d)\n", addr, len(addr))

		// Also log what it would look like unprefixed
		unprefixedAddr := removeLengthPrefixIfNeeded(addr)
		if len(addr) != len(unprefixedAddr) {
			fmt.Printf("  - Would be unprefixed to: %X (length: %d)\n", unprefixedAddr, len(unprefixedAddr))
		}
	}

	return contractAddresses
}

// MigrateWasmKeys Exported for testing
func MigrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	return migrateWasmKeys(ctx, wasmKeeper, wasmStoreKey)
}

// RemoveLengthPrefixIfNeeded Exported for testing
func RemoveLengthPrefixIfNeeded(bz []byte) []byte {
	return removeLengthPrefixIfNeeded(bz)
}

// CollectContractAddresses Exported for testing
func CollectContractAddresses(store sdk.KVStore) [][]byte {
	return collectContractAddresses(store)
}

// MigrateContractStoreKeys Exported for testing
func MigrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	return migrateContractStoreKeys(store, contractAddresses)
}

// MigrateContractKeys Exported for testing
func MigrateContractKeys(store sdk.KVStore) error {
	return migrateContractKeys(store)
}

// ReadContractHistoryWithFallback reads contract history with fallback to old prefix for backward compatibility
// This function handles the case where some contract history data might not have been migrated yet
func ReadContractHistoryWithFallback(store sdk.KVStore, contractAddr sdk.AccAddress) ([]byte, bool) {
	// First try to read from the new prefix (0x05) - migrated data
	newPrefix := []byte{0x05}
	newKey := append(newPrefix, contractAddr...)
	value := store.Get(newKey)

	if value != nil {
		return value, true
	}

	// If not found in new location, try the old prefix (0x06) - unmigrated data
	oldPrefix := []byte{0x06}

	// Handle potential length-prefixed addresses in old data
	// Try with original address first
	oldKey := append(oldPrefix, contractAddr...)
	value = store.Get(oldKey)

	if value != nil {
		return value, true
	}

	// Also try with length-prefixed address for old data
	lengthPrefixedAddr := append([]byte{byte(len(contractAddr))}, contractAddr...)
	oldKeyWithPrefix := append(oldPrefix, lengthPrefixedAddr...)
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
		unprefixedAddr := removeLengthPrefixIfNeeded(contractAddr)

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
	newKey := append(newPrefix, contractAddr...)
	if v := store.Get(newKey); v != nil {
		return v, true
	}

	// Old prefix before migration: 0x04 (may include length-prefixed address)
	oldPrefix := []byte{0x04}
	oldKey := append(oldPrefix, contractAddr...)
	if v := store.Get(oldKey); v != nil {
		return v, true
	}

	// Try length-prefixed variant
	lengthPrefixedAddr := append([]byte{byte(len(contractAddr))}, contractAddr...)
	oldKeyWithPrefix := append(oldPrefix, lengthPrefixedAddr...)
	if v := store.Get(oldKeyWithPrefix); v != nil {
		return v, true
	}

	return nil, false
}

// ReadRawContractStateWithFallback reads a single contract store entry by key with fallback
func ReadRawContractStateWithFallback(store sdk.KVStore, contractAddr []byte, key []byte) ([]byte, bool) {
	// New: 0x03 | addr | key
	newPrefix := []byte{0x03}
	newKey := append(append([]byte{}, newPrefix...), contractAddr...)
	newKey = append(newKey, key...)
	if v := store.Get(newKey); v != nil {
		return v, true
	}

	// Old: 0x05 | addr | key
	oldPrefix := []byte{0x05}
	oldKey := append(append([]byte{}, oldPrefix...), contractAddr...)
	oldKey = append(oldKey, key...)
	if v := store.Get(oldKey); v != nil {
		return v, true
	}

	// Old with length-prefixed address
	lengthPrefixedAddr := append([]byte{byte(len(contractAddr))}, contractAddr...)
	oldKeyWithPrefix := append(append([]byte{}, oldPrefix...), lengthPrefixedAddr...)
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
	newContractPrefix := append([]byte{0x03}, contractAddr...)
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
	oldContractPrefix := append([]byte{0x05}, contractAddr...)
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
	lengthPrefixedAddr := append([]byte{byte(len(contractAddr))}, contractAddr...)
	oldPrefixedContractPrefix := append([]byte{0x05}, lengthPrefixedAddr...)
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
