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

// Helper for saving sequence keys
type sequenceKeys struct {
	codeIDValue     []byte
	instanceIDValue []byte
}

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

	// Collect contract addresses before migration
	contractAddresses := collectContractAddresses(store)
	ctx.Logger().Info(fmt.Sprintf("Found %d contracts for migration", len(contractAddresses)))

	if len(contractAddresses) == 0 {
		ctx.Logger().Info("No contracts found for migration, this might indicate an issue")
	}

	// Save sequence keys for later migration
	sequenceKeys := saveSequenceKeys(store)

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

	if err := migratePrefix(store, LegacyPrefixes.ContractCodeHistoryElementPrefix, wasmtypes.ContractCodeHistoryElementPrefix, "contractHistoryKey"); err != nil {
		return fmt.Errorf("failed to migrate contract history keys: %w", err)
	}

	if err := migrateSecondaryIndexKeys(store); err != nil {
		return fmt.Errorf("failed to migrate secondary index keys: %w", err)
	}

	if err := migrateParamsKey(store); err != nil {
		return fmt.Errorf("failed to migrate params key: %w", err)
	}

	ctx.Logger().Info("WASM key migration completed successfully")

	return nil
}

// migrateContractKeys move contracts key from 0x04 -> 0x02
func migrateContractKeys(store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractKeyPrefix
	newPrefix := wasmtypes.ContractKeyPrefix

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	var lengthPrefixRemovedCount int

	for ; iterator.Valid(); iterator.Next() {
		originalKey := copyBytes(iterator.Key())
		originalValue := copyBytes(iterator.Value())

		unprefixedKey, stripped := removeLengthPrefixIfNeeded(originalKey)
		fmt.Printf("Processing contract key: %X\n", originalKey)
		fmt.Printf("  - Unprefixed key: %X (length: %d), stripped: %v\n", unprefixedKey, len(unprefixedKey), stripped)
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

// Save sequence keys to a variable for later migration
func saveSequenceKeys(store sdk.KVStore) sequenceKeys {
	oldCodeIDKey := LegacyPrefixes.KeySequenceCodeID
	oldInstanceIDKey := LegacyPrefixes.KeySequenceInstanceID

	seq := sequenceKeys{}
	if v := store.Get(oldCodeIDKey); v != nil {
		seq.codeIDValue = append([]byte{}, v...) // copy
	}
	if v := store.Get(oldInstanceIDKey); v != nil {
		seq.instanceIDValue = append([]byte{}, v...) // copy
	}
	return seq
}

// migrateSequenceKeys migrates the saved sequence keys from old to new prefix
// 0x01 → 0x04/"lastCodeId"
// 0x02 → 0x04/"lastContractId"
func migrateSequenceKeys(store sdk.KVStore, seq sequenceKeys, ctx sdk.Context) error {
	if seq.codeIDValue != nil {
		newKey := wasmtypes.KeySequenceCodeID
		if !store.Has(newKey) {
			store.Set(newKey, seq.codeIDValue)
			ctx.Logger().Info(fmt.Sprintf("Migrated code ID sequence to %X", newKey))
		}
		store.Delete(LegacyPrefixes.KeySequenceCodeID) // delete old only after new exists
	}

	if seq.instanceIDValue != nil {
		newKey := wasmtypes.KeySequenceInstanceID
		if !store.Has(newKey) {
			store.Set(newKey, seq.instanceIDValue)
			ctx.Logger().Info(fmt.Sprintf("Migrated instance ID sequence to %X", newKey))
		}

		// Don't delete here because 0x02 is intended for contract keys
		// store.Delete(LegacyPrefixes.KeySequenceInstanceID)
	}

	return nil
}

func migrateSecondaryIndexKeys(store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractByCodeIDAndCreatedSecondaryIndexPrefix
	newPrefix := wasmtypes.ContractByCodeIDAndCreatedSecondaryIndexPrefix

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

// migrateContractStoreKeys migrates contract store keys from old to new prefix
func migrateContractStoreKeys(store sdk.KVStore, contractAddresses [][]byte) error {
	oldPrefix := LegacyPrefixes.ContractStorePrefix
	newPrefix := wasmtypes.ContractStorePrefix

	fmt.Printf("Using %d pre-collected contracts to migrate storage\n", len(contractAddresses))

	totalMigrated := migrateContractSpecificKeys(store, oldPrefix, newPrefix, contractAddresses)
	directMigrated := migrateDirectContractStoreKeys(store, oldPrefix, newPrefix)

	fmt.Printf("Total migrated contract store keys: %d\n", totalMigrated+directMigrated)
	return nil
}

// migrateContractSpecificKeys migrates contract-specific keys from old to new prefix
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

		oldContractPrefix := append(oldPrefix, contractAddr...)   // nolint:gocritic
		newContractPrefix := append(newPrefix, unprefixedAddr...) // nolint:gocritic

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
	oldKey := LegacyPrefixes.ParamsKey
	newKey := wasmtypes.ParamsKey

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
	oldPrefix := LegacyPrefixes.CodeKeyPrefix
	newPrefix := wasmtypes.CodeKeyPrefix

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
	// If not length-prefixed, check if already a valid address
	if err := sdk.VerifyAddressFormat(b); err == nil {
		return bytes.Clone(b), false
	}
	// Check for length prefix pattern
	if len(b) > 1 && int(b[0]) == len(b)-1 {
		payload := b[1:]
		// Verify the payload is a valid address
		if err := sdk.VerifyAddressFormat(payload); err == nil {
			return bytes.Clone(payload), true
		}
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
	contractInfoPrefix := LegacyPrefixes.ContractKeyPrefix
	contractInfoStore := prefix.NewStore(store, contractInfoPrefix)
	contractInfoIter := contractInfoStore.Iterator(nil, nil)
	defer contractInfoIter.Close()

	var contractAddresses [][]byte
	for ; contractInfoIter.Valid(); contractInfoIter.Next() {
		addr := contractInfoIter.Key()
		contractAddresses = append(contractAddresses, addr)
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

// Generic prefix migration from an old prefix to a new prefix
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
