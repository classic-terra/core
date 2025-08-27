package v13_test

import (
	"bytes"
	"fmt"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	apptesting "github.com/classic-terra/core/v3/app/testing"
	v13 "github.com/classic-terra/core/v3/app/upgrades/v13"
)

type MigrationRetryTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

func TestMigrationRetryTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationRetryTestSuite))
}

// TestMigrationIdempotency ensures the migration can be safely retried without corruption
func (s *MigrationRetryTestSuite) TestMigrationIdempotency() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Setup initial test data
	kvStore := ctx.KVStore(wasmStoreKey)
	s.setupTestData(kvStore)

	// Capture initial state (for potential debugging)
	_ = s.captureStoreState(kvStore)

	// Create mock wasm keeper
	mockWasmKeeper := wasmkeeper.Keeper{}

	// Run migration first time
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit changes
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Capture state after first migration
	firstMigrationState := s.captureStoreState(kvStore)

	// Verify migration marker exists
	migrationMarker := []byte("v13_wasm_migrated")
	require.True(s.T(), kvStore.Has(migrationMarker), "Migration marker should exist")
	require.Equal(s.T(), []byte("true"), kvStore.Get(migrationMarker), "Migration marker should be 'true'")

	// Run migration second time (retry scenario)
	err = v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit changes
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Capture state after second migration
	secondMigrationState := s.captureStoreState(kvStore)

	// Verify states are identical (idempotent)
	s.compareStoreStates(firstMigrationState, secondMigrationState, "First and second migration should be identical")

	// Run migration third time to be extra sure
	err = v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	thirdMigrationState := s.captureStoreState(kvStore)
	s.compareStoreStates(firstMigrationState, thirdMigrationState, "First and third migration should be identical")

	fmt.Printf("Migration idempotency test passed - %d retries produced identical results\n", 3)
}

// TestPartialMigrationRecovery simulates recovery from a partial migration
func (s *MigrationRetryTestSuite) TestPartialMigrationRecovery() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup test data
	s.setupTestData(kvStore)

	// Simulate partial migration: manually migrate some sequence keys but not others
	// This simulates a crash during migration
	newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
	kvStore.Set(newCodeIDKey, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Leave instance ID unmigrated to simulate partial state
	// Also leave contract keys partially migrated
	kvStore.Set(append([]byte{0x02}, bytes.Repeat([]byte{0xAA}, 20)...), []byte("partially-migrated-contract"))

	// Now run the full migration - it should recover gracefully
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit and verify
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify migration completed successfully
	migrationMarker := []byte("v13_wasm_migrated")
	require.True(s.T(), kvStore.Has(migrationMarker), "Migration should complete even after partial state")

	// Verify sequence keys are correctly migrated
	newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
	require.True(s.T(), kvStore.Has(newCodeIDKey), "Code ID sequence should exist")
	require.True(s.T(), kvStore.Has(newInstanceIDKey), "Instance ID sequence should exist")

	// Verify old sequence keys are cleaned up
	require.False(s.T(), kvStore.Has([]byte{0x01}), "Old code ID key should be deleted")
	require.False(s.T(), kvStore.Has([]byte{0x02}), "Old instance ID key should be deleted")

	fmt.Printf("Partial migration recovery test passed\n")
}

