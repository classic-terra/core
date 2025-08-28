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

func setupRealisticTestData(store sdk.KVStore) map[string]interface{} {
	sdk.GetConfig().SetAddressVerifier(wasmtypes.VerifyAddressLen())
	sdk.GetConfig().SetBech32PrefixForAccount("terra", "terrapub")

	// Generate valid SDK addresses
	addr1 := sdk.MustAccAddressFromBech32("terra1fex9f78reuwhfsnc8sun6mz8rl9zwqh03fhwf3") // 20 bytes
	addr2 := sdk.MustAccAddressFromBech32("terra1k4zsjshs2ukv959mfwnrlq68rmqm8xesd9dj6l") // 20 bytes
	addr3 := sdk.MustAccAddressFromBech32("terra1cf3dvu8jxaam2v92032exeuqe3ch5t8u72uzp0") // 20 bytes

	contracts := []sdk.AccAddress{addr1, addr2, addr3}
	testData := make(map[string]interface{})

	// Add sequence keys to test data
	seqCodeValue := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}
	seqContractValue := []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}

	store.Set([]byte{0x01}, seqCodeValue)
	store.Set([]byte{0x02}, seqContractValue)
	testData["seq_code"] = seqCodeValue
	testData["seq_contract"] = seqContractValue

	// Add some code entries
	for i := 1; i <= 3; i++ {
		codeKey := append([]byte{0x03}, []byte{byte(i), 0, 0, 0, 0, 0, 0, 0}...)
		codeValue := []byte(fmt.Sprintf("code-data-%d", i))
		store.Set(codeKey, codeValue)
		testData[fmt.Sprintf("code_%d", i)] = codeValue
	}

	// Add params
	paramsValue := []byte("wasm-params-data")
	store.Set([]byte{0x11}, paramsValue)
	testData["params"] = paramsValue

	for i, addr := range contracts {
		addrBytes := addr.Bytes()
		fmt.Printf("Addr length %v", len(addrBytes))

		// Verify this is a valid address
		if err := sdk.VerifyAddressFormat(addrBytes); err != nil {
			panic(fmt.Sprintf("Generated address doesn't pass validation: %v", err))
		}

		// Create length-prefixed address (only used where actually needed)
		lengthPrefixedAddr := append([]byte{byte(len(addrBytes))}, addrBytes...)

		fmt.Printf("Address %d: %s\n", i, addr.String())
		fmt.Printf("Raw bytes: %X (len=%d)\n", addrBytes, len(addrBytes))
		fmt.Printf("Length-prefixed: %X (len=%d)\n", lengthPrefixedAddr, len(lengthPrefixedAddr))

		// ---- ContractInfo (0x04) with LENGTH PREFIX (uses GetContractAddressKey) ----
		oldContractInfoKey := append([]byte{0x04}, lengthPrefixedAddr...)
		infoVal := []byte(fmt.Sprintf("contract-info-%d", i))
		store.Set(oldContractInfoKey, infoVal)
		testData[fmt.Sprintf("contractInfo_%d", i)] = map[string]interface{}{
			"value":          infoVal,
			"originalAddr":   lengthPrefixedAddr,
			"unPrefixedAddr": addrBytes,
			"sdkAddr":        addr,
		}

		// ---- ContractStore (0x05) with LENGTH PREFIX (uses GetContractStorePrefix) ----
		oldStoreKey := append(append([]byte{0x05}, lengthPrefixedAddr...), []byte("state")...)
		storeVal := []byte(fmt.Sprintf("store-value-%d", i))
		store.Set(oldStoreKey, storeVal)
		testData[fmt.Sprintf("contractStore_%d", i)] = map[string]interface{}{
			"value":          storeVal,
			"originalAddr":   lengthPrefixedAddr,
			"unPrefixedAddr": addrBytes,
			"subkey":         []byte("state"),
			"sdkAddr":        addr,
		}

		// ---- ContractHistory (0x06) WITHOUT length prefix (direct address) ----
		oldHistoryKey := append([]byte{0x06}, addrBytes...) // NO length prefix
		histVal := []byte(fmt.Sprintf("history-%d", i))
		store.Set(oldHistoryKey, histVal)
		testData[fmt.Sprintf("contractHistory_%d", i)] = map[string]interface{}{
			"value":   histVal,
			"addr":    addrBytes, // Direct address, no prefix
			"sdkAddr": addr,
		}

		// ---- SecondaryIndex (0x10) WITHOUT length prefix (direct address) ----
		oldSecIdxKey := append([]byte{0x10}, addrBytes...) // NO length prefix
		secIdxVal := []byte(fmt.Sprintf("sec-index-%d", i))
		store.Set(oldSecIdxKey, secIdxVal)
		testData[fmt.Sprintf("secondaryIndex_%d", i)] = map[string]interface{}{
			"value":   secIdxVal,
			"addr":    addrBytes, // Direct address, no prefix
			"sdkAddr": addr,
		}

		// ---- ContractsByCreator (0x09) with LENGTH PREFIX (uses GetContractsByCreatorPrefix) ----
		// This one also uses length prefix according to the original code
		oldCreatorKey := append([]byte{0x09}, lengthPrefixedAddr...)
		creatorVal := []byte(fmt.Sprintf("creator-index-%d", i))
		store.Set(oldCreatorKey, creatorVal)
		testData[fmt.Sprintf("contractsByCreator_%d", i)] = map[string]interface{}{
			"value":          creatorVal,
			"originalAddr":   lengthPrefixedAddr,
			"unPrefixedAddr": addrBytes,
			"sdkAddr":        addr,
		}
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

	// Capture post-migration state
	postMigrationState := s.captureStoreState(kvStore)
	fmt.Printf("Post-migration entries: %d\n", len(postMigrationState))

	// Basic sanity check
	s.Require().Greater(len(postMigrationState), 0, "Post-migration store should not be empty")

	// Verify sequence key migrations
	s.verifySequenceKeyMigration(kvStore, testData)

	// Verify code migrations
	s.verifyCodeMigrations(kvStore, testData)

	// Verify contract-specific migrations
	s.verifyContractMigrations(kvStore, testData)

	// Verify params migration
	s.verifyParamsMigration(kvStore, testData)

	// Verify old keys are completely removed
	s.verifyOldKeysRemoved(kvStore)

	// Verify migration completion marker
	s.verifyMigrationMarker(kvStore)

	fmt.Println("✅ All migration verifications passed")
}

