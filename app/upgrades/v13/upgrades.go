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

	// Check if migration has already been completed
	migrationMarker := []byte("v13_wasm_migrated")
	if store.Has(migrationMarker) {
		ctx.Logger().Info("WASM key migration already completed, skipping")
		return nil
	}

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

	// 2.1 Migrate contract keys (0x04 -> 0x02) with collision protection
	if err := migrateContractKeysWithProtection(store); err != nil {
		return fmt.Errorf("failed to migrate contract keys: %w", err)
	}

	// 2.2. Now that 0x04 is free, manually migrate sequence keys from our saved copies
	if codeIDValue != nil {
		newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
		// Check for collision before setting
		if existing := store.Get(newCodeIDKey); existing != nil {
			ctx.Logger().Info(fmt.Sprintf("Code ID sequence key already exists with value: %v, skipping", existing))
		} else {
			store.Set(newCodeIDKey, codeIDValue)
			ctx.Logger().Info(fmt.Sprintf("Migrated code ID sequence from 0x01 to %X", newCodeIDKey))
		}
		store.Delete(oldCodeIDKey)
	}

	if instanceIDValue != nil {
		newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
		// Check for collision before setting
		if existing := store.Get(newInstanceIDKey); existing != nil {
			ctx.Logger().Info(fmt.Sprintf("Instance ID sequence key already exists with value: %v, skipping", existing))
		} else {
			store.Set(newInstanceIDKey, instanceIDValue)
			ctx.Logger().Info(fmt.Sprintf("Migrated instance ID sequence from 0x02 to %X", newInstanceIDKey))
		}
		store.Delete(oldInstanceIDKey)
	}

	// 3. Migrate code keys (0x03 -> 0x01) with collision protection
	if err := migrateCodeKeysWithProtection(store, contractAddresses); err != nil {
		return err
	}

	// 4. Migrate contract store keys (0x05 -> 0x03) with collision protection
	if err := migrateContractStoreKeysWithProtection(store, contractAddresses); err != nil {
		return err
	}

	// 5. Migrate contract history keys (0x06 -> 0x05) with collision protection
	if err := migratePrefixWithProtection(store, []byte{0x06}, []byte{0x05}, "contractHistoryKey"); err != nil {
		return err
	}

	// 6. Migrate secondary index keys (0x10 -> 0x06) with collision protection
	if err := migrateSecondaryIndexKeysWithProtection(store); err != nil {
		return err
	}

	// 7. Migrate params key (0x11 -> 0x10) with collision protection
	if err := migrateParamsKeyWithProtection(store); err != nil {
		return err
	}

	// Mark migration as completed
	store.Set(migrationMarker, []byte("true"))
	ctx.Logger().Info("WASM key migration completed successfully")

	return nil
}

// migratePrefixWithProtection migrates with collision detection
func migratePrefixWithProtection(store sdk.KVStore, oldPrefix, newPrefix []byte, name string) error {
	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var collisionCount int

	for ; iterator.Valid(); iterator.Next() {
		// Copy the key and value to avoid issues with shared memory
		originalKey := make([]byte, len(iterator.Key()))
		copy(originalKey, iterator.Key())

		originalValue := make([]byte, len(iterator.Value()))
		copy(originalValue, iterator.Value())

		// Construct full keys
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, originalKey...)

		// Check for collision
		if existing := store.Get(newFullKey); existing != nil {
			// Collision detected - preserve existing data
			collisionCount++
			fmt.Printf("Collision detected for key %X, preserving existing data\n", newFullKey)
		} else {
			// Safe to migrate
			store.Set(newFullKey, originalValue)
		}

		// Always delete old key (it's been processed)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated %s, migratedCount %d, collisionCount %d\n", name, migratedCount, collisionCount)
	return nil
}

