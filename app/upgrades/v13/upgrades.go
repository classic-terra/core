//nolint:revive
package v13

import (
	"bytes"
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
	// This needs to happen before we write to 0x04 with sequence keys
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
// Accepts only exact address blobs (or a 1-byte length prefix + address).
// Never run this on arbitrary composite keys.
func removeLengthPrefixIfNeeded(b []byte) (out []byte, stripped bool) {
	// Already a valid address for this chain? Keep as-is.
	if err := sdk.VerifyAddressFormat(b); err == nil {
		return bytes.Clone(b), false
	}
	// Looks like a [len|payload] and payload is a valid address?
	if len(b) > 1 && int(b[0]) == len(b)-1 {
		payload := b[1:]
		if err := sdk.VerifyAddressFormat(payload); err == nil {
			return bytes.Clone(payload), true
		}
	}
	// Not an address (or composite) -> don't touch
	return bytes.Clone(b), false
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
		unprefixedKey, stripped := removeLengthPrefixIfNeeded(originalKey)

		// Track if we removed a length prefix
		if stripped {
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
		unprefixedAddr, _ := removeLengthPrefixIfNeeded(contractAddr)

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

			// Skip nil keys or values
			if originalKey == nil || originalValue == nil {
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

		// Skip nil keys or values
		if originalKey == nil || originalValue == nil {
			continue
		}

		// The structure here is [address_or_len_prefixed_address | subkey...]
		// We must ONLY attempt to strip a 1-byte length prefix from the address portion,
		// not from the entire composite key. Determine the address prefix to transform.
		var rebuiltKey []byte
		if len(originalKey) > 1 {
			// Try interpret the first segment as a potential [len|payload]
			// candidateLen covers the length-prefix byte plus payload
			candidateLen := int(originalKey[0]) + 1
			if candidateLen <= len(originalKey) {
				// Evaluate only the candidate head
				head := originalKey[:candidateLen]
				tail := originalKey[candidateLen:]
				if unprefHead, stripped := removeLengthPrefixIfNeeded(head); stripped {
					// Rebuild as [unprefixed_address | tail]
					rebuiltKey = append([]byte{}, unprefHead...)
					rebuiltKey = append(rebuiltKey, tail...)
				}
			}
		}
		if rebuiltKey == nil {
			// Either not length-prefixed or not a valid address head; keep as-is
			rebuiltKey = originalKey
		}

		// Construct full keys - create new slices to avoid modifying the original prefixes
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, rebuiltKey...)

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
		unprefixedAddr, stripped := removeLengthPrefixIfNeeded(addr)
		if stripped {
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
func RemoveLengthPrefixIfNeeded(bz []byte) ([]byte, bool) {
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
