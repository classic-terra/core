package v13_test

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
	v13 "github.com/classic-terra/core/v3/app/upgrades/v13"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

type UpgradeTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

// Configure address verification for tests so 20-byte payloads are valid addresses.
func (s *UpgradeTestSuite) SetupSuite() {
	cfg := sdk.GetConfig()
	cfg.SetAddressVerifier(func(bz []byte) error {
		if len(bz) == 20 {
			return nil
		}
		return fmt.Errorf("invalid address length %d", len(bz))
	})
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
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
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
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
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
	addresses := v13.CollectContractAddresses(kvStore, ctx.Logger())

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
	// Composite-looking head but not a valid address: len=5 + 5 bytes
	fakeHead := append([]byte{0x05}, bytes.Repeat([]byte{0xEE}, 5)...)
	directCompositeKey := append(append([]byte{0x05}, fakeHead...), []byte{0x01}...)

	kvStore.Set(directKey1, []byte("direct-store1"))
	kvStore.Set(directKey2, []byte("direct-store2"))
	kvStore.Set(directPrefixedKey, []byte("direct-store3"))
	kvStore.Set(directCompositeKey, []byte("direct-store4"))

	// Collect contract addresses (deliberately excluding directAddr and directLengthPrefixedAddr)
	contractAddresses := [][]byte{addr1, lengthPrefixedAddr}

	// Run the migration
	err := v13.MigrateContractStoreKeys(kvStore, contractAddresses)
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

	// Composite-looking head (len=5) must NOT be stripped; should migrate intact
	expectedCompositeNewKey := append(append([]byte{0x03}, fakeHead...), []byte{0x01}...)
	require.Equal(s.T(), []byte("direct-store4"),
		kvStore.Get(expectedCompositeNewKey),
		"Composite head should remain unchanged in new key")
	require.Nil(s.T(), kvStore.Get(directCompositeKey),
		"Old composite key should be deleted after migration")
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
	// Not an address: len=5 + 5 bytes
	fakeHead := append([]byte{0x05}, bytes.Repeat([]byte{0xAB}, 5)...)

	// Add contract data
	kvStore.Set(append([]byte{0x04}, addr1...), []byte("contract1"))
	kvStore.Set(append([]byte{0x04}, lengthPrefixedAddr...), []byte("contract2"))
	kvStore.Set(append([]byte{0x04}, fakeHead...), []byte("notacontract"))

	// Run the migration
	err := v13.MigrateContractKeys(kvStore)
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

	// For non-address head (len=5), it must be keeped
	require.Equal(s.T(), []byte("notacontract"),
		kvStore.Get(append([]byte{0x04}, fakeHead...)), "Non-address-like head must remain unchanged")
}

// TestReadContractHistoryWithFallback tests reading contract history with fallback to old prefix
func (s *UpgradeTestSuite) TestReadContractHistoryWithFallback() {
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
	addr2 := bytes.Repeat([]byte{0xBB}, 20)
	addr3 := bytes.Repeat([]byte{0xCC}, 20)
	lengthPrefixedAddr := append([]byte{20}, bytes.Repeat([]byte{0xDD}, 20)...)

	// Scenario 1: Contract history exists in new prefix (0x05) - migrated data
	kvStore.Set(append([]byte{0x05}, addr1...), []byte("migrated-history-1"))

	// Scenario 2: Contract history exists in old prefix (0x06) - unmigrated data
	kvStore.Set(append([]byte{0x06}, addr2...), []byte("unmigrated-history-2"))

	// Scenario 3: Contract history exists in old prefix with length-prefixed address
	kvStore.Set(append([]byte{0x06}, lengthPrefixedAddr...), []byte("unmigrated-history-3"))

	// Scenario 4: Contract history exists in both prefixes (should prefer new one)
	kvStore.Set(append([]byte{0x05}, addr3...), []byte("migrated-history-3"))
	kvStore.Set(append([]byte{0x06}, addr3...), []byte("unmigrated-history-3-old"))

	// Test reading from new prefix (migrated data)
	value, found := v13.ReadContractHistoryWithFallback(kvStore, addr1)
	s.Require().True(found, "Should find migrated contract history")
	s.Require().Equal([]byte("migrated-history-1"), value, "Should return migrated history")

	// Test reading from old prefix (unmigrated data)
	value, found = v13.ReadContractHistoryWithFallback(kvStore, addr2)
	s.Require().True(found, "Should find unmigrated contract history")
	s.Require().Equal([]byte("unmigrated-history-2"), value, "Should return unmigrated history")

	// Test reading from old prefix with length-prefixed address
	unprefixedAddr3 := bytes.Repeat([]byte{0xDD}, 20)
	value, found = v13.ReadContractHistoryWithFallback(kvStore, unprefixedAddr3)
	s.Require().True(found, "Should find unmigrated contract history with length prefix")
	s.Require().Equal([]byte("unmigrated-history-3"), value, "Should return unmigrated history")

	// Test preference for new prefix when both exist
	value, found = v13.ReadContractHistoryWithFallback(kvStore, addr3)
	s.Require().True(found, "Should find contract history when both exist")
	s.Require().Equal([]byte("migrated-history-3"), value, "Should prefer migrated history over unmigrated")

	// Test non-existent contract
	nonExistentAddr := bytes.Repeat([]byte{0xFF}, 20)
	value, found = v13.ReadContractHistoryWithFallback(kvStore, nonExistentAddr)
	s.Require().False(found, "Should not find non-existent contract history")
	s.Require().Nil(value, "Should return nil for non-existent contract")
}