// TestMigrationWithPreExistingData tests migration when destination prefixes have data
func (s *MigrationRetryTestSuite) TestMigrationWithPreExistingData() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup source data
	s.setupTestData(kvStore)

	// Pre-populate destination prefixes with some data
	// This simulates what could happen during a retry or upgrade rollback scenario

	// Add some data to destination prefixes
	kvStore.Set(append([]byte{0x01}, []byte{0x99}...), []byte("pre-existing-code"))
	kvStore.Set(append([]byte{0x02}, []byte{0x99}...), []byte("pre-existing-contract"))
	kvStore.Set(append([]byte{0x03}, []byte{0x99}...), []byte("pre-existing-store"))

	// Also add sequence keys
	newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
	newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)
	kvStore.Set(newCodeIDKey, []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	kvStore.Set(newInstanceIDKey, []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Run migration - should handle pre-existing data gracefully
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Verify migration completed
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	migrationMarker := []byte("v13_wasm_migrated")
	require.True(s.T(), kvStore.Has(migrationMarker), "Migration should complete with pre-existing data")

	// Verify pre-existing data is preserved
	require.Equal(s.T(), []byte("pre-existing-code"), kvStore.Get(append([]byte{0x01}, []byte{0x99}...)))
	require.Equal(s.T(), []byte("pre-existing-contract"), kvStore.Get(append([]byte{0x02}, []byte{0x99}...)))
	require.Equal(s.T(), []byte("pre-existing-store"), kvStore.Get(append([]byte{0x03}, []byte{0x99}...)))

	fmt.Printf("Migration with pre-existing data test passed\n")
}

// TestSequenceKeyValidation tests the sequence key validation logic
func (s *MigrationRetryTestSuite) TestSequenceKeyValidation() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Test case 1: Valid sequence keys
	kvStore.Set([]byte{0x01}, []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	kvStore.Set([]byte{0x02}, []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Verify migration worked
	newCodeIDKey := append([]byte{0x04}, []byte("lastCodeId")...)
	newInstanceIDKey := append([]byte{0x04}, []byte("lastContractId")...)

	require.Equal(s.T(), []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, kvStore.Get(newCodeIDKey))
	require.Equal(s.T(), []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, kvStore.Get(newInstanceIDKey))

	fmt.Printf("Sequence key validation test passed\n")
}

// TestCollisionDetectionDuringRetry tests that collision detection works during retries
func (s *MigrationRetryTestSuite) TestCollisionDetectionDuringRetry() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup conflicting scenario
	addr := bytes.Repeat([]byte{0xAA}, 20)

	// Add data to old location
	kvStore.Set(append([]byte{0x04}, addr...), []byte("original-data"))

	// Add conflicting data to new location
	kvStore.Set(append([]byte{0x02}, addr...), []byte("conflicting-data"))

	// Add other required data
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Migration should detect the collision and fail or handle it safely
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)

	// Depending on implementation, this might fail (which is safe) or handle the collision
	// The important thing is that it doesn't silently corrupt data
	if err != nil {
		require.Contains(s.T(), err.Error(), "overwrite", "Error should mention collision/overwrite protection")
		fmt.Printf("Collision detection working - migration safely failed: %v\n", err)
	} else {
		// If migration succeeded, verify no data was corrupted
		// The collision guard should have prevented overwriting
		currentValue := kvStore.Get(append([]byte{0x02}, addr...))
		require.Equal(s.T(), []byte("conflicting-data"), currentValue, "Collision guard should prevent overwrite")
		fmt.Printf("Collision detection working - existing data preserved\n")
	}
}

// Helper functions

func (s *MigrationRetryTestSuite) setupTestData(kvStore sdk.KVStore) {
	// Sequence keys
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Code keys
	kvStore.Set([]byte{0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, []byte("code1"))

	// Contract keys (mix of regular and length-prefixed)
	addr1 := bytes.Repeat([]byte{0xAA}, 20)
	kvStore.Set(append([]byte{0x04}, addr1...), []byte("contract1"))

	legitimateAddr := append([]byte{20}, bytes.Repeat([]byte{0xBB}, 20)...)
	kvStore.Set(append([]byte{0x04}, legitimateAddr...), []byte("contract2"))
	_ = legitimateAddr // Keep for potential future use

	// Contract store keys
	kvStore.Set(append(append([]byte{0x05}, addr1...), []byte{0x01}...), []byte("store1"))

	// Contract history keys
	kvStore.Set(append([]byte{0x06}, addr1...), []byte("history1"))

	// Secondary index keys
	kvStore.Set([]byte{0x10, 0x01, 0x00}, []byte("index1"))

	// Params key
	kvStore.Set([]byte{0x11}, []byte("params"))
}

func (s *MigrationRetryTestSuite) captureStoreState(kvStore sdk.KVStore) map[string][]byte {
	state := make(map[string][]byte)

	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		state[key] = value
	}

	return state
}

func (s *MigrationRetryTestSuite) compareStoreStates(state1, state2 map[string][]byte, message string) {
	// Check that all keys from state1 exist in state2 with same values
	for key, value1 := range state1 {
		value2, exists := state2[key]
		require.True(s.T(), exists, "Key %X should exist in second state", key)
		require.Equal(s.T(), value1, value2, "Values should match for key %X", key)
	}

	// Check that state2 doesn't have extra keys
	for key := range state2 {
		_, exists := state1[key]
		require.True(s.T(), exists, "Second state should not have extra key %X", key)
	}

	require.Equal(s.T(), len(state1), len(state2), "%s: States should have same number of entries", message)
}
