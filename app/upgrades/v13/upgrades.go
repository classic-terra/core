//nolint:revive
package v13

import (
	"bytes"
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cometbft/cometbft/libs/log"
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

func migrateWasmKeys(ctx sdk.Context, _ wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	store := ctx.KVStore(wasmStoreKey)

	ctx.Logger().Info("Starting WASM key migration from forked to original format")

	// Check if migration was already completed
	migrationMarker := []byte("v13_wasm_migrated")
	if store.Has(migrationMarker) {
		ctx.Logger().Info("WASM migration already completed, skipping")
		return nil
	}

	// Validate that destination prefixes are safe to migrate to
	if err := validateDestinationPrefixes(store, ctx.Logger()); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	// First, collect all contract addresses before any migration
	contractAddresses := collectContractAddresses(store, ctx.Logger())
	ctx.Logger().Info(fmt.Sprintf("Found %d contracts for migration", len(contractAddresses)))

	// Add validation of collected addresses
	if len(contractAddresses) == 0 {
		ctx.Logger().Info("No contracts found for migration, this might indicate an issue")
	}

	// 1. Safely migrate sequence keys with validation
	if err := migrateSequenceKeysAtomically(store, ctx.Logger()); err != nil {
		return fmt.Errorf("failed to migrate sequence keys: %w", err)
	}

	// 2.1 Migrate contract keys (0x04 -> 0x02)
	// This needs to happen before we write to 0x02 with sequence keys
	if err := migrateContractKeys(store); err != nil {
		return fmt.Errorf("failed to migrate contract keys: %w", err)
	}

	// Sequence keys are now migrated atomically by migrateSequenceKeysAtomically

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

	// Set migration marker to indicate completion
	store.Set(migrationMarker, []byte("true"))
	ctx.Logger().Info("WASM key migration completed successfully and marked as done")

	return nil
}

// migrateSequenceKeysAtomically safely migrates sequence keys with proper validation
func migrateSequenceKeysAtomically(store sdk.KVStore, logger log.Logger) error {
	// Source keys
	oldCodeIDKey := []byte{0x01}
	oldInstanceIDKey := []byte{0x02}

	// Destination keys
	newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
	newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)

	// Check if destination keys already exist (retry scenario)
	if store.Has(newCodeIDKey) && store.Has(newInstanceIDKey) {
		logger.Info("Sequence keys already migrated, validating existing values")

		// Validate that existing migrated values make sense
		newCodeIDValue := store.Get(newCodeIDKey)
		newInstanceIDValue := store.Get(newInstanceIDKey)

		if len(newCodeIDValue) == 8 && len(newInstanceIDValue) == 8 {
			logger.Info("Existing sequence key values are valid, migration already complete")

			// Clean up old keys if they still exist
			if store.Has(oldCodeIDKey) {
				store.Delete(oldCodeIDKey)
				logger.Info("Cleaned up old code ID sequence key")
			}
			if store.Has(oldInstanceIDKey) {
				store.Delete(oldInstanceIDKey)
				logger.Info("Cleaned up old instance ID sequence key")
			}

			return nil
		} else {
			return fmt.Errorf("existing sequence values are invalid: codeID=%X, instanceID=%X",
				newCodeIDValue, newInstanceIDValue)
		}
	}

	// Get old values
	oldCodeIDValue := store.Get(oldCodeIDKey)
	oldInstanceIDValue := store.Get(oldInstanceIDKey)

	// Validate old values exist and are properly formatted
	if oldCodeIDValue == nil {
		logger.Info("No code ID sequence found at old location")
		oldCodeIDValue = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Default to 0
	} else if len(oldCodeIDValue) != 8 {
		return fmt.Errorf("invalid code ID sequence format: %X", oldCodeIDValue)
	}

	if oldInstanceIDValue == nil {
		logger.Info("No instance ID sequence found at old location")
		oldInstanceIDValue = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Default to 0
	} else if len(oldInstanceIDValue) != 8 {
		return fmt.Errorf("invalid instance ID sequence format: %X", oldInstanceIDValue)
	}

	logger.Info(fmt.Sprintf("Migrating code ID sequence: %X", oldCodeIDValue))
	logger.Info(fmt.Sprintf("Migrating instance ID sequence: %X", oldInstanceIDValue))

	// Copy values to avoid memory aliasing
	codeIDValue := make([]byte, len(oldCodeIDValue))
	copy(codeIDValue, oldCodeIDValue)

	instanceIDValue := make([]byte, len(oldInstanceIDValue))
	copy(instanceIDValue, oldInstanceIDValue)

	// Write new values first (fail-safe)
	store.Set(newCodeIDKey, codeIDValue)
	store.Set(newInstanceIDKey, instanceIDValue)

	// Verify the write was successful
	verifyCodeID := store.Get(newCodeIDKey)
	verifyInstanceID := store.Get(newInstanceIDKey)

	if !bytes.Equal(verifyCodeID, codeIDValue) {
		return fmt.Errorf("failed to write code ID sequence: expected %X, got %X", codeIDValue, verifyCodeID)
	}

	if !bytes.Equal(verifyInstanceID, instanceIDValue) {
		return fmt.Errorf("failed to write instance ID sequence: expected %X, got %X", instanceIDValue, verifyInstanceID)
	}

	// Only delete old keys after verifying new ones are written correctly
	store.Delete(oldCodeIDKey)
	store.Delete(oldInstanceIDKey)

	logger.Info(fmt.Sprintf("Successfully migrated code ID sequence from %X to %X", oldCodeIDKey, newCodeIDKey))
	logger.Info(fmt.Sprintf("Successfully migrated instance ID sequence from %X to %X", oldInstanceIDKey, newInstanceIDKey))

	return nil
}

