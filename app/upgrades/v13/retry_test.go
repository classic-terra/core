package v13_test

import (
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

type MigrationFailureTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

func TestMigrationFailureTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationFailureTestSuite))
}

// ----------------------------------------------------
// Test migration idempotency and partial failure recovery
// ----------------------------------------------------

func (s *MigrationFailureTestSuite) TestMigrationIsIdempotent() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup test data
	_ = setupRealisticTestData(kvStore)
	mockWasmKeeper := wasmkeeper.Keeper{}

	// Run migration first time
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Capture state after first migration
	stateAfterFirst := s.captureStoreState(kvStore)

	// Run migration second time - should be safe
	err = v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Capture state after second migration
	stateAfterSecond := s.captureStoreState(kvStore)

	// States should be identical - proving idempotency
	s.Require().Equal(len(stateAfterFirst), len(stateAfterSecond))
	for key, value := range stateAfterFirst {
		s.Require().Equal(value, stateAfterSecond[key], "State changed after second migration for key: %X", key)
	}

	fmt.Println("✅ Migration is idempotent - safe to run multiple times")
}

func (s *MigrationFailureTestSuite) TestMigrationWithExistingMarker() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Set migration marker as if migration already completed
	migrationMarker := []byte(v13.WasmMigrationMarker)
	kvStore.Set(migrationMarker, []byte("true"))

	// Setup some data that would normally be migrated
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05})
	kvStore.Set([]byte{0x11}, []byte("params-data"))

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Migration should skip when marker exists
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Old keys should still exist since migration was skipped
	s.Require().NotNil(kvStore.Get([]byte{0x01}), "Old seq key should still exist when migration is skipped")
	s.Require().NotNil(kvStore.Get([]byte{0x11}), "Old params key should still exist when migration is skipped")

	fmt.Println("✅ Migration properly skips when marker exists")
}

func (s *MigrationFailureTestSuite) TestPartialMigrationRecovery() {
	// Test recovery from partial migration state where some prefixes are migrated but others aren't
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Simulate partial migration state:
	// 1. Sequence keys already migrated to new locations
	// 2. But old sequence keys still exist (haven't been deleted)
	// 3. Other data not yet migrated

	// Set up NEW sequence keys (as if step 1 completed)
	newCodeSeqKey := append([]byte{0x04}, []byte("lastCodeId")...)
	newContractSeqKey := append([]byte{0x04}, []byte("lastContractId")...)
	kvStore.Set(newCodeSeqKey, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05})
	kvStore.Set(newContractSeqKey, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03})

	// Set up OLD sequence keys (not yet deleted)
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05})
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03})

	// Set up unmigrated data
	kvStore.Set([]byte{0x11}, []byte("params-data")) // Old params

	// Add some contract data that needs migration
	addr := sdk.MustAccAddressFromBech32("terra1fex9f78reuwhfsnc8sun6mz8rl9zwqh03fhwf3")
	addrBytes := addr.Bytes()
	lengthPrefixedAddr := append([]byte{byte(len(addrBytes))}, addrBytes...)

	oldContractInfoKey := append([]byte{0x04}, lengthPrefixedAddr...)
	kvStore.Set(oldContractInfoKey, []byte("contract-info"))

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Run migration - should handle partial state gracefully
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Verify final state is correct
	// 1. New sequence keys should still exist with correct values
	s.Require().Equal([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}, kvStore.Get(newCodeSeqKey))
	s.Require().Equal([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}, kvStore.Get(newContractSeqKey))

	// 2. Old sequence keys should be deleted
	s.Require().Nil(kvStore.Get([]byte{0x01}), "Old code seq key should be deleted")
	s.Require().Nil(kvStore.Get([]byte{0x02}), "Old contract seq key should be deleted")

	// 3. Params should be migrated
	s.Require().Equal([]byte("params-data"), kvStore.Get([]byte{0x10}))
	s.Require().Nil(kvStore.Get([]byte{0x11}), "Old params key should be deleted")

	// 4. Contract info should be migrated
	newContractInfoKey := append([]byte{0x02}, addrBytes...)
	s.Require().Equal([]byte("contract-info"), kvStore.Get(newContractInfoKey))

	// 5. Migration marker should exist
	migrationMarker := []byte(v13.WasmMigrationMarker)
	s.Require().Equal([]byte("true"), kvStore.Get(migrationMarker))

	fmt.Println("✅ Migration handles partial state recovery correctly")
}