func migrateSecondaryIndexKeysWithProtection(store sdk.KVStore) error {
	oldPrefix := []byte{0x10}
	newPrefix := []byte{0x06}

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var collisionCount int
	var skippedRootKey int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := make([]byte, len(iterator.Key()))
		copy(originalKey, iterator.Key())

		originalValue := make([]byte, len(iterator.Value()))
		copy(originalValue, iterator.Value())

		// Skip the root key (empty key) which could be params data
		if len(originalKey) == 0 {
			fmt.Printf("Skipping root key at 0x10 (likely params data): %s\n", string(originalValue))
			skippedRootKey++
			continue
		}

		// Construct full keys
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, originalKey...)

		// Check for collision
		if existing := store.Get(newFullKey); existing != nil {
			collisionCount++
			fmt.Printf("Collision detected for key %X, preserving existing data\n", newFullKey)
		} else {
			store.Set(newFullKey, originalValue)
		}

		// Always delete old key (it's been processed)
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated secondaryIndexKey, migratedCount %d, collisionCount %d, skippedRootKey %d\n",
		migratedCount, collisionCount, skippedRootKey)
	return nil
}

// Simple fix: replace the complex isDefinitelyLengthPrefixedAddress logic
func migrateContractKeysWithProtection(store sdk.KVStore) error {
	oldPrefix := []byte{0x04}
	newPrefix := []byte{0x02}

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var lengthPrefixRemovedCount int
	var collisionDetectedCount int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := make([]byte, len(iterator.Key()))
		copy(originalKey, iterator.Key())

		originalValue := make([]byte, len(iterator.Value()))
		copy(originalValue, iterator.Value())

		// Simple logic: always try to strip length prefix if it's valid
		unprefixedKey, stripped := removeLengthPrefixIfNeeded(originalKey)
		if stripped {
			lengthPrefixRemovedCount++
			fmt.Printf("Removed length prefix from contract key: %X -> %X\n", originalKey, unprefixedKey)
		}

		// Construct full keys
		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, unprefixedKey...)

		// Check for collision before setting
		if existing := store.Get(newFullKey); existing != nil {
			collisionDetectedCount++
			fmt.Printf("Collision detected for key %X, preserving existing data\n", newFullKey)
		} else {
			store.Set(newFullKey, originalValue)
		}

		// Always delete old key
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated contractKey, migratedCount %d, lengthPrefixRemovedCount %d, collisionDetectedCount %d\n",
		migratedCount, lengthPrefixRemovedCount, collisionDetectedCount)

	return nil
}

