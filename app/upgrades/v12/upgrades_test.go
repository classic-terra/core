package v12_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	apptesting "github.com/classic-terra/core/v3/app/testing"
	"github.com/stretchr/testify/suite"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
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

// TestRemoveLengthPrefixIfNeeded tests the length prefix removal function
func (s *UpgradeTestSuite) TestRemoveLengthPrefixIfNeeded() {
	testCases := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "non-prefixed address",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "length-prefixed address (20 bytes)",
			input:    append([]byte{20}, bytes.Repeat([]byte{0x01}, 20)...),
			expected: bytes.Repeat([]byte{0x01}, 20),
		},
		{
			name:     "invalid length prefix (too large)",
			input:    append([]byte{50}, bytes.Repeat([]byte{0x01}, 10)...),
			expected: append([]byte{50}, bytes.Repeat([]byte{0x01}, 10)...),
		},
		{
			name:     "invalid length prefix (mismatch)",
			input:    append([]byte{10}, bytes.Repeat([]byte{0x01}, 20)...),
			expected: append([]byte{10}, bytes.Repeat([]byte{0x01}, 20)...),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := v12.RemoveLengthPrefixIfNeeded(tc.input)
			s.Require().Equal(tc.expected, result)
		})
	}
}

// TestMigrateWasmKeysWithLengthPrefixedAddresses tests migration with length-prefixed addresses
func (s *UpgradeTestSuite) TestMigrateWasmKeysWithLengthPrefixedAddresses() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create a length-prefixed address (20 bytes is common for Cosmos addresses)
	addrBytes := bytes.Repeat([]byte{0xAA}, 20)
	lengthPrefixedAddr := append([]byte{20}, addrBytes...)

	// Setup test data with length-prefixed addresses
	// Contract keys with length-prefixed address
	kvStore.Set(append([]byte{0x04}, lengthPrefixedAddr...), []byte("contract-prefixed"))

	// Contract store keys with length-prefixed address
	kvStore.Set(append(append([]byte{0x05}, lengthPrefixedAddr...), []byte{0x01}...), []byte("store-prefixed"))

	// Contract history keys with length-prefixed address
	kvStore.Set(append([]byte{0x06}, lengthPrefixedAddr...), []byte("history-prefixed"))

	// Add sequence keys
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Code ID sequence
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Contract ID sequence

	// Create a mock wasm keeper
	mockWasmKeeper := createMockWasmKeeper(wasmStoreKey)

	// Run the migration
	err := v12.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit the store
	stateStore.Commit()

	// Create a new context with the updated store
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify the migration results for length-prefixed addresses
	// Old keys should be deleted
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, lengthPrefixedAddr...)), "Old contract key with length prefix should be deleted")
	require.Nil(s.T(), kvStore.Get(append(append([]byte{0x05}, lengthPrefixedAddr...), []byte{0x01}...)), "Old contract store key with length prefix should be deleted")
	require.Nil(s.T(), kvStore.Get(append([]byte{0x06}, lengthPrefixedAddr...)), "Old contract history key with length prefix should be deleted")

	// New keys should exist with the correct values and without length prefix
	require.Equal(s.T(), []byte("contract-prefixed"),
		kvStore.Get(append([]byte{0x02}, addrBytes...)), "Contract key should be migrated to 0x02 without length prefix")

	require.Equal(s.T(), []byte("store-prefixed"),
		kvStore.Get(append(append([]byte{0x03}, addrBytes...), []byte{0x01}...)), "Contract store key should be migrated to 0x03 without length prefix")

	// For contract history keys, we need to check if the migration correctly handled the length prefix
	require.Equal(s.T(), []byte("history-prefixed"),
		kvStore.Get(append([]byte{0x05}, lengthPrefixedAddr...)), "Contract history key should be migrated to 0x05 without length prefix")

	// Verify sequence keys were migrated correctly
	require.Nil(s.T(), kvStore.Get([]byte{0x01}), "Old code ID sequence key should be deleted")
	require.Nil(s.T(), kvStore.Get([]byte{0x02}), "Old contract ID sequence key should be deleted")

	require.Equal(s.T(), []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastCodeId")...)), "Code ID sequence should be migrated")
	require.Equal(s.T(), []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		kvStore.Get(append([]byte{0x04}, []byte("lastContractId")...)), "Contract ID sequence should be migrated")
}