// migrateCodeKeys migrates code keys from 0x03 to 0x01
func migrateCodeKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x03}
	newPrefix := []byte{0x01}
	return migratePrefix(store, oldPrefix, newPrefix, "codeKey")
}

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

func migrateContractKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x04}
	newPrefix := []byte{0x02}

	oldStore := prefix.NewStore(store, oldPrefix)
	iter := oldStore.Iterator(nil, nil)
	defer iter.Close()

	type kv struct {
		oldKey []byte
		newKey []byte
		value  []byte
	}
	var toMigrate []kv

	for ; iter.Valid(); iter.Next() {
		originalKey := append([]byte{}, iter.Key()...)
		originalValue := append([]byte{}, iter.Value()...)

		// Only migrate if looks like contract address
		if !(len(originalKey) == 20 || (len(originalKey) == 21 && int(originalKey[0]) == 20)) {
			continue
		}

		unprefixedKey, _ := stripLenPrefixAddrOnly(originalKey)

		oldFullKey := append(append([]byte{}, oldPrefix...), originalKey...)
		newFullKey := append(append([]byte{}, newPrefix...), unprefixedKey...)

		toMigrate = append(toMigrate, kv{oldKey: oldFullKey, newKey: newFullKey, value: originalValue})
	}

	// Now write & delete
	for _, entry := range toMigrate {
		if store.Has(entry.newKey) && !bytes.Equal(store.Get(entry.newKey), entry.value) {
			// Collision: keep existing, just delete old
			store.Delete(entry.oldKey)
			continue
		}
		store.Set(entry.newKey, entry.value)
		store.Delete(entry.oldKey)
	}

	return nil
}

// migrateContractStoreKeys migrates contract store keys from 0x05 to 0x03
// and removes length prefixes from addresses in the keys
func migrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x05}
	newPrefix := []byte{0x03}

	fmt.Printf("Using %d pre-collected contracts to migrate storage\n", len(contractAddresses))

	// Now migrate each contract's storage
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
		unprefixedAddr, _ := stripLenPrefixAddrOnly(contractAddr)

		// Construct the old and new prefixes for this specific contract
		oldContractPrefix := append([]byte{0x05}, contractAddr...)   // Original key with potential length prefix
		newContractPrefix := append([]byte{0x03}, unprefixedAddr...) // New key without length prefix

		// Create iterator for this contract's storage
		oldContractStore := prefix.NewStore(store, oldContractPrefix)
		oldContractIter := oldContractStore.Iterator(nil, nil)

		var contractKeyCount int
		// inside migrateContractStoreKeys
		for ; oldContractIter.Valid(); oldContractIter.Next() {
			originalKey := append([]byte{}, oldContractIter.Key()...)
			originalValue := append([]byte{}, oldContractIter.Value()...)

			oldFullKey := append(append([]byte{}, oldContractPrefix...), originalKey...)
			newFullKey := append(append([]byte{}, newContractPrefix...), originalKey...)

			if len(originalKey) > 0 && len(originalValue) > 0 {
				if store.Has(newFullKey) && !bytes.Equal(store.Get(newFullKey), originalValue) {
					return fmt.Errorf("collision on %X", newFullKey)
				}
				store.Set(newFullKey, originalValue)
			}

			// delete last to avoid losing data
			store.Delete(oldFullKey)
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

		// Construct full keys - create new slices to avoid modifying the original prefixes
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		// Always delete the old key to avoid stray entries
		store.Delete(oldFullKey)

		// Only create new entry if we have non-empty key and value
		if len(originalKey) > 0 && len(originalValue) > 0 {
			// For direct contract store keys, we should NOT strip length prefixes
			// as these are composite keys [contractAddr + storageKey], not pure addresses
			newFullKey := append([]byte{}, newPrefix...)
			newFullKey = append(newFullKey, originalKey...)

			// Add collision guard before setting
			if store.Has(newFullKey) && !bytes.Equal(store.Get(newFullKey), originalValue) {
				return fmt.Errorf("refusing to overwrite existing key %X during direct contract store migration", newFullKey)
			}

			store.Set(newFullKey, originalValue)
		}

		directMigrated++
	}
	directOldIter.Close()

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

		// Collision guard: if destination already has a different value, preserve it
		if newStore.Has(originalKey) {
			existing := newStore.Get(originalKey)
			if !bytes.Equal(existing, originalValue) {
				// Preserve destination, skip migrating this key (do not delete old)
				continue
			}
		}

		newStore.Set(originalKey, originalValue)
		oldStore.Delete(originalKey)
		migratedCount++
	}

	fmt.Printf("migrated name %s, migratedCount %d\n", name, migratedCount)

	return nil
}

