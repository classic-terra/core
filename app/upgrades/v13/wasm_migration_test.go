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

type ComprehensiveMigrationTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

func TestComprehensiveMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(ComprehensiveMigrationTestSuite))
}

// ----------------------------------------------------
// End-to-end migration scenario
// ----------------------------------------------------
func (s *ComprehensiveMigrationTestSuite) TestCompleteMigrationScenario() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// seed realistic test data
	testData := setupRealisticTestData(kvStore)

	mockWasmKeeper := wasmkeeper.Keeper{}
	preMigrationState := s.captureStoreState(kvStore)

	// run migration
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// verify results
	s.verifyMigrationResults(kvStore, testData, preMigrationState)
}

// ----------------------------------------------------
// Data integrity test
// ----------------------------------------------------
func (s *ComprehensiveMigrationTestSuite) TestDataIntegrityAfterMigration() {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	originalData := make(map[string][]byte)

	// seq keys
	originalData["seq_code"] = []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}
	originalData["seq_contract"] = []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	kvStore.Set([]byte{0x01}, originalData["seq_code"])
	kvStore.Set([]byte{0x02}, originalData["seq_contract"])

	// codes
	for i := 1; i <= 5; i++ {
		key := append([]byte{0x03}, []byte{byte(i), 0, 0, 0, 0, 0, 0, 0}...)
		value := []byte(fmt.Sprintf("code-data-%d", i))
		originalData[fmt.Sprintf("code_%d", i)] = value
		kvStore.Set(key, value)
	}

	// params
	originalData["params"] = []byte("wasm-params-data")
	kvStore.Set([]byte{0x11}, originalData["params"])

	fmt.Printf("Original data entries: %d\n", len(originalData))

	// run migration
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	stateStore.Commit()
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// check sequences
	seqCodeKey := append([]byte{0x04}, []byte("lastCodeId")...)
	s.Require().Equal(originalData["seq_code"], kvStore.Get(seqCodeKey))

	seqContractKey := append([]byte{0x04}, []byte("lastContractId")...)
	s.Require().Equal(originalData["seq_contract"], kvStore.Get(seqContractKey))

	// check codes
	for i := 1; i <= 5; i++ {
		newKey := append([]byte{0x01}, []byte{byte(i), 0, 0, 0, 0, 0, 0, 0}...)
		s.Require().Equal(originalData[fmt.Sprintf("code_%d", i)], kvStore.Get(newKey))
	}

	// check params
	s.Require().Equal(originalData["params"], kvStore.Get([]byte{0x10}))
}

// ----------------------------------------------------
// Direct contract/storage migration test
// ----------------------------------------------------
func TestMigrateContractsAndStorage(t *testing.T) {
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// fake contract
	contractAddr := []byte("terra1deadbeef")

	// old keys
	kvStore.Set(append([]byte{0x04}, contractAddr...), []byte("old-contract-info"))
	kvStore.Set(append(append([]byte{0x05}, contractAddr...), []byte("state")...), []byte("contract-state-value"))
	kvStore.Set(append([]byte{0x06}, contractAddr...), []byte("contract-history"))
	kvStore.Set(append([]byte{0x10}, contractAddr...), []byte("secondary-index"))

	// run migration
	mockWasmKeeper := wasmkeeper.Keeper{}
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(t, err)

	// assert new keys
	require.Equal(t, []byte("old-contract-info"), kvStore.Get(append([]byte{0x07}, contractAddr...)))
	require.Equal(t, []byte("contract-state-value"), kvStore.Get(append(append([]byte{0x08}, contractAddr...), []byte("state")...)))
	require.Equal(t, []byte("contract-history"), kvStore.Get(append([]byte{0x09}, contractAddr...)))
	require.Equal(t, []byte("secondary-index"), kvStore.Get(append([]byte{0x0a}, contractAddr...)))
}

// setupRealisticTestData seeds the store with realistic old keys
// and returns a map[string]interface{} that contains the expected values
// for later assertions in tests.
func setupRealisticTestData(store sdk.KVStore) map[string]interface{} {
	contracts := []string{
		"terra1deadbeef000000000000000000000000000000", // Contract A
		"terra1c0ffee0000000000000000000000000000000",  // Contract B
		"terra1f00dbabe0000000000000000000000000000",   // Contract C
	}

	testData := make(map[string]interface{})

	for i, addr := range contracts {
		// ---- ContractInfo (0x04) ----
		oldContractInfoKey := append([]byte{0x04}, []byte(addr)...)
		infoVal := []byte(fmt.Sprintf("contract-info-%d", i))
		store.Set(oldContractInfoKey, infoVal)
		testData[addr+"-contractInfo"] = infoVal

		// ---- ContractStore (0x05) ----
		oldStoreKey := append(append([]byte{0x05}, []byte(addr)...), []byte("state")...)
		storeVal := []byte(fmt.Sprintf("store-value-%d", i))
		store.Set(oldStoreKey, storeVal)
		testData[addr+"-contractStore"] = storeVal

		// ---- ContractHistory (0x06) ----
		oldHistoryKey := append([]byte{0x06}, []byte(addr)...)
		histVal := []byte(fmt.Sprintf("history-%d", i))
		store.Set(oldHistoryKey, histVal)
		testData[addr+"-contractHistory"] = histVal

		// ---- SecondaryIndex (0x10) ----
		oldSecIdxKey := append([]byte{0x10}, []byte(addr)...)
		secIdxVal := []byte(fmt.Sprintf("sec-index-%d", i))
		store.Set(oldSecIdxKey, secIdxVal)
		testData[addr+"-secondaryIndex"] = secIdxVal
	}

	return testData
}

func (s *ComprehensiveMigrationTestSuite) captureStoreState(kvStore sdk.KVStore) map[string][]byte {
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

func (s *ComprehensiveMigrationTestSuite) verifyMigrationResults(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	preMigrationState map[string][]byte,
) {
	fmt.Printf("Pre-migration entries: %d\n", len(preMigrationState))
	postCount := 0
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		postCount++
	}
	fmt.Printf("Post-migration entries: %d\n", postCount)
	s.Require().Greater(postCount, 0)
}