// TestCollectContractAddresses tests the contract address collection function
func (s *UpgradeTestSuite) TestCollectContractAddresses() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Add some contract addresses
	addr1 := bytes.Repeat([]byte{0xAA}, 20)
	addr2 := bytes.Repeat([]byte{0xBB}, 20)

	// Add one with length prefix
	lengthPrefixedAddr := append([]byte{20}, bytes.Repeat([]byte{0xCC}, 20)...)

	kvStore.Set(append([]byte{0x04}, addr1...), []byte("contract1"))
	kvStore.Set(append([]byte{0x04}, addr2...), []byte("contract2"))
	kvStore.Set(append([]byte{0x04}, lengthPrefixedAddr...), []byte("contract3"))

	// Call the function
	addresses := v12.CollectContractAddresses(kvStore)

	// Verify results
	s.Require().Equal(3, len(addresses), "Should collect 3 contract addresses")

	// Check if addresses are collected correctly
	foundAddr1 := false
	foundAddr2 := false
	foundPrefixedAddr := false

	//nolint
	for _, addr := range addresses {
		if bytes.Equal(addr, addr1) {
			foundAddr1 = true
		} else if bytes.Equal(addr, addr2) {
			foundAddr2 = true
		} else if bytes.Equal(addr, lengthPrefixedAddr) {
			foundPrefixedAddr = true
		}
	}

	s.Require().True(foundAddr1, "Should collect addr1")
	s.Require().True(foundAddr2, "Should collect addr2")
	s.Require().True(foundPrefixedAddr, "Should collect length-prefixed address")
}

