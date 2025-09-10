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

	// Save sequence keys for later migration
	sequenceKeys := saveSequenceKeys(ctx, store)

	// Perform migrations in order
	if err := migrateContractKeys(ctx, store); err != nil {
		return fmt.Errorf("failed to migrate contract keys: %w", err)
	}

	if err := migrateSequenceKeys(ctx, store, sequenceKeys); err != nil {
		return fmt.Errorf("failed to migrate sequence keys: %w", err)
	}

	if err := migrateCodeKeys(ctx, store); err != nil {
		return fmt.Errorf("failed to migrate code keys: %w", err)
	}

	if err := migrateContractStoreKeys(ctx, store); err != nil {
		return fmt.Errorf("failed to migrate contract store keys: %w", err)
	}

	if err := migrateContractHistoryKey(ctx, store); err != nil {
		return fmt.Errorf("failed to migrate contract history keys: %w", err)
	}

	if err := migrateSecondaryIndexKeys(ctx, store); err != nil {
		return fmt.Errorf("failed to migrate secondary index keys: %w", err)
	}

	if err := migrateParamsKey(store); err != nil {
		return fmt.Errorf("failed to migrate params key: %w", err)
	}

	ctx.Logger().Info("WASM key migration completed successfully")

	return nil
}

// migrateContractKeys move contracts key from 0x04 -> 0x02
// 0x04/"length-prefixed-contract-addr" -> 0x02/"raw-contract-addr"
func migrateContractKeys(ctx sdk.Context, store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractKeyPrefix
	newPrefix := wasmtypes.ContractKeyPrefix

	oldStore := prefix.NewStore(store, oldPrefix)
	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int
	for ; iterator.Valid(); iterator.Next() {
		originalKey := copyBytes(iterator.Key())
		originalValue := copyBytes(iterator.Value())
		unprefixedKey, _ := removeLengthPrefixIfNeeded(originalKey)

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, unprefixedKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	ctx.Logger().Info(fmt.Sprintf("migrated contractKey, migratedCount %d\n",
		migratedCount))

	return nil
}

// saveSequenceKeys save sequence keys temporarily, then delete from store for later migration
func saveSequenceKeys(ctx sdk.Context, store sdk.KVStore) sequenceKeys {
	oldCodeIDKey := LegacyPrefixes.KeySequenceCodeID
	oldInstanceIDKey := LegacyPrefixes.KeySequenceInstanceID

	seq := sequenceKeys{}
	if v := store.Get(oldCodeIDKey); v != nil {
		seq.codeIDValue = append([]byte{}, v...) // copy
		ctx.Logger().Info(fmt.Sprintf("Saved code ID sequence: %X", oldCodeIDKey))
		// Delete old key after copying
		store.Delete(oldCodeIDKey)
	}
	if v := store.Get(oldInstanceIDKey); v != nil {
		seq.instanceIDValue = append([]byte{}, v...) // copy
		ctx.Logger().Info(fmt.Sprintf("Saved instance ID sequence: %X", oldInstanceIDKey))
		// Delete old key after copying
		store.Delete(oldInstanceIDKey)
	}
	return seq
}

// migrateSequenceKeys migrates the saved sequence keys from old to new prefix
// 0x01 → 0x04/"lastCodeId"
// 0x02 → 0x04/"lastContractId"
func migrateSequenceKeys(ctx sdk.Context, store sdk.KVStore, seq sequenceKeys) error {
	newKey := wasmtypes.KeySequenceCodeID
	if !store.Has(newKey) {
		if seq.codeIDValue == nil {
			seq.codeIDValue = []byte{0, 0, 0, 0, 0, 0, 0, 0} // default to zero if not found
		}
		store.Set(newKey, seq.codeIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated code ID sequence to %X", newKey))
	}

	newKey = wasmtypes.KeySequenceInstanceID
	if !store.Has(newKey) {
		if seq.instanceIDValue == nil {
			seq.instanceIDValue = []byte{0, 0, 0, 0, 0, 0, 0, 0} // default to zero if not found
		}
		store.Set(newKey, seq.instanceIDValue)
		ctx.Logger().Info(fmt.Sprintf("Migrated instance ID sequence to %X", newKey))
	}

	return nil
}

// migrateContractHistoryKey migrates contract history keys from 0x06 -> 0x04
func migrateContractHistoryKey(ctx sdk.Context, store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractCodeHistoryElementPrefix
	newPrefix := wasmtypes.ContractCodeHistoryElementPrefix

	if err := migratePrefix(ctx, store, oldPrefix, newPrefix, "contractHistoryKey"); err != nil {
		return fmt.Errorf("failed to migrate contract history keys: %w", err)
	}
	return nil
}

// migrateSecondaryIndexKeys migrates secondary index keys from 0x10 -> 0x06
func migrateSecondaryIndexKeys(ctx sdk.Context, store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractByCodeIDAndCreatedSecondaryIndexPrefix
	newPrefix := wasmtypes.ContractByCodeIDAndCreatedSecondaryIndexPrefix

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

	ctx.Logger().Info(fmt.Sprintf("migrated secondaryIndexKey, migratedCount %d\n",
		migratedCount))
	return nil
}

// migrateContractStoreKeys migrates contract store keys from old to new prefix
func migrateContractStoreKeys(ctx sdk.Context, store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.ContractStorePrefix
	newPrefix := wasmtypes.ContractStorePrefix

	directMigrated := migrateDirectContractStoreKeys(ctx, store, oldPrefix, newPrefix)

	ctx.Logger().Info(fmt.Sprintf("Total migrated contract store keys: %d\n", directMigrated))
	return nil
}

func migrateDirectContractStoreKeys(ctx sdk.Context, store sdk.KVStore, oldPrefix, newPrefix []byte) int {
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

		rebuiltKey := rebuildCompositeKey(ctx, originalKey)
		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, rebuiltKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		directMigrated++
	}

	ctx.Logger().Info(fmt.Sprintf("Additionally migrated %d direct contract store keys\n", directMigrated))
	return directMigrated
}

func rebuildCompositeKey(ctx sdk.Context, originalKey []byte) []byte {
	if len(originalKey) > 1 {
		candidateLen := int(originalKey[0]) + 1
		if candidateLen <= len(originalKey) {
			head := originalKey[:candidateLen]
			tail := originalKey[candidateLen:]

			if unprefHead, stripped := removeLengthPrefixIfNeeded(head); stripped {
				rebuiltKey := append([]byte{}, unprefHead...)
				rebuiltKey = append(rebuiltKey, tail...)
				ctx.Logger().Info(fmt.Sprintf("Stripped composite key: %X -> %X\n", originalKey, rebuiltKey))
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

func migrateCodeKeys(ctx sdk.Context, store sdk.KVStore) error {
	oldPrefix := LegacyPrefixes.CodeKeyPrefix
	newPrefix := wasmtypes.CodeKeyPrefix

	oldStore := prefix.NewStore(store, oldPrefix)
	iter := oldStore.Iterator(nil, nil)
	defer iter.Close()

	migratedCount := 0

	for ; iter.Valid(); iter.Next() {
		originalKey := copyBytes(iter.Key())
		originalValue := copyBytes(iter.Value())

		oldFullKey := buildFullKey(oldPrefix, originalKey)
		newFullKey := buildFullKey(newPrefix, originalKey)

		store.Set(newFullKey, originalValue)
		store.Delete(oldFullKey)
		migratedCount++
	}

	ctx.Logger().Info(fmt.Sprintf("migrated codeKey, migratedCount %d\n",
		migratedCount))
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

// Generic prefix migration from an old prefix to a new prefix
func migratePrefix(ctx sdk.Context, store sdk.KVStore, oldPrefix, newPrefix []byte, name string) error {
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

	ctx.Logger().Info(fmt.Sprintf("migrated %s, migratedCount %d\n", name, migratedCount))
	return nil
}

// Exported functions for testing
func MigrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	return migrateWasmKeys(ctx, wasmKeeper, wasmStoreKey)
}

func RemoveLengthPrefixIfNeeded(bz []byte) ([]byte, bool) {
	return removeLengthPrefixIfNeeded(bz)
}