// collectContractAddresses gets all contract addresses before any migration
func collectContractAddresses(store sdk.KVStore, logger log.Logger) [][]byte {
	// Contract addresses are stored with prefix 0x04 before migration
	contractInfoPrefix := []byte{0x04}
	contractInfoStore := prefix.NewStore(store, contractInfoPrefix)
	contractInfoIter := contractInfoStore.Iterator(nil, nil)
	defer contractInfoIter.Close()

	var contractAddresses [][]byte
	for ; contractInfoIter.Valid(); contractInfoIter.Next() {
		// The key is the contract address (potentially with length prefix)
		addr := contractInfoIter.Key()

		// Only collect entries that look like contract addresses
		// Valid forms:
		// - 20-byte raw address
		// - 21-byte length-prefixed with first byte == 20
		if len(addr) == 20 || (len(addr) == 21 && int(addr[0]) == 20) {
			contractAddresses = append(contractAddresses, addr)

			// Log each contract address for debugging
			logger.Info(fmt.Sprintf("Found contract address: %X (length: %d)\n", addr, len(addr)))
		}
	}

	return contractAddresses
}

// validateDestinationPrefixes ensures destination prefixes are safe for migration
func validateDestinationPrefixes(store sdk.KVStore, logger log.Logger) error {
	destinationPrefixes := map[string][]byte{
		"sequence_keys":    {0x04}, // lastCodeId, lastContractId will go here
		"code_keys":        {0x01}, // code data migrates here
		"contract_keys":    {0x02}, // contract info migrates here
		"contract_store":   {0x03}, // contract storage migrates here
		"contract_history": {0x05}, // contract history migrates here
		"secondary_index":  {0x06}, // secondary index migrates here
		"params":           {0x10}, // params migrate here
	}

	for name, prefixBytes := range destinationPrefixes {
		// For sequence keys (0x04), we expect it to be empty or contain only sequence data
		if bytes.Equal(prefixBytes, []byte{0x04}) {
			if err := validateSequenceKeyDestination(store, prefixBytes, logger); err != nil {
				return fmt.Errorf("sequence key destination validation failed: %w", err)
			}
			continue
		}

		// For other prefixes, check if they contain unexpected data
		prefixStore := prefix.NewStore(store, prefixBytes)
		iter := prefixStore.Iterator(nil, nil)
		count := 0
		for ; iter.Valid(); iter.Next() {
			count++
			if count > 10 { // Don't log too many, just sample
				break
			}
			logger.Info(fmt.Sprintf("Found existing data in destination prefix %s: key %X", name, iter.Key()))
		}
		iter.Close()

		if count > 0 {
			logger.Info(fmt.Sprintf("Destination prefix %s contains %d+ existing entries - migration will proceed with collision guards", name, count))
		}
	}

	return nil
}

// validateSequenceKeyDestination checks if the sequence key destination is safe
func validateSequenceKeyDestination(store sdk.KVStore, prefixBytes []byte, logger log.Logger) error {
	prefixStore := prefix.NewStore(store, prefixBytes)

	// Check for existing sequence keys
	lastCodeIdKey := []byte("lastCodeId")
	lastContractIdKey := []byte("lastContractId")

	if prefixStore.Has(lastCodeIdKey) {
		existingValue := prefixStore.Get(lastCodeIdKey)
		logger.Info(fmt.Sprintf("Found existing lastCodeId in destination: %X", existingValue))
	}

	if prefixStore.Has(lastContractIdKey) {
		existingValue := prefixStore.Get(lastContractIdKey)
		logger.Info(fmt.Sprintf("Found existing lastContractId in destination: %X", existingValue))
	}

	return nil
}

// MigrateWasmKeys Exported for testing
func MigrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	return migrateWasmKeys(ctx, wasmKeeper, wasmStoreKey)
}

// CollectContractAddresses Exported for testing
func CollectContractAddresses(store sdk.KVStore, logger log.Logger) [][]byte {
	return collectContractAddresses(store, logger)
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