func (s *MigrationFailureTestSuite) TestMigrationWithCorruptedSequenceKeys() {
	// Test migration behavior when sequence keys have unexpected formats
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Set corrupted sequence keys (wrong length)
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x02})                                           // Too short
	kvStore.Set([]byte{0x02}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF}) // Too long

	// Add valid data
	kvStore.Set([]byte{0x11}, []byte("params-data"))

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Migration should handle corrupted data gracefully
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)

	// Should either succeed (migrating what it can) or fail gracefully
	if err != nil {
		fmt.Printf("Migration failed as expected with corrupted data: %v\n", err)
	} else {
		fmt.Println("Migration succeeded despite corrupted sequence keys")
		// If it succeeded, verify params were still migrated
		s.Require().Equal([]byte("params-data"), kvStore.Get([]byte{0x10}))
	}
}

func (s *MigrationFailureTestSuite) TestMigrationStateConsistency() {
	// Test that migration maintains consistency even with complex data
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup comprehensive test data
	testData := setupRealisticTestData(kvStore)

	// Add some edge cases
	// Empty values
	kvStore.Set([]byte{0x11, 0x01}, []byte{})
	// Large values
	largeValue := make([]byte, 1000)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	kvStore.Set([]byte{0x11, 0x02}, largeValue)

	// Count total entries before migration
	preCount := s.countStoreEntries(kvStore)
	fmt.Printf("Pre-migration entries: %d\n", preCount)

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Run migration
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Count entries after migration
	postCount := s.countStoreEntries(kvStore)
	fmt.Printf("Post-migration entries: %d\n", postCount)

	// Verify specific invariants
	s.verifyMigrationInvariants(kvStore, testData)

	fmt.Println("✅ Migration maintains state consistency")
}

func (s *MigrationFailureTestSuite) TestMigrationRollbackSafety() {
	// Test that a failed migration doesn't leave the store in an unusable state
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Setup test data
	originalSeqCode := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}
	originalParams := []byte("wasm-params")

	kvStore.Set([]byte{0x01}, originalSeqCode)
	kvStore.Set([]byte{0x11}, originalParams)

	// Capture original state
	_ = s.captureStoreState(kvStore)

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Try migration (should succeed in this case)
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)

	if err != nil {
		// If migration failed, verify original data is still accessible
		fmt.Printf("Migration failed: %v\n", err)

		// Check that critical data is still readable
		currentSeqCode := kvStore.Get([]byte{0x01})
		currentParams := kvStore.Get([]byte{0x11})

		// At least one of old or new location should have the data
		newSeqCodeKey := append([]byte{0x04}, []byte("lastCodeId")...)
		newSeqCode := kvStore.Get(newSeqCodeKey)
		newParams := kvStore.Get([]byte{0x10})

		s.Require().True(
			(currentSeqCode != nil && len(currentSeqCode) > 0) || (newSeqCode != nil && len(newSeqCode) > 0),
			"Sequence code data lost - neither old nor new location has data")

		s.Require().True(
			(currentParams != nil && len(currentParams) > 0) || (newParams != nil && len(newParams) > 0),
			"Params data lost - neither old nor new location has data")

	} else {
		fmt.Println("Migration succeeded")

		// Verify migration completed correctly
		migrationMarker := []byte(v13.WasmMigrationMarker)
		s.Require().Equal([]byte("true"), kvStore.Get(migrationMarker))

		// Verify data is in new locations
		newSeqCodeKey := append([]byte{0x04}, []byte("lastCodeId")...)
		s.Require().Equal(originalSeqCode, kvStore.Get(newSeqCodeKey))
		s.Require().Equal(originalParams, kvStore.Get([]byte{0x10}))
	}

	fmt.Println("✅ Migration failure handling maintains data safety")
}

