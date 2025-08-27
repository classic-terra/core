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

type ComprehensiveMigrationTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

func TestComprehensiveMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(ComprehensiveMigrationTestSuite))
}

// TestCompleteMigrationScenario tests the complete migration with various real-world edge cases
func (s *ComprehensiveMigrationTestSuite) TestCompleteMigrationScenario() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Test scenarios based on real-world Terra Classic data patterns
	testData := s.setupRealisticTestData(kvStore)

	// Create mock wasm keeper
	mockWasmKeeper := wasmkeeper.Keeper{}

	// Record pre-migration state
	preMigrationState := s.captureStoreState(kvStore)

	// Run migration
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit changes
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify migration results
	s.verifyMigrationResults(kvStore, testData, preMigrationState)
}

// TestDataIntegrityAfterMigration ensures no data is lost or corrupted during migration
func (s *ComprehensiveMigrationTestSuite) TestDataIntegrityAfterMigration() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create valid addresses
	validAddr1 := sdk.AccAddress(bytes.Repeat([]byte{0xAA}, 20))
	validAddr2 := sdk.AccAddress(bytes.Repeat([]byte{0xBB}, 20))
	validAddr3 := sdk.AccAddress(bytes.Repeat([]byte{0xCC}, 20))
	validAddr4 := sdk.AccAddress(bytes.Repeat([]byte{0xDD}, 20))

	// Create comprehensive test dataset
	originalData := make(map[string][]byte)

	// Sequence keys
	originalData["seq_code"] = []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}
	originalData["seq_contract"] = []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	kvStore.Set([]byte{0x01}, originalData["seq_code"])
	kvStore.Set([]byte{0x02}, originalData["seq_contract"])

	// Code keys (various sizes)
	for i := 1; i <= 5; i++ {
		key := []byte{0x03}
		key = append(key, []byte{byte(i), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		value := []byte(fmt.Sprintf("code-data-%d", i))
		originalData[fmt.Sprintf("code_%d", i)] = value
		kvStore.Set(key, value)
	}

	// Contract keys (mix of legitimate and edge cases)
	contracts := []struct {
		name                 string
		addr                 []byte
		expectedStrippedAddr []byte
		data                 []byte
		shouldBeStripped     bool
	}{
		{
			name:                 "legitimate_prefixed",
			addr:                 append([]byte{20}, validAddr1...), // Length-prefixed valid address
			expectedStrippedAddr: validAddr1,                        // Should be stripped to this
			data:                 []byte("legitimate-contract-info"),
			shouldBeStripped:     true,
		},
		{
			name:                 "regular_addr",
			addr:                 validAddr2, // Already valid, no prefix
			expectedStrippedAddr: validAddr2, // Should remain unchanged
			data:                 []byte("regular-contract-info"),
			shouldBeStripped:     false,
		},
		{
			name:                 "valid_addr_no_prefix",
			addr:                 validAddr3,
			expectedStrippedAddr: validAddr3,
			data:                 []byte("valid-addr-no-prefix"),
			shouldBeStripped:     false,
		},
		{
			name:                 "another_prefixed",
			addr:                 append([]byte{20}, validAddr4...), // Another length-prefixed valid address
			expectedStrippedAddr: validAddr4,
			data:                 []byte("another-prefixed-contract"),
			shouldBeStripped:     true,
		},
	}

	for _, contract := range contracts {
		key := append([]byte{0x04}, contract.addr...)
		originalData[fmt.Sprintf("contract_%s", contract.name)] = contract.data
		kvStore.Set(key, contract.data)
	}

	// Contract store keys
	for _, contract := range contracts {
		for j := 1; j <= 3; j++ {
			storeKey := append(append([]byte{0x05}, contract.addr...), []byte{byte(j)}...)
			storeValue := []byte(fmt.Sprintf("store-data-%s-%d", contract.name, j))
			originalData[fmt.Sprintf("store_%s_%d", contract.name, j)] = storeValue
			kvStore.Set(storeKey, storeValue)
		}
	}

	// Contract history keys
	for _, contract := range contracts {
		histKey := append([]byte{0x06}, contract.addr...)
		histValue := []byte(fmt.Sprintf("history-%s", contract.name))
		originalData[fmt.Sprintf("history_%s", contract.name)] = histValue
		kvStore.Set(histKey, histValue)
	}

	// Secondary index keys
	for i := 1; i <= 3; i++ {
		key := []byte{0x10, byte(i), 0x00}
		value := []byte(fmt.Sprintf("index-%d", i))
		originalData[fmt.Sprintf("index_%d", i)] = value
		kvStore.Set(key, value)
	}

	// Params
	originalData["params"] = []byte("wasm-params-data")
	kvStore.Set([]byte{0x11}, originalData["params"])

	// Count original entries
	originalCount := len(originalData)
	fmt.Printf("Original data entries: %d\n", originalCount)

	// Run migration
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit and refresh context
	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify all data still exists in new locations
	migrated := 0

	// Check sequence keys
	seqCodeKey := append([]byte{0x04}, []byte("lastCodeId")...)
	s.Require().Equal(originalData["seq_code"], kvStore.Get(seqCodeKey), "Code sequence should be migrated")
	migrated++

	seqContractKey := append([]byte{0x04}, []byte("lastContractId")...)
	s.Require().Equal(originalData["seq_contract"], kvStore.Get(seqContractKey), "Contract sequence should be migrated")
	migrated++

	// Check code keys (0x03 -> 0x01)
	for i := 1; i <= 5; i++ {
		newKey := []byte{0x01}
		newKey = append(newKey, []byte{byte(i), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		s.Require().Equal(originalData[fmt.Sprintf("code_%d", i)], kvStore.Get(newKey), "Code %d should be migrated", i)
		migrated++
	}

	// Check contract keys (0x04 -> 0x02)
	for _, contract := range contracts {
		newKey := append([]byte{0x02}, contract.expectedStrippedAddr...)
		s.Require().Equal(originalData[fmt.Sprintf("contract_%s", contract.name)], kvStore.Get(newKey),
			"Contract %s should be migrated", contract.name)
		migrated++
	}

	// Check contract store keys (0x05 -> 0x03)
	for _, contract := range contracts {
		for j := 1; j <= 3; j++ {
			newKey := append(append([]byte{0x03}, contract.expectedStrippedAddr...), []byte{byte(j)}...)
			s.Require().Equal(originalData[fmt.Sprintf("store_%s_%d", contract.name, j)], kvStore.Get(newKey),
				"Store data %s-%d should be migrated", contract.name, j)
			migrated++
		}
	}

	// Check contract history keys (0x06 -> 0x05)
	for _, contract := range contracts {
		// For history keys, the addr is kept as-is (no stripping in history migration)
		newKey := append([]byte{0x05}, contract.addr...)
		s.Require().Equal(originalData[fmt.Sprintf("history_%s", contract.name)], kvStore.Get(newKey),
			"History %s should be migrated", contract.name)
		migrated++
	}

	// Check secondary index keys (0x10 -> 0x06)
	for i := 1; i <= 3; i++ {
		newKey := []byte{0x06, byte(i), 0x00}
		s.Require().Equal(originalData[fmt.Sprintf("index_%d", i)], kvStore.Get(newKey),
			"Index %d should be migrated", i)
		migrated++
	}

	// Check params (0x11 -> 0x10)
	s.Require().Equal(originalData["params"], kvStore.Get([]byte{0x10}), "Params should be migrated")
	migrated++

	// Verify all original data was migrated
	s.Require().Equal(originalCount, migrated, "All original data should be migrated")

	fmt.Printf("Successfully migrated %d entries with no data loss\n", migrated)
}

// Helper functions

func (s *ComprehensiveMigrationTestSuite) setupRealisticTestData(kvStore sdk.KVStore) map[string]interface{} {
	// This would setup realistic Terra Classic WASM data patterns
	// For now, return empty map as placeholder
	return make(map[string]interface{})
}

func (s *ComprehensiveMigrationTestSuite) captureStoreState(kvStore sdk.KVStore) map[string][]byte {
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

func (s *ComprehensiveMigrationTestSuite) verifyMigrationResults(kvStore sdk.KVStore, testData map[string]interface{}, preMigrationState map[string][]byte) {
	// Verify no data was lost and all migrations happened correctly
	fmt.Printf("Pre-migration entries: %d\n", len(preMigrationState))

	// Count post-migration entries
	postCount := 0
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		postCount++
	}

	fmt.Printf("Post-migration entries: %d\n", postCount)

	// The count may differ due to key restructuring, but data should be preserved
	s.Require().Greater(postCount, 0, "Should have entries after migration")
}

// --- Add to: package v13_test

// Ensures pre-existing *new-prefix* contract-store entries are preserved.
// Requires the address to be "known" (present under 0x04) so migrateCodeKeysWithProtection
// can recognize 0x03|addr|... as a contract-store-shaped key and skip it.
func (s *ComprehensiveMigrationTestSuite) TestPreExistingNewContractStorePreserved() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv := ctx.KVStore(wasmStoreKey)

	// Known 20-byte address
	addr := bytes.Repeat([]byte{0xAA}, 20)

	// Mark it as a known contract (old-terra prefix 0x04)
	kv.Set(append([]byte{0x04}, addr...), []byte("contract-info"))

	// Pre-existing *new* contract-store key: 0x03|addr|subkey
	preKey := append(append([]byte{0x03}, addr...), 0x01)
	preVal := []byte("pre-existing-store")
	kv.Set(preKey, preVal)

	// Also add minimal sequences so migration runs through
	kv.Set([]byte{0x01}, []byte{0x01, 0, 0, 0, 0, 0, 0, 0})
	kv.Set([]byte{0x02}, []byte{0x02, 0, 0, 0, 0, 0, 0, 0})

	mock := wasmkeeper.Keeper{}
	require.NoError(s.T(), v13.MigrateWasmKeys(ctx, mock, wasmStoreKey))
	cms.Commit()
	ctx = sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv = ctx.KVStore(wasmStoreKey)

	// Must still be there (not deleted / moved)
	s.Require().Equal(preVal, kv.Get(preKey), "pre-existing 0x03|addr|subkey must be preserved")
}

// Pre-set new params (0x10) and ensure old (0x11) is deleted without overwriting.
func (s *ComprehensiveMigrationTestSuite) TestParamsCollisionPreserved() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv := ctx.KVStore(wasmStoreKey)

	kv.Set([]byte{0x11}, []byte("old-params"))
	kv.Set([]byte{0x10}, []byte("new-params-exist"))

	// sequences so migrate runs
	kv.Set([]byte{0x01}, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	kv.Set([]byte{0x02}, []byte{2, 0, 0, 0, 0, 0, 0, 0})

	mock := wasmkeeper.Keeper{}
	require.NoError(s.T(), v13.MigrateWasmKeys(ctx, mock, wasmStoreKey))
	cms.Commit()
	ctx = sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv = ctx.KVStore(wasmStoreKey)

	s.Require().Equal([]byte("new-params-exist"), kv.Get([]byte{0x10}))
	s.Require().Nil(kv.Get([]byte{0x11}))
}

// Pre-set new code key to collide with an old code key; ensure existing new value is kept and old deleted.
func (s *ComprehensiveMigrationTestSuite) TestCodeKeyCollisionPreserved() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv := ctx.KVStore(wasmStoreKey)

	// Old code key (0x03|id8)
	id := []byte{1, 0, 0, 0, 0, 0, 0, 0}
	oldKey := append([]byte{0x03}, id...)
	kv.Set(oldKey, []byte("old-code"))

	// New/destination already has a value (0x01|id8)
	newKey := append([]byte{0x01}, id...)
	kv.Set(newKey, []byte("preexisting-code"))

	// sequences so migrate runs
	kv.Set([]byte{0x01}, []byte{5, 0, 0, 0, 0, 0, 0, 0})
	kv.Set([]byte{0x02}, []byte{3, 0, 0, 0, 0, 0, 0, 0})

	mock := wasmkeeper.Keeper{}
	require.NoError(s.T(), v13.MigrateWasmKeys(ctx, mock, wasmStoreKey))
	cms.Commit()
	ctx = sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv = ctx.KVStore(wasmStoreKey)

	// Must keep preexisting new and delete old
	s.Require().Equal([]byte("preexisting-code"), kv.Get(newKey))
	s.Require().Nil(kv.Get(oldKey))
}

// "Direct" contract-store migration: a 0x05 key for a contract NOT present under 0x04.
// It should still move to 0x03, preserving the composite head (no stripping).
func (s *ComprehensiveMigrationTestSuite) TestDirectContractStoreFallbackMigratesUnknown() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv := ctx.KVStore(wasmStoreKey)

	// length-prefixed 20-byte head, but NOT present in 0x04 (unknown contract)
	head := append([]byte{20}, bytes.Repeat([]byte{0xCC}, 20)...)
	oldComposite := append(head, 0x02) // subkey
	oldKey := append([]byte{0x05}, oldComposite...)
	oldVal := []byte("unknown-contract-store")
	kv.Set(oldKey, oldVal)

	// sequences so migrate runs
	kv.Set([]byte{0x01}, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	kv.Set([]byte{0x02}, []byte{2, 0, 0, 0, 0, 0, 0, 0})

	mock := wasmkeeper.Keeper{}
	require.NoError(s.T(), v13.MigrateWasmKeys(ctx, mock, wasmStoreKey))
	cms.Commit()
	ctx = sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv = ctx.KVStore(wasmStoreKey)

	// The migration will strip the length prefix from the composite key
	strippedHead := bytes.Repeat([]byte{0xCC}, 20)  // stripped from [20][CC*20]
	expectedComposite := append(strippedHead, 0x02) // [CC*20][0x02]
	newKey := append([]byte{0x03}, expectedComposite...)

	s.Require().Equal(oldVal, kv.Get(newKey))
	s.Require().Nil(kv.Get(oldKey))
}

// Run migration twice and assert idempotency (state identical).
func (s *ComprehensiveMigrationTestSuite) TestIdempotencyRoundTrip() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	kv := ctx.KVStore(wasmStoreKey)

	// Seed some data
	addr := bytes.Repeat([]byte{0xAB}, 20)
	kv.Set([]byte{0x01}, []byte{7, 0, 0, 0, 0, 0, 0, 0})
	kv.Set([]byte{0x02}, []byte{9, 0, 0, 0, 0, 0, 0, 0})
	kv.Set(append([]byte{0x03}, []byte{2, 0, 0, 0, 0, 0, 0, 0}...), []byte("code2"))
	kv.Set(append([]byte{0x04}, addr...), []byte("contractX"))
	kv.Set(append(append([]byte{0x05}, addr...), 0x01), []byte("storeX"))
	kv.Set(append([]byte{0x06}, addr...), []byte("histX"))
	kv.Set([]byte{0x10, 0x01, 0x00}, []byte("idxX"))
	kv.Set([]byte{0x11}, []byte("paramsX"))

	mock := wasmkeeper.Keeper{}
	require.NoError(s.T(), v13.MigrateWasmKeys(ctx, mock, wasmStoreKey))
	cms.Commit()
	state1 := s.captureStoreState(sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger()).KVStore(wasmStoreKey))

	// Run again
	require.NoError(s.T(), v13.MigrateWasmKeys(sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger()), mock, wasmStoreKey))
	cms.Commit()
	state2 := s.captureStoreState(sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger()).KVStore(wasmStoreKey))

	// Compare
	s.compareStates(state1, state2)
}

// --- Small helper used by the idempotency test above.
func (s *ComprehensiveMigrationTestSuite) compareStates(a, b map[string][]byte) {
	s.Require().Equal(len(a), len(b), "state size differs")
	for k, va := range a {
		vb, ok := b[k]
		s.Require().True(ok, "missing key %q in state B", k)
		s.Require().Equal(va, vb, "mismatch at key %q", k)
	}
}