// verifySequenceKeyMigration checks sequence key migrations:
// seq(lastCodeId) 0x01 → 0x04/"lastCodeId"
// seq(lastContractId) 0x02 → 0x04/"lastContractId"
func (s *ComprehensiveMigrationTestSuite) verifySequenceKeyMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
) {
	fmt.Println("Verifying sequence key migrations...")

	// Check lastCodeId migration: 0x01 → 0x04/"lastCodeId"
	if expectedSeqCode, exists := testData["seq_code"]; exists {
		expectedBytes := expectedSeqCode.([]byte)
		newCodeSeqKey := append([]byte{0x04}, []byte("lastCodeId")...)
		actualCodeSeq := kvStore.Get(newCodeSeqKey)
		s.Require().Equal(expectedBytes, actualCodeSeq,
			"Code sequence migration failed: 0x01 → 0x04/\"lastCodeId\"")

		// Ensure old key is deleted
		oldCodeSeq := kvStore.Get([]byte{0x01})
		s.Require().Nil(oldCodeSeq, "Old code sequence key 0x01 should be deleted")

		fmt.Printf("✓ Code sequence migrated: 0x01 → 0x04/\"lastCodeId\"\n")
	}

	// Check lastContractId migration: 0x02 → 0x04/"lastContractId"
	if expectedSeqContract, exists := testData["seq_contract"]; exists {
		expectedBytes := expectedSeqContract.([]byte)
		newContractSeqKey := append([]byte{0x04}, []byte("lastContractId")...)
		actualContractSeq := kvStore.Get(newContractSeqKey)
		s.Require().Equal(expectedBytes, actualContractSeq,
			"Contract sequence migration failed: 0x02 → 0x04/\"lastContractId\"")

		// Ensure old key is deleted
		oldContractSeq := kvStore.Get([]byte{0x02})
		s.Require().Nil(oldContractSeq, "Old contract sequence key 0x02 should be deleted")

		fmt.Printf("✓ Contract sequence migrated: 0x02 → 0x04/\"lastContractId\"\n")
	}
}