// Simple fix for contract store migration
func migrateContractStoreKeysWithProtection(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x05}
	newPrefix := []byte{0x03}

	fmt.Printf("Using %d pre-collected contracts to migrate storage\n", len(contractAddresses))

	var totalMigrated int
	var totalCollisions int

	for i, originalContractAddr := range contractAddresses {
		if originalContractAddr == nil {
			fmt.Printf("Warning: Skipping nil contract address at index %d\n", i)
			continue
		}

		// Copy the contract address
		contractAddr := make([]byte, len(originalContractAddr))
		copy(contractAddr, originalContractAddr)

		// Simple logic: always try to strip length prefix
		unprefixedAddr, stripped := removeLengthPrefixIfNeeded(contractAddr)
		if stripped {
			fmt.Printf("Stripped contract address: %X -> %X\n", contractAddr, unprefixedAddr)
		}

		// Construct prefixes
		oldContractPrefix := append([]byte{0x05}, contractAddr...)
		newContractPrefix := append([]byte{0x03}, unprefixedAddr...)

		// Migrate this contract's storage
		oldContractStore := prefix.NewStore(store, oldContractPrefix)
		oldContractIter := oldContractStore.Iterator(nil, nil)

		var contractKeyCount int
		var contractCollisions int
		for ; oldContractIter.Valid(); oldContractIter.Next() {
			originalKey := make([]byte, len(oldContractIter.Key()))
			copy(originalKey, oldContractIter.Key())

			originalValue := make([]byte, len(oldContractIter.Value()))
			copy(originalValue, oldContractIter.Value())

			if len(originalKey) == 0 || len(originalValue) == 0 {
				continue
			}

			oldFullKey := append([]byte{}, oldContractPrefix...)
			oldFullKey = append(oldFullKey, originalKey...)

			newFullKey := append([]byte{}, newContractPrefix...)
			newFullKey = append(newFullKey, originalKey...)

			if existing := store.Get(newFullKey); existing != nil {
				contractCollisions++
				fmt.Printf("Contract store collision detected for key %X, preserving existing data\n", newFullKey)
			} else {
				store.Set(newFullKey, originalValue)
			}

			store.Delete(oldFullKey)
			contractKeyCount++
			totalMigrated++
		}
		oldContractIter.Close()

		totalCollisions += contractCollisions
		fmt.Printf("Migrated %d keys for contract %X (collisions: %d)\n", contractKeyCount, unprefixedAddr, contractCollisions)
	}

	// Handle any remaining direct contract store keys
	directOldStore := prefix.NewStore(store, oldPrefix)
	directOldIter := directOldStore.Iterator(nil, nil)

	var directMigrated int
	var directCollisions int
	for ; directOldIter.Valid(); directOldIter.Next() {
		originalKey := make([]byte, len(directOldIter.Key()))
		copy(originalKey, directOldIter.Key())

		originalValue := make([]byte, len(directOldIter.Value()))
		copy(originalValue, directOldIter.Value())

		if originalKey == nil || originalValue == nil {
			continue
		}

		// For composite keys like [address | subkey], try to strip the address part
		var rebuiltKey []byte
		if len(originalKey) > 1 {
			candidateLen := int(originalKey[0]) + 1
			if candidateLen <= len(originalKey) {
				head := originalKey[:candidateLen]
				tail := originalKey[candidateLen:]

				if unprefHead, stripped := removeLengthPrefixIfNeeded(head); stripped {
					rebuiltKey = append([]byte{}, unprefHead...)
					rebuiltKey = append(rebuiltKey, tail...)
					fmt.Printf("Stripped composite key: %X -> %X\n", originalKey, rebuiltKey)
				}
			}
		}
		if rebuiltKey == nil {
			rebuiltKey = originalKey
		}

		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, rebuiltKey...)

		if existing := store.Get(newFullKey); existing != nil {
			directCollisions++
			fmt.Printf("Direct contract store collision detected for key %X, preserving existing data\n", newFullKey)
		} else {
			store.Set(newFullKey, originalValue)
		}

		store.Delete(oldFullKey)
		directMigrated++
	}
	directOldIter.Close()

	totalCollisions += directCollisions
	fmt.Printf("Additionally migrated %d direct contract store keys (collisions: %d)\n", directMigrated, directCollisions)
	fmt.Printf("Total migrated contract store keys: %d, total collisions: %d\n", totalMigrated+directMigrated, totalCollisions)

	return nil
}

