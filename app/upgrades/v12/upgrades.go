//nolint:revive
package v12

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"reflect"
)

func CreateV12UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Perform wasm key migration
		if err := migrateWasmKeys(ctx, keepers.WasmKeeper); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

// getWasmStoreKey uses reflection to access the unexported storeKey field
func getWasmStoreKey(k wasmkeeper.Keeper) storetypes.StoreKey {
	// Use reflection to access the unexported storeKey field
	v := reflect.ValueOf(k)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("storeKey")
	return field.Interface().(storetypes.StoreKey)
}

// migrateWasmKeys handles the migration of wasm keys from forked to original format
func migrateWasmKeys(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper) error {
	store := ctx.KVStore(getWasmStoreKey(wasmKeeper))

	// 1. Migrate sequence keys
	if err := migrateSequenceKeys(store); err != nil {
		return err
	}

	// 2. Migrate code keys
	if err := migrateCodeKeys(store); err != nil {
		return err
	}

	// 3. Migrate contract keys
	if err := migrateContractKeys(store); err != nil {
		return err
	}

	// 4. Migrate contract store keys
	if err := migrateContractStoreKeys(store); err != nil {
		return err
	}

	// 5. Migrate contract history keys
	if err := migrateContractHistoryKeys(store); err != nil {
		return err
	}

	// 6. Migrate secondary index keys
	if err := migrateSecondaryIndexKeys(store); err != nil {
		return err
	}

	// 7. Migrate params key
	if err := migrateParamsKey(store); err != nil {
		return err
	}

	return nil
}

// migrateSequenceKeys migrates from direct bytes to SequenceKeyPrefix based keys
func migrateSequenceKeys(store sdk.KVStore) error {
	// Migrate KeySequenceCodeID: 0x01 -> append(0x04, "lastCodeId"...)
	oldCodeIDKey := []byte{0x01}
	oldCodeIDValue := store.Get(oldCodeIDKey)
	if oldCodeIDValue != nil {
		newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
		store.Set(newCodeIDKey, oldCodeIDValue)
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

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()

		newStore.Set(key, value)
		oldStore.Delete(key)
	}

	return nil
}