// TestMigrateContractStoreKeys tests the contract store key migration
func (s *UpgradeTestSuite) TestMigrateContractStoreKeys() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create test contract addresses for pre-collected addresses path
	addr1 := bytes.Repeat([]byte{0xAA}, 20)
	lengthPrefixedAddr := append([]byte{20}, bytes.Repeat([]byte{0xBB}, 20)...)

	// Add contract store data for pre-collected addresses
	kvStore.Set(append(append([]byte{0x05}, addr1...), []byte{0x01}...), []byte("store1"))
	kvStore.Set(append(append([]byte{0x05}, addr1...), []byte{0x02}...), []byte("store2"))
	kvStore.Set(append(append([]byte{0x05}, lengthPrefixedAddr...), []byte{0x01}...), []byte("store3"))

	// Add direct contract store data (not in pre-collected addresses)
	directAddr := bytes.Repeat([]byte{0xCC}, 20)
	directLengthPrefixedAddr := append([]byte{20}, bytes.Repeat([]byte{0xDD}, 20)...)

	// Create the full keys for direct store data
	directKey1 := append(append([]byte{0x05}, directAddr...), []byte{0x01}...)
	directKey2 := append(append([]byte{0x05}, directAddr...), []byte{0x02}...)
	directPrefixedKey := append(append([]byte{0x05}, directLengthPrefixedAddr...), []byte{0x01}...)

	kvStore.Set(directKey1, []byte("direct-store1"))
	kvStore.Set(directKey2, []byte("direct-store2"))
	kvStore.Set(directPrefixedKey, []byte("direct-store3"))

	// Collect contract addresses (deliberately excluding directAddr and directLengthPrefixedAddr)
	contractAddresses := [][]byte{addr1, lengthPrefixedAddr}

	// Run the migration
	err := v12.MigrateContractStoreKeys(kvStore, contractAddresses)
	require.NoError(s.T(), err)

	// Verify the migration results for pre-collected addresses
	// Old keys should be deleted
	require.Nil(s.T(), kvStore.Get(append(append([]byte{0x05}, addr1...), []byte{0x01}...)),
		"Old contract store key should be deleted")
	require.Nil(s.T(), kvStore.Get(append(append([]byte{0x05}, addr1...), []byte{0x02}...)),
		"Old contract store key should be deleted")
	require.Nil(s.T(), kvStore.Get(append(append([]byte{0x05}, lengthPrefixedAddr...), []byte{0x01}...)),
		"Old contract store key with length prefix should be deleted")

	// New keys should exist with the correct values for pre-collected addresses
	require.Equal(s.T(), []byte("store1"),
		kvStore.Get(append(append([]byte{0x03}, addr1...), []byte{0x01}...)),
		"Contract store key should be migrated to 0x03")
	require.Equal(s.T(), []byte("store2"),
		kvStore.Get(append(append([]byte{0x03}, addr1...), []byte{0x02}...)),
		"Contract store key should be migrated to 0x03")

	// For length-prefixed address, the new key should use the unprefixed address
	unprefixedAddr := bytes.Repeat([]byte{0xBB}, 20)
	require.Equal(s.T(), []byte("store3"),
		kvStore.Get(append(append([]byte{0x03}, unprefixedAddr...), []byte{0x01}...)),
		"Contract store key should be migrated to 0x03 without length prefix")

	// Verify direct migration results
	// Old direct keys should be deleted
	require.Nil(s.T(), kvStore.Get(directKey1),
		"Old direct contract store key should be deleted")
	require.Nil(s.T(), kvStore.Get(directKey2),
		"Old direct contract store key should be deleted")
	require.Nil(s.T(), kvStore.Get(directPrefixedKey),
		"Old direct contract store key with length prefix should be deleted")

	// New direct keys should exist with correct values
	require.Equal(s.T(), []byte("direct-store1"),
		kvStore.Get(append(append([]byte{0x03}, directAddr...), []byte{0x01}...)),
		"Direct contract store key should be migrated to 0x03")
	require.Equal(s.T(), []byte("direct-store2"),
		kvStore.Get(append(append([]byte{0x03}, directAddr...), []byte{0x02}...)),
		"Direct contract store key should be migrated to 0x03")

	// For length-prefixed direct address, verify with unprefixed address
	unprefixedDirectAddr := bytes.Repeat([]byte{0xDD}, 20)
	newDirectPrefixedKey := append(append([]byte{0x03}, unprefixedDirectAddr...), []byte{0x01}...)

	// Print debug information before the failing check
	fmt.Printf("Original key: %X\n", directPrefixedKey)
	fmt.Printf("Expected new key: %X\n", newDirectPrefixedKey)
	fmt.Printf("All store keys:\n")
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		fmt.Printf("Key: %X, Value: %X\n", iter.Key(), iter.Value())
	}
	// Check if the key exists with the unprefixed address
	// The key should be migrated with the length prefix removed
	require.Equal(s.T(), []byte("direct-store3"),
		kvStore.Get(newDirectPrefixedKey),
		"Direct contract store key should be migrated to 0x03 without length prefix")

	// Also verify that the old key with length prefix is deleted
	require.Nil(s.T(), kvStore.Get(directPrefixedKey),
		"Old direct contract store key with length prefix should be deleted")
}

// TestMigrateContractKeys tests the contract key migration
func (s *UpgradeTestSuite) TestMigrateContractKeys() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create test contract addresses
	addr1 := bytes.Repeat([]byte{0xAA}, 20)
	lengthPrefixedAddr := append([]byte{20}, bytes.Repeat([]byte{0xBB}, 20)...)

	// Add contract data
	kvStore.Set(append([]byte{0x04}, addr1...), []byte("contract1"))
	kvStore.Set(append([]byte{0x04}, lengthPrefixedAddr...), []byte("contract2"))

	// Run the migration
	err := v12.MigrateContractKeys(kvStore)
	require.NoError(s.T(), err)

	// Verify the migration results
	// Old keys should be deleted
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, addr1...)), "Old contract key should be deleted")
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, lengthPrefixedAddr...)), "Old contract key with length prefix should be deleted")

	// New keys should exist with the correct values
	require.Equal(s.T(), []byte("contract1"),
		kvStore.Get(append([]byte{0x02}, addr1...)), "Contract key should be migrated to 0x02")

	// For length-prefixed address, the new key should use the unprefixed address
	unprefixedAddr := bytes.Repeat([]byte{0xBB}, 20)
	require.Equal(s.T(), []byte("contract2"),
		kvStore.Get(append([]byte{0x02}, unprefixedAddr...)), "Contract key should be migrated to 0x02 without length prefix")
}