// verifyCodeMigrations checks code key migrations: 0x03 → 0x01
func (s *ComprehensiveMigrationTestSuite) verifyCodeMigrations(
	kvStore sdk.KVStore,
	testData map[string]interface{},
) {
	fmt.Println("Verifying code key migrations...")

	for i := 1; i <= 3; i++ {
		codeKey := fmt.Sprintf("code_%d", i)
		if expectedCode, exists := testData[codeKey]; exists {
			expectedBytes := expectedCode.([]byte)

			// Original key was: 0x03 + [i, 0, 0, 0, 0, 0, 0, 0]
			// New key should be: 0x01 + [i, 0, 0, 0, 0, 0, 0, 0]
			newCodeKey := append([]byte{0x01}, []byte{byte(i), 0, 0, 0, 0, 0, 0, 0}...)
			actualCode := kvStore.Get(newCodeKey)
			s.Require().Equal(expectedBytes, actualCode,
				"Code migration failed for code %d: 0x03 → 0x01", i)

			fmt.Printf("✓ Code %d migrated: 0x03 → 0x01\n", i)
		}
	}
}

// verifyContractMigrations checks contract-specific migrations
func (s *ComprehensiveMigrationTestSuite) verifyContractMigrations(
	kvStore sdk.KVStore,
	testData map[string]interface{},
) {
	fmt.Println("Verifying contract-specific migrations...")

	for i := 0; i < 3; i++ {
		// Verify ContractInfo migration: 0x04 → 0x02
		s.verifyContractInfoMigration(kvStore, testData, i)

		// Verify ContractStore migration: 0x05 → 0x03
		s.verifyContractStoreMigration(kvStore, testData, i)

		// Verify ContractHistory migration: 0x06 → 0x05
		s.verifyContractHistoryMigration(kvStore, testData, i)

		// Verify SecondaryIndex migration: 0x10 → 0x06
		s.verifySecondaryIndexMigration(kvStore, testData, i)
	}
}

func (s *ComprehensiveMigrationTestSuite) verifyContractInfoMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	contractIndex int,
) {
	infoKey := fmt.Sprintf("contractInfo_%d", contractIndex)
	if infoData, exists := testData[infoKey]; exists {
		infoMap := infoData.(map[string]interface{})
		expectedValue := infoMap["value"].([]byte)
		unPrefixedAddr := infoMap["unPrefixedAddr"].([]byte)
		fmt.Printf("Verifying ContractInfo for contract %d with unprefixed address: %X\n", contractIndex, unPrefixedAddr)
		// After migration: 0x02 + unprefixed_address
		newInfoKey := append([]byte{0x02}, unPrefixedAddr...)
		actualValue := kvStore.Get(newInfoKey)
		s.Require().Equal(expectedValue, actualValue,
			"ContractInfo migration failed for contract %d: 0x04 → 0x02", contractIndex)

		fmt.Printf("✓ ContractInfo %d migrated: 0x04 → 0x02\n", contractIndex)
	}
}

func (s *ComprehensiveMigrationTestSuite) verifyContractStoreMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	contractIndex int,
) {
	storeKey := fmt.Sprintf("contractStore_%d", contractIndex)
	if storeData, exists := testData[storeKey]; exists {
		storeMap := storeData.(map[string]interface{})
		expectedValue := storeMap["value"].([]byte)
		unPrefixedAddr := storeMap["unPrefixedAddr"].([]byte)
		subkey := storeMap["subkey"].([]byte)

		// After migration: 0x03 + unprefixed_address + subkey
		newStoreKey := append(append([]byte{0x03}, unPrefixedAddr...), subkey...)
		actualValue := kvStore.Get(newStoreKey)
		s.Require().Equal(expectedValue, actualValue,
			"ContractStore migration failed for contract %d: 0x05 → 0x03", contractIndex)

		fmt.Printf("✓ ContractStore %d migrated: 0x05 → 0x03\n", contractIndex)
	}
}

// Updated verification functions to match corrected test data
func (s *ComprehensiveMigrationTestSuite) verifyContractHistoryMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	contractIndex int,
) {
	historyKey := fmt.Sprintf("contractHistory_%d", contractIndex)
	if historyData, exists := testData[historyKey]; exists {
		historyMap := historyData.(map[string]interface{})
		expectedValue := historyMap["value"].([]byte)
		addr := historyMap["addr"].([]byte) // Direct address, no prefix

		// After migration: 0x05 + direct_address (no prefix to strip)
		newHistoryKey := append([]byte{0x05}, addr...)
		actualValue := kvStore.Get(newHistoryKey)
		s.Require().Equal(expectedValue, actualValue,
			"ContractHistory migration failed for contract %d: 0x06 → 0x05", contractIndex)

		fmt.Printf("✓ ContractHistory %d migrated: 0x06 → 0x05\n", contractIndex)
	}
}

