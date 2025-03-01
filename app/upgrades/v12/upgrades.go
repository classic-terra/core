//nolint:revive
package v12

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

func CreateV12UpgradeHandler(
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

// MigrateWasmKeys handles the migration of wasm keys from forked to original format
// Exported for testing
func MigrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	return migrateWasmKeys(ctx, wasmKeeper, wasmStoreKey)
}

// migrateWasmKeys handles the migration of wasm keys from forked to original format
func migrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper, wasmStoreKey storetypes.StoreKey) error {
	store := ctx.KVStore(wasmStoreKey)

	// Log the migration start
	ctx.Logger().Info("Starting WASM key migration from forked to original format")

	// We need to be careful about the order of migrations to avoid overwriting data
	// First, migrate keys that don't conflict with any destination keys

	// 1. Migrate contract keys (0x04 -> 0x02)
	// This needs to happen before sequence keys migration to free up 0x04
	if err := migrateContractKeys(store); err != nil {
		return err
	}

	// 2. Migrate sequence keys (0x01 -> 0x04 with appended strings)
	// This can only be done after contract keys are migrated away from 0x04
	// This frees up 0x01 for code keys
	if err := migrateSequenceKeys(store); err != nil {
		return err
	}

	// 3. Migrate code keys (0x03 -> 0x01)
	// This can only be done after sequence keys are migrated away from 0x01
	if err := migrateCodeKeys(store); err != nil {
		return err
	}

	// 4. Migrate contract store keys (0x05 -> 0x03)
	// This needs to happen before contract history keys migration
	if err := migrateContractStoreKeys(store); err != nil {
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

// migrateSequenceKeys migrates from direct bytes to SequenceKeyPrefix based keys
func migrateSequenceKeys(store sdk.KVStore) error {
	// Migrate KeySequenceCodeID: 0x01 -> append(0x04, "lastCodeId"...)
	oldCodeIDKey := []byte{0x01}
	oldCodeIDValue := store.Get(oldCodeIDKey)

	if oldCodeIDValue != nil {
		newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
		// Set the new key with the old value
		store.Set(newCodeIDKey, oldCodeIDValue)
		// Delete the old key
		store.Delete(oldCodeIDKey)
	}

	// Migrate KeySequenceInstanceID: 0x02 -> append(0x04, "lastContractId"...)
	oldInstanceIDKey := []byte{0x02}
	oldInstanceIDValue := store.Get(oldInstanceIDKey)
	if oldInstanceIDValue != nil {
		newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
		store.Set(newInstanceIDKey, oldInstanceIDValue)
		store.Delete(oldInstanceIDKey)
	}

	return nil
}

// migrateCodeKeys migrates code keys from 0x03 to 0x01
func migrateCodeKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x03}
	newPrefix := []byte{0x01}
	return migratePrefix(store, oldPrefix, newPrefix)
}

// migrateContractKeys migrates contract keys from 0x04 to 0x02
func migrateContractKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x04}
	newPrefix := []byte{0x02}
	return migratePrefix(store, oldPrefix, newPrefix)
}

// migrateContractStoreKeys migrates contract store keys from 0x05 to 0x03
func migrateContractStoreKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x05}
	newPrefix := []byte{0x03}
	return migratePrefix(store, oldPrefix, newPrefix)
}

// migrateContractHistoryKeys migrates contract history keys from 0x06 to 0x05
func migrateContractHistoryKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x06}
	newPrefix := []byte{0x05}
	return migratePrefix(store, oldPrefix, newPrefix)
}

// migrateSecondaryIndexKeys migrates secondary index keys from 0x10 to 0x06
func migrateSecondaryIndexKeys(store sdk.KVStore) error {
	oldPrefix := []byte{0x10}
	newPrefix := []byte{0x06}
	return migratePrefix(store, oldPrefix, newPrefix)
}

// migrateParamsKey migrates params key from 0x11 to 0x10
func migrateParamsKey(store sdk.KVStore) error {
	oldKey := []byte{0x11}
	newKey := []byte{0x10}

	value := store.Get(oldKey)
	if value != nil {
		store.Set(newKey, value)
		store.Delete(oldKey)
	}

	return nil
}

// migratePrefix is a helper function to migrate all keys with a given prefix
func migratePrefix(store sdk.KVStore, oldPrefix, newPrefix []byte) error {
	oldStore := prefix.NewStore(store, oldPrefix)
	newStore := prefix.NewStore(store, newPrefix)

	iterator := oldStore.Iterator(nil, nil)
	defer iterator.Close()

	var migratedCount int

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()

		newStore.Set(key, value)
		oldStore.Delete(key)
		migratedCount++
	}

	fmt.Println("migrated", migratedCount)

	return nil
}
