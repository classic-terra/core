package v12_test

import (
	"fmt"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	apptesting "github.com/classic-terra/core/v3/app/testing"
	"github.com/stretchr/testify/suite"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/classic-terra/core/v3/app/keepers"
	v12 "github.com/classic-terra/core/v3/app/upgrades/v12"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

type UpgradeTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

// TestMigrateWasmKeys tests the migration of wasm keys
func (s *UpgradeTestSuite) TestMigrateWasmKeys() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Setup test data in the old format
	kvStore := ctx.KVStore(wasmStoreKey)

	// Sequence keys
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Code ID sequence
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Contract ID sequence

	// Code keys
	kvStore.Set([]byte{0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, []byte("code1"))

	// Contract keys
	kvStore.Set([]byte{0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, []byte("contract1"))

	// Contract store keys
	kvStore.Set([]byte{0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, []byte("store1"))

	// Contract history keys
	kvStore.Set([]byte{0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, []byte("history1"))

	// Secondary index keys
	kvStore.Set([]byte{0x10, 0x01, 0x00}, []byte("index1"))

	// Params key
	kvStore.Set([]byte{0x11}, []byte("params"))

	// Create a mock wasm keeper with the store key
	mockWasmKeeper := createMockWasmKeeper(wasmStoreKey)

	// Run the migration
	err := v12.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Try to flush the cache directly
	cacheKVStore, ok := kvStore.(storetypes.CacheKVStore)
	if ok {
		fmt.Println("Found CacheKVStore, writing to underlying store")
		cacheKVStore.Write()
	}

	// Commit the store
	stateStore.Commit()

	// Create a new context with the updated store
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify the migration results

	// Old keys should be deleted
	require.Nil(s.T(), kvStore.Get([]byte{0x01}), "Old sequence code ID key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x02}), "Old sequence instance ID key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Old code key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Old contract key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}), "Old contract store key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Old contract history key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x10, 0x01, 0x00}), "Old secondary index key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x11}), "Old params key should be deleted")

	// New keys should exist with the correct values
	// Migration order in the implementation:
	// 1. Secondary index keys: 0x10 -> 0x06
	require.Equal(s.T(), []byte("index1"),
		kvStore.Get([]byte{0x06, 0x01, 0x00}), "Secondary index key should be migrated to 0x06")

	// 2. Params key: 0x11 -> 0x10
	require.Equal(s.T(), []byte("params"),
		kvStore.Get([]byte{0x10}), "Params key should be migrated to 0x10")

	// 3. Contract keys: 0x04 -> 0x02
	require.Equal(s.T(), []byte("contract1"),
		kvStore.Get([]byte{0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Contract key should be migrated to 0x02")

	// 4. Sequence keys: 0x01, 0x02 -> append(0x04, "lastCodeId"...), append(0x04, "lastContractId"...)
	require.Equal(s.T(), []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastCodeId")...)), "Sequence code ID key should be migrated to 0x04+lastCodeId")
	require.Equal(s.T(), []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastContractId")...)), "Sequence instance ID key should be migrated to 0x04+lastContractId")

	// 5. Code keys: 0x03 -> 0x01
	require.Equal(s.T(), []byte("code1"),
		kvStore.Get([]byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Code key should be migrated to 0x01")

	// 6. Contract store keys: 0x05 -> 0x03
	require.Equal(s.T(), []byte("store1"),
		kvStore.Get([]byte{0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}), "Contract store key should be migrated to 0x03")

	// 7. Contract history keys: 0x06 -> 0x05
	require.Equal(s.T(), []byte("history1"),
		kvStore.Get([]byte{0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), "Contract history key should be migrated to 0x05")
}

// TestMigrateWasmKeysWithContractState tests the migration of wasm keys with focus on contract state
func (s *UpgradeTestSuite) TestMigrateWasmKeysWithContractState() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Setup test data in the old format
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create multiple contract states with different data to test migration thoroughly

	// Contract 1 with multiple state entries
	contractID1 := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	kvStore.Set([]byte{0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, []byte("contract1")) // Contract key

	// Contract 1 state entries (using 0x05 prefix for contract store)
	kvStore.Set(append([]byte{0x05}, append(contractID1, []byte("balance")...)...), []byte("100"))
	kvStore.Set(append([]byte{0x05}, append(contractID1, []byte("owner")...)...), []byte("address1"))
	kvStore.Set(append([]byte{0x05}, append(contractID1, []byte("config")...)...), []byte("{\"key\":\"value\"}"))

	// Contract 2 with different state structure
	contractID2 := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	kvStore.Set([]byte{0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, []byte("contract2")) // Contract key

	// Contract 2 state entries
	kvStore.Set(append([]byte{0x05}, append(contractID2, []byte("tokens")...)...), []byte("[1,2,3,4,5]"))
	kvStore.Set(append([]byte{0x05}, append(contractID2, []byte("admin")...)...), []byte("admin_address"))

	// Add some contract history entries
	kvStore.Set([]byte{0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, []byte("history_contract1"))
	kvStore.Set([]byte{0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, []byte("history_contract2"))

	// Add code entries
	kvStore.Set([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, []byte("code1"))

	// Add sequence keys
	kvStore.Set([]byte{0x01}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Code ID sequence
	kvStore.Set([]byte{0x02}, []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Contract ID sequence

	// Add secondary index and params
	kvStore.Set([]byte{0x10, 0x01, 0x00}, []byte("index1"))
	kvStore.Set([]byte{0x11}, []byte("params"))

	// Create a mock wasm keeper with the store key
	mockWasmKeeper := createMockWasmKeeper(wasmStoreKey)

	// Take a snapshot of all contract state before migration
	contractStates := make(map[string][]byte)

	// Capture all contract 1 states
	storePrefix1 := append([]byte{0x05}, contractID1...)
	iterContract1 := sdk.KVStorePrefixIterator(kvStore, storePrefix1)
	defer iterContract1.Close()
	for ; iterContract1.Valid(); iterContract1.Next() {
		key := iterContract1.Key()
		contractStates[string(key)] = iterContract1.Value()
	}

	// Capture all contract 2 states
	storePrefix2 := append([]byte{0x05}, contractID2...)
	iterContract2 := sdk.KVStorePrefixIterator(kvStore, storePrefix2)
	defer iterContract2.Close()
	for ; iterContract2.Valid(); iterContract2.Next() {
		key := iterContract2.Key()
		contractStates[string(key)] = iterContract2.Value()
	}

	// Run the migration
	err := v12.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit the store
	stateStore.Commit()

	// Create a new context with the updated store
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify contract state migration

	// Check contract 1 states - should be migrated from 0x05 to 0x03
	newStorePrefix1 := append([]byte{0x03}, contractID1...)
	iterNewContract1 := sdk.KVStorePrefixIterator(kvStore, newStorePrefix1)
	defer iterNewContract1.Close()

	var migratedStateCount1 int
	for ; iterNewContract1.Valid(); iterNewContract1.Next() {
		key := iterNewContract1.Key()
		value := iterNewContract1.Value()

		// Construct the old key to check against our saved states
		oldKey := append([]byte{0x05}, key[1:]...) // Replace 0x03 with 0x05

		// Verify the value matches what we had before migration
		require.Equal(s.T(), contractStates[string(oldKey)], value,
			"Contract 1 state value mismatch for key %v", key[len(newStorePrefix1):])

		migratedStateCount1++
	}

	// Check we found all contract 1 states
	expectedStateCount1 := 3 // balance, owner, config
	require.Equal(s.T(), expectedStateCount1, migratedStateCount1,
		"Not all contract 1 states were migrated")

	// Check contract 2 states
	newStorePrefix2 := append([]byte{0x03}, contractID2...)
	iterNewContract2 := sdk.KVStorePrefixIterator(kvStore, newStorePrefix2)
	defer iterNewContract2.Close()

	var migratedStateCount2 int
	for ; iterNewContract2.Valid(); iterNewContract2.Next() {
		key := iterNewContract2.Key()
		value := iterNewContract2.Value()

		// Construct the old key to check against our saved states
		oldKey := append([]byte{0x05}, key[1:]...) // Replace 0x03 with 0x05

		// Verify the value matches what we had before migration
		require.Equal(s.T(), contractStates[string(oldKey)], value,
			"Contract 2 state value mismatch for key %v", key[len(newStorePrefix2):])

		migratedStateCount2++
	}

	// Check we found all contract 2 states
	expectedStateCount2 := 2 // tokens, admin
	require.Equal(s.T(), expectedStateCount2, migratedStateCount2,
		"Not all contract 2 states were migrated")

	// Verify contract keys were migrated (0x04 -> 0x02)
	require.Equal(s.T(), []byte("contract1"),
		kvStore.Get([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}),
		"Contract 1 key should be migrated to 0x02")

	require.Equal(s.T(), []byte("contract2"),
		kvStore.Get([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}),
		"Contract 2 key should be migrated to 0x02")

	// Verify contract history keys were migrated (0x06 -> 0x05)
	require.Equal(s.T(), []byte("history_contract1"),
		kvStore.Get([]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}),
		"Contract 1 history should be migrated to 0x05")

	require.Equal(s.T(), []byte("history_contract2"),
		kvStore.Get([]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}),
		"Contract 2 history should be migrated to 0x05")

	// Verify code keys were migrated (0x03 -> 0x01)
	require.Equal(s.T(), []byte("code1"),
		kvStore.Get([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}),
		"Code key should be migrated to 0x01")

	// Verify sequence keys were migrated
	require.Equal(s.T(), []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastCodeId")...)),
		"Code ID sequence should be migrated to 0x04+lastCodeId")

	require.Equal(s.T(), []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastContractId")...)),
		"Contract ID sequence should be migrated to 0x04+lastContractId")

	// Verify secondary index and params were migrated
	require.Equal(s.T(), []byte("index1"),
		kvStore.Get([]byte{0x06, 0x01, 0x00}),
		"Secondary index should be migrated to 0x06")

	require.Equal(s.T(), []byte("params"),
		kvStore.Get([]byte{0x10}),
		"Params should be migrated to 0x10")
}

// createMockWasmKeeper creates a mock wasm keeper with the given store key
func createMockWasmKeeper(storeKey storetypes.StoreKey) wasmkeeper.Keeper {
	// Create a minimal mock keeper that has the store key
	// We only need the storeKey field to be set for the migration to work

	// Create a mock keeper using reflection to set just the storeKey field
	// This is a hack, but it's the simplest way to create a mock keeper for testing
	// without having to provide all the dependencies

	// Create an empty keeper
	keeper := wasmkeeper.Keeper{}

	// Use reflection to set the storeKey field
	keVal := reflect.ValueOf(&keeper).Elem()
	storeKeyField := keVal.FieldByName("storeKey")

	// Check if the field exists and is settable
	if storeKeyField.IsValid() && storeKeyField.CanSet() {
		storeKeyVal := reflect.ValueOf(storeKey)
		storeKeyField.Set(storeKeyVal)
	}

	return keeper
}

// TestCreateV12UpgradeHandler tests the upgrade handler creation
func (s *UpgradeTestSuite) TestCreateV12UpgradeHandler() {
	s.Setup(s.T(), "terra")

	// This is a simple test to ensure the upgrade handler is created without errors
	handler := v12.CreateV12UpgradeHandler(nil, nil, nil, &keepers.AppKeepers{})
	s.Require().NotNil(handler)
}

// TestUpgradeHandlerWithKeeperTestHelper tests the upgrade handler with a more realistic setup
func (s *UpgradeTestSuite) TestUpgradeHandlerWithKeeperTestHelper() {
	// Setup the test environment
	s.Setup(s.T(), "terra")

	// Create the upgrade handler with nil values
	// We're just testing that the handler can be created without errors
	handler := v12.CreateV12UpgradeHandler(nil, nil, nil, &keepers.AppKeepers{})

	// Verify the handler is created
	s.Require().NotNil(handler)
}