// TestIterateContractHistoryWithFallback tests iteration over contract history with fallback
func (s *UpgradeTestSuite) TestIterateContractHistoryWithFallback() {
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
	addr2 := bytes.Repeat([]byte{0xBB}, 20)
	addr3 := bytes.Repeat([]byte{0xCC}, 20)
	addr4 := bytes.Repeat([]byte{0xDD}, 20)
	lengthPrefixedAddr4 := append([]byte{20}, addr4...)

	// Add contract history in new prefix (0x05) - migrated data
	kvStore.Set(append([]byte{0x05}, addr1...), []byte("migrated-history-1"))
	kvStore.Set(append([]byte{0x05}, addr2...), []byte("migrated-history-2"))

	// Add contract history in old prefix (0x06) - unmigrated data
	kvStore.Set(append([]byte{0x06}, addr3...), []byte("unmigrated-history-3"))
	kvStore.Set(append([]byte{0x06}, lengthPrefixedAddr4...), []byte("unmigrated-history-4"))

	// Add duplicate in both prefixes (should only see migrated version)
	kvStore.Set(append([]byte{0x05}, addr3...), []byte("migrated-history-3-new"))

	// Collect all contract histories
	var contractHistories []struct {
		addr    []byte
		history []byte
	}

	v13.IterateContractHistoryWithFallback(kvStore, func(contractAddr []byte, history []byte) bool {
		contractHistories = append(contractHistories, struct {
			addr    []byte
			history []byte
		}{
			addr:    contractAddr,
			history: history,
		})
		return true
	})

	// Verify results
	s.Require().Equal(4, len(contractHistories), "Should find exactly 4 contract histories")

	// Convert to map for easier verification
	historyMap := make(map[string]string)
	for _, ch := range contractHistories {
		historyMap[string(ch.addr)] = string(ch.history)
	}

	// Verify migrated data
	s.Require().Equal("migrated-history-1", historyMap[string(addr1)], "Should have migrated history for addr1")
	s.Require().Equal("migrated-history-2", historyMap[string(addr2)], "Should have migrated history for addr2")

	// Verify preference for migrated data when both exist
	s.Require().Equal("migrated-history-3-new", historyMap[string(addr3)], "Should prefer migrated history for addr3")

	// Verify unmigrated data with length prefix removed
	s.Require().Equal("unmigrated-history-4", historyMap[string(addr4)], "Should have unmigrated history for addr4 with length prefix removed")

	// Test early termination
	var count int
	v13.IterateContractHistoryWithFallback(kvStore, func(contractAddr []byte, history []byte) bool {
		count++
		return count < 2 // Stop after 2 iterations
	})

	s.Require().Equal(2, count, "Should stop iteration early when callback returns false")
}