func (s *ComprehensiveMigrationTestSuite) verifySecondaryIndexMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	contractIndex int,
) {
	secIdxKey := fmt.Sprintf("secondaryIndex_%d", contractIndex)
	if secIdxData, exists := testData[secIdxKey]; exists {
		secIdxMap := secIdxData.(map[string]interface{})
		expectedValue := secIdxMap["value"].([]byte)
		addr := secIdxMap["addr"].([]byte) // Direct address, no prefix

		// After migration: 0x06 + direct_address (no prefix to strip)
		newSecIdxKey := append([]byte{0x06}, addr...)
		actualValue := kvStore.Get(newSecIdxKey)
		s.Require().Equal(expectedValue, actualValue,
			"SecondaryIndex migration failed for contract %d: 0x10 → 0x06", contractIndex)

		fmt.Printf("✓ SecondaryIndex %d migrated: 0x10 → 0x06\n", contractIndex)
	}
}

func (s *ComprehensiveMigrationTestSuite) verifyContractsByCreatorMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
	contractIndex int,
) {
	creatorKey := fmt.Sprintf("contractsByCreator_%d", contractIndex)
	if creatorData, exists := testData[creatorKey]; exists {
		creatorMap := creatorData.(map[string]interface{})
		expectedValue := creatorMap["value"].([]byte)
		unPrefixedAddr := creatorMap["unPrefixedAddr"].([]byte)

		// After migration: 0x07 + unprefixed_address (length prefix stripped)
		newCreatorKey := append([]byte{0x07}, unPrefixedAddr...)
		actualValue := kvStore.Get(newCreatorKey)
		s.Require().Equal(expectedValue, actualValue,
			"ContractsByCreator migration failed for contract %d: 0x09 → 0x07", contractIndex)

		fmt.Printf("✓ ContractsByCreator %d migrated: 0x09 → 0x07\n", contractIndex)
	}
}

// verifyParamsMigration checks params migration: 0x11 → 0x10
func (s *ComprehensiveMigrationTestSuite) verifyParamsMigration(
	kvStore sdk.KVStore,
	testData map[string]interface{},
) {
	fmt.Println("Verifying params migration...")

	if expectedParams, exists := testData["params"]; exists {
		expectedBytes := expectedParams.([]byte)
		newParamsKey := []byte{0x10}
		actualParams := kvStore.Get(newParamsKey)
		s.Require().Equal(expectedBytes, actualParams,
			"Params migration failed: 0x11 → 0x10")

		// Ensure old key is deleted
		oldParams := kvStore.Get([]byte{0x11})
		s.Require().Nil(oldParams, "Old params key 0x11 should be deleted")

		fmt.Printf("✓ Params migrated: 0x11 → 0x10\n")
	}
}

// verifyOldKeysRemoved ensures that old key prefixes are completely removed
func (s *ComprehensiveMigrationTestSuite) verifyOldKeysRemoved(kvStore sdk.KVStore) {
	fmt.Println("Verifying old keys are completely removed...")

	// Define old prefixes that should be completely EMPTY after migration
	oldPrefixesShouldBeEmpty := []byte{0x11}

	for _, prefix := range oldPrefixesShouldBeEmpty {
		iter := kvStore.Iterator([]byte{prefix}, []byte{prefix + 1})
		hasOldKeys := iter.Valid()
		if hasOldKeys {
			fmt.Printf("❌ Found remaining old keys with prefix 0x%02x:\n", prefix)
			for ; iter.Valid(); iter.Next() {
				fmt.Printf("  - Key: %X, Value: %X\n", iter.Key(), iter.Value())
			}
		}
		iter.Close()

		s.Require().False(hasOldKeys,
			"Found remaining old keys with prefix 0x%02x - migration incomplete", prefix)

		fmt.Printf("✓ All old keys with prefix 0x%02x removed\n", prefix)
	}
}

// verifyMigrationMarker checks that the migration completion marker exists
func (s *ComprehensiveMigrationTestSuite) verifyMigrationMarker(kvStore sdk.KVStore) {
	fmt.Println("Verifying migration completion marker...")

	migrationMarker := []byte(v13.WasmMigrationMarker)
	markerValue := kvStore.Get(migrationMarker)
	s.Require().NotNil(markerValue, "Migration completion marker should exist")
	s.Require().Equal([]byte("true"), markerValue, "Migration marker should have 'true' value")

	fmt.Printf("✓ Migration completion marker found: %s\n", v13.WasmMigrationMarker)
}