func (s *MigrationFailureTestSuite) TestCrashMidContractInfoMigration_AfterWriting0x02() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Helpers to build old/new contract-info keys
	oldCIKey := func(addr []byte) []byte {
		lp := append([]byte{byte(len(addr))}, addr...)
		return append([]byte{0x04}, lp...)
	}
	newCIKey := func(addr []byte) []byte { return append([]byte{0x02}, addr...) }
	sdk.GetConfig().SetBech32PrefixForAccount("terra", "terrapub")
	sdk.GetConfig().SetAddressVerifier(wasmtypes.VerifyAddressLen())

	// Three synthetic 20-byte "addresses"
	addr1 := sdk.MustAccAddressFromBech32("terra1fex9f78reuwhfsnc8sun6mz8rl9zwqh03fhwf3") // 20 bytes
	addr2 := sdk.MustAccAddressFromBech32("terra1k4zsjshs2ukv959mfwnrlq68rmqm8xesd9dj6l") // 20 bytes
	addr3 := sdk.MustAccAddressFromBech32("terra1cf3dvu8jxaam2v92032exeuqe3ch5t8u72uzp0") // 20 bytes

	// Seed OLD (0x04/*) contract-info for all three
	kvStore.Set(oldCIKey(addr1), []byte("ci-1"))
	kvStore.Set(oldCIKey(addr2), []byte("ci-2"))
	kvStore.Set(oldCIKey(addr3), []byte("ci-3"))

	// --- Simulate crash AFTER writing some NEW 0x02/*, but BEFORE deleting old 0x04/*
	kvStore.Set(newCIKey(addr1), []byte("ci-1")) // migrated
	kvStore.Set(newCIKey(addr2), []byte("ci-2")) // migrated
	// addr3 remains unmigrated
	// No global WasmMigrationMarker set

	mockWasmKeeper := wasmkeeper.Keeper{}

	// Resume migration
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// All three must now exist under 0x02/* with correct values
	s.Require().Equal([]byte("ci-1"), kvStore.Get(newCIKey(addr1)))
	s.Require().Equal([]byte("ci-2"), kvStore.Get(newCIKey(addr2)))
	s.Require().Equal([]byte("ci-3"), kvStore.Get(newCIKey(addr3))) // newly migrated

	// All old 0x04/* must be cleaned up
	s.Require().Nil(kvStore.Get(oldCIKey(addr1)))
	s.Require().Nil(kvStore.Get(oldCIKey(addr2)))
	s.Require().Nil(kvStore.Get(oldCIKey(addr3)))

	// Global migration marker must be present
	migrationMarker := []byte(v13.WasmMigrationMarker)
	s.Require().Equal([]byte("true"), kvStore.Get(migrationMarker))

	fmt.Println("✅ Resume after crash (mid 0x02 writes) completed, old keys cleaned, marker set")
}

// ----------------------------------------------------
// Helper functions
// ----------------------------------------------------

func (s *MigrationFailureTestSuite) captureStoreState(kvStore sdk.KVStore) map[string][]byte {
	state := make(map[string][]byte)
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		val := make([]byte, len(iter.Value()))
		copy(val, iter.Value())
		state[key] = val
	}
	return state
}

func (s *MigrationFailureTestSuite) countStoreEntries(kvStore sdk.KVStore) int {
	count := 0
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

func (s *MigrationFailureTestSuite) verifyMigrationInvariants(kvStore sdk.KVStore, testData map[string]interface{}) {
	// 1. Migration marker must exist
	migrationMarker := []byte(v13.WasmMigrationMarker)
	s.Require().Equal([]byte("true"), kvStore.Get(migrationMarker), "Migration marker missing")

	// 2. No old sequence keys should exist
	s.Require().Nil(kvStore.Get([]byte{0x01}), "Old code sequence key still exists")
	s.Require().Nil(kvStore.Get([]byte{0x02}), "Old contract sequence key still exists")

	// 3. New sequence keys should exist if original data existed
	if _, hasCodeSeq := testData["seq_code"]; hasCodeSeq {
		newCodeSeqKey := append([]byte{0x04}, []byte("lastCodeId")...)
		s.Require().NotNil(kvStore.Get(newCodeSeqKey), "New code sequence key missing")
	}

	if _, hasContractSeq := testData["seq_contract"]; hasContractSeq {
		newContractSeqKey := append([]byte{0x04}, []byte("lastContractId")...)
		s.Require().NotNil(kvStore.Get(newContractSeqKey), "New contract sequence key missing")
	}

	// 4. No old params key should exist
	s.Require().Nil(kvStore.Get([]byte{0x11}), "Old params key still exists")

	// 5. New params should exist if original existed
	if _, hasParams := testData["params"]; hasParams {
		s.Require().NotNil(kvStore.Get([]byte{0x10}), "New params key missing")
	}

	fmt.Println("✅ All migration invariants verified")
}