// TestReadContractInfoWithFallback verifies reading contract info across prefixes
func (s *UpgradeTestSuite) TestReadContractInfoWithFallback() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	addr := bytes.Repeat([]byte{0xAA}, 20)
	lengthPrefixedAddr := append([]byte{20}, addr...)

	// Encode a minimal ContractInfo
	info := wasmtypes.ContractInfo{CodeID: 7}
	bz, err := info.Marshal()
	s.Require().NoError(err)

	// Case 1: Only new prefix
	kvStore.Set(append([]byte{0x02}, addr...), bz)
	v, ok := v13.ReadContractInfoWithFallback(kvStore, addr)
	s.Require().True(ok)
	s.Require().Equal(bz, v)
	kvStore.Delete(append([]byte{0x02}, addr...))

	// Case 2: Only old prefix without length prefix
	kvStore.Set(append([]byte{0x04}, addr...), bz)
	v, ok = v13.ReadContractInfoWithFallback(kvStore, addr)
	s.Require().True(ok)
	s.Require().Equal(bz, v)
	kvStore.Delete(append([]byte{0x04}, addr...))

	// Case 3: Old prefix with length-prefixed address
	kvStore.Set(append([]byte{0x04}, lengthPrefixedAddr...), bz)
	v, ok = v13.ReadContractInfoWithFallback(kvStore, addr)
	s.Require().True(ok)
	s.Require().Equal(bz, v)
}

// TestReadRawContractStateWithFallback verifies single key reads across prefixes
func (s *UpgradeTestSuite) TestReadRawContractStateWithFallback() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	addr := bytes.Repeat([]byte{0xAB}, 20)
	lengthPrefixedAddr := append([]byte{20}, addr...)
	key := []byte{0x01, 0x02}
	val := []byte("value")

	// Case 1: New prefix
	kvStore.Set(append(append([]byte{0x03}, addr...), key...), val)
	v, ok := v13.ReadRawContractStateWithFallback(kvStore, addr, key)
	s.Require().True(ok)
	s.Require().Equal(val, v)
	kvStore.Delete(append(append([]byte{0x03}, addr...), key...))

	// Case 2: Old prefix without length prefix
	kvStore.Set(append(append([]byte{0x05}, addr...), key...), val)
	v, ok = v13.ReadRawContractStateWithFallback(kvStore, addr, key)
	s.Require().True(ok)
	s.Require().Equal(val, v)
	kvStore.Delete(append(append([]byte{0x05}, addr...), key...))

	// Case 3: Old prefix with length-prefixed address
	kvStore.Set(append(append([]byte{0x05}, lengthPrefixedAddr...), key...), val)
	v, ok = v13.ReadRawContractStateWithFallback(kvStore, addr, key)
	s.Require().True(ok)
	s.Require().Equal(val, v)
}

// TestIterateAllContractStateWithFallback verifies iteration across prefixes
func (s *UpgradeTestSuite) TestIterateAllContractStateWithFallback() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	addr := bytes.Repeat([]byte{0xAC}, 20)
	lengthPrefixedAddr := append([]byte{20}, addr...)

	// Populate new prefix entries
	kvStore.Set(append(append([]byte{0x03}, addr...), []byte{0x01}...), []byte("v1"))
	kvStore.Set(append(append([]byte{0x03}, addr...), []byte{0x02}...), []byte("v2"))

	// Populate old prefix entries (should be skipped if duplicate keys)
	kvStore.Set(append(append([]byte{0x05}, addr...), []byte{0x02}...), []byte("old-v2"))

	// Populate old prefix with length-prefixed address
	kvStore.Set(append(append([]byte{0x05}, lengthPrefixedAddr...), []byte{0x03}...), []byte("v3"))

	var keys [][]byte
	var values [][]byte
	v13.IterateAllContractStateWithFallback(kvStore, addr, func(k, v []byte) bool {
		keys = append(keys, append([]byte{}, k...))
		values = append(values, append([]byte{}, v...))
		return true
	})

	// We expect keys: 0x01, 0x02, 0x03 with values v1, v2, v3
	s.Require().Equal(3, len(keys))
	s.Require().ElementsMatch([][]byte{{0x01}, {0x02}, {0x03}}, keys)
	s.Require().ElementsMatch([][]byte{[]byte("v1"), []byte("v2"), []byte("v3")}, values)
}