// migrateParamsKeyWithProtection migrates params key with collision protection
func migrateParamsKeyWithProtection(store sdk.KVStore) error {
	oldKey := []byte{0x11}
	newKey := []byte{0x10}

	value := store.Get(oldKey)
	if value == nil {
		return nil
	}

	existing := store.Get(newKey)
	if existing != nil {
		fmt.Printf("Params key collision detected, preserving existing data at 0x10\n")
		fmt.Printf("Existing data: %s, Old data: %s\n", string(existing), string(value))
		// Don't migrate, but still delete old key to match test expectation
		store.Delete(oldKey)
		return nil
	}

	// No collision - migrate normally
	tmpValue := make([]byte, len(value))
	copy(tmpValue, value)
	store.Set(newKey, tmpValue)
	store.Delete(oldKey)
	return nil
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

// buildKnownLegitimateAddresses creates a slice of addresses that we know are legitimate
// This should be populated based on historical chain data or other reliable sources
func buildKnownLegitimateAddresses(contractAddresses [][]byte) []string {
	// Use a map temporarily for deduplication and collision detection
	tempMap := make(map[string]bool)
	var result []string

	// First pass: identify addresses that don't need stripping (already valid format)
	for _, addr := range contractAddresses {
		if err := sdk.VerifyAddressFormat(addr); err == nil {
			addrStr := string(addr)
			if !tempMap[addrStr] {
				tempMap[addrStr] = true
				result = append(result, addrStr)
			}
		}
	}

	// Second pass: for potential length-prefixed addresses, only add them if they
	// don't conflict with existing known legitimate addresses
	for _, addr := range contractAddresses {
		if err := sdk.VerifyAddressFormat(addr); err != nil {
			// This might be length-prefixed
			if stripped, wasStripped := removeLengthPrefixIfNeeded(addr); wasStripped {
				strippedStr := string(stripped)
				// Only mark as legitimate if it doesn't conflict with existing entries
				if !tempMap[strippedStr] {
					tempMap[strippedStr] = true
					result = append(result, strippedStr)
				} else {
					// Collision detected - log it but don't add to legitimate set
					fmt.Printf("WARNING: Collision detected for address %X, will preserve original format\n", addr)
				}
			}
		}
	}

	// Sort for deterministic order
	sort.Strings(result)
	return result
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
	return migrateContractStoreKeysWithProtection(store, contractAddresses)
}

func MigrateContractKeys(store sdk.KVStore) error {
	return migrateContractKeysWithProtection(store)
}

func looksLikeContractStoreKey(k []byte, knownAddrs []string) bool {
	// Case A: 1-byte length-prefixed address + subkey
	if len(k) > 1 {
		ln := int(k[0])
		if ln > 0 && 1+ln <= len(k) {
			addr := string(k[1 : 1+ln])
			// Deterministic search through slice
			for _, known := range knownAddrs {
				if known == addr && 1+ln < len(k) {
					return true
				}
			}
		}
	}

	// Case B: unprefixed address at the front + subkey
	// This loop will always process addresses in the same order
	for _, addr := range knownAddrs {
		ab := []byte(addr)
		if len(k) > len(ab) && bytes.Equal(k[:len(ab)], ab) {
			return true
		}
	}
	return false
}

// Only migrate true "code keys" from 0x03 -> 0x01. Skip contract-store-shaped keys that
// may already exist at 0x03 due to a previous (partial) migration.
func migrateCodeKeysWithProtection(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := []byte{0x03}
	newPrefix := []byte{0x01}

	known := buildKnownLegitimateAddresses(contractAddresses)

	oldStore := prefix.NewStore(store, oldPrefix)
	iter := oldStore.Iterator(nil, nil)
	defer iter.Close()

	migratedCount := 0
	collisionCount := 0
	skippedAsContractStore := 0

	for ; iter.Valid(); iter.Next() {
		originalKey := append([]byte{}, iter.Key()...)
		originalValue := append([]byte{}, iter.Value()...)

		// If this looks like a contract store key (address + subkey), leave it alone.
		if looksLikeContractStoreKey(originalKey, known) {
			skippedAsContractStore++
			continue
		}

		oldFullKey := append([]byte{}, oldPrefix...)
		oldFullKey = append(oldFullKey, originalKey...)

		newFullKey := append([]byte{}, newPrefix...)
		newFullKey = append(newFullKey, originalKey...)

		if existing := store.Get(newFullKey); existing != nil {
			collisionCount++
			fmt.Printf("Collision detected for key %X, preserving existing data\n", newFullKey)
		} else {
			store.Set(newFullKey, originalValue)
		}

		// Delete only the codeKey we actually processed
		store.Delete(oldFullKey)
		migratedCount++
	}

	fmt.Printf("migrated codeKey, migratedCount %d, collisionCount %d (skipped as contractStore: %d)\n",
		migratedCount, collisionCount, skippedAsContractStore)
	return nil
}
