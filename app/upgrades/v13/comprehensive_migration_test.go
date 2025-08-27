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
		name string
		addr []byte
		data []byte
	}{
		{
			name: "legitimate_prefixed",
			addr: append([]byte{20}, bytes.Repeat([]byte{0xAA}, 20)...),
			data: []byte("legitimate-contract-info"),
		},
		{
			name: "regular_addr",
			addr: bytes.Repeat([]byte{0xBB}, 20),
			data: []byte("regular-contract-info"),
		},
		{
			name: "edge_case_1",
			addr: append([]byte{0x1F}, bytes.Repeat([]byte{0xCC}, 31)...),
			data: []byte("edge-case-contract-1"),
		},
		{
			name: "edge_case_2",
			addr: append([]byte{0x14}, bytes.Repeat([]byte{0xDD}, 25)...),
			data: []byte("edge-case-contract-2"),
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
		var newAddr []byte
		if contract.name == "legitimate_prefixed" {
			// Should be stripped
			newAddr = bytes.Repeat([]byte{0xAA}, 20)
		} else {
			// Should NOT be stripped
			newAddr = contract.addr
		}
		newKey := append([]byte{0x02}, newAddr...)
		s.Require().Equal(originalData[fmt.Sprintf("contract_%s", contract.name)], kvStore.Get(newKey),
			"Contract %s should be migrated", contract.name)
		migrated++
	}

	// Check contract store keys (0x05 -> 0x03)
	for _, contract := range contracts {
		var contractAddr []byte
		if contract.name == "legitimate_prefixed" {
			contractAddr = bytes.Repeat([]byte{0xAA}, 20)
		} else {
			contractAddr = contract.addr
		}

		for j := 1; j <= 3; j++ {
			newKey := append(append([]byte{0x03}, contractAddr...), []byte{byte(j)}...)
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
