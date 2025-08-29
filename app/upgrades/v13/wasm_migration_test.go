package v13_test

import (
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

	// Common test data
	ctx          sdk.Context
	kvStore      sdk.KVStore
	wasmStoreKey *storetypes.KVStoreKey

	// Common test addresses
	testAddr1 sdk.AccAddress
	testAddr2 sdk.AccAddress
	testAddr3 sdk.AccAddress
}

func TestComprehensiveMigrationTestSuite(t *testing.T) {
	sdk.GetConfig().SetAddressVerifier(wasmtypes.VerifyAddressLen())
	sdk.GetConfig().SetBech32PrefixForAccount("terra", "terrapub")

	suite.Run(t, new(ComprehensiveMigrationTestSuite))
}

func (s *ComprehensiveMigrationTestSuite) SetupTest() {
	// Initialize test addresses
	s.testAddr1 = sdk.MustAccAddressFromBech32("terra1fex9f78reuwhfsnc8sun6mz8rl9zwqh03fhwf3")
	s.testAddr2 = sdk.MustAccAddressFromBech32("terra1k4zsjshs2ukv959mfwnrlq68rmqm8xesd9dj6l")
	s.testAddr3 = sdk.MustAccAddressFromBech32("terra1cf3dvu8jxaam2v92032exeuqe3ch5t8u72uzp0")

	// Setup common store infrastructure
	s.setupStore()
}

func (s *ComprehensiveMigrationTestSuite) setupStore() {
	db := dbm.NewMemDB()
	s.wasmStoreKey = sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(s.wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	s.kvStore = s.ctx.KVStore(s.wasmStoreKey)
}

func (s *ComprehensiveMigrationTestSuite) runMigration() {
	require.NoError(s.T(), v13.MigrateWasmKeys(s.ctx, wasmkeeper.Keeper{}, s.wasmStoreKey))
}

// TestKeySequenceCodeID tests the migration of key sequence code IDs.
// It changes the prefix from 0x01 to 0x04+"lastCodeId"
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L47
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L27
func (s *ComprehensiveMigrationTestSuite) TestKeySequenceCodeID() {
	oldKey := v13.LegacyPrefixes.KeySequenceCodeID
	oldVal := []byte{0x10}
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.KeySequenceCodeID
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
	s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestKeySequenceInstanceID tests the migration of key sequence instance IDs.
// It changes the prefix from 0x02 to 0x04+"lastContractId"
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L28C2-L28C23
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L38
func (s *ComprehensiveMigrationTestSuite) TestKeySequenceInstanceID() {
	oldKey := v13.LegacyPrefixes.KeySequenceInstanceID
	oldVal := []byte{0x10}
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.KeySequenceInstanceID
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))

	// Don't delete key at 0x02 because we store the contract keys here
	// s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestContractKeyMigration_LengthPrefixed tests the migration of contract info keys.
// It changes the prefix from 0x04 to 0x02 and address is changed from prefixed to unprefixed
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L47
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L47
func (s *ComprehensiveMigrationTestSuite) TestContractKeyMigration_LengthPrefixed() {
	oldKey := v13.GetContractAddressKeyLegacy(s.testAddr1)
	oldVal := []byte("contract-info-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.GetContractAddressKey(s.testAddr1)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
	s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestContractStoreMigration_LengthPrefixed tests the migration of contract store keys.
// The prefix is changed from 0x05 to 0x03, and address is changed from prefixed to unprefixed
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L58
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L77
func (s *ComprehensiveMigrationTestSuite) TestContractStoreMigration_LengthPrefixed() {
	oldKey := v13.GetContractStorePrefixLegacy(s.testAddr2)
	oldVal := []byte("contract-store-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.GetContractStorePrefix(s.testAddr2)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
	s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestContractHistoryMigration_Direct test the migration of contract history keys.
// It changes the prefix from 0x06 to 0x05
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L110
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L109
func (s *ComprehensiveMigrationTestSuite) TestContractHistoryMigration_Direct() {
	oldKey := v13.GetContractCodeHistoryElementKeyLegacy(s.testAddr3, 1)
	oldVal := []byte("history-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.GetContractCodeHistoryElementKey(s.testAddr3, 1)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
	s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestSecondaryIndexMigration_Direct tests the migration of secondary index keys.
// It changes the prefix from 0x10 to 0x06, and address still un-prefixed
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L77
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L76
func (s *ComprehensiveMigrationTestSuite) TestSecondaryIndexMigration_Direct() {
	contractCodeHistoryEntry := wasmtypes.ContractCodeHistoryEntry{
		CodeID: 42,
		Updated: &wasmtypes.AbsoluteTxPosition{
			BlockHeight: 10,
			TxIndex:     10,
		},
	}

	oldKey := v13.GetContractByCreatedSecondaryIndexKeyLegacy(s.testAddr2, contractCodeHistoryEntry)
	oldVal := []byte("sec-index-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	newKey := wasmtypes.GetContractByCreatedSecondaryIndexKey(s.testAddr2, contractCodeHistoryEntry)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
	s.Require().Nil(s.kvStore.Get(oldKey))
}

// TestPinnedCodeIndexMigration tests the migration of the pinned code indexes
// It stays the same after migration
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L32C2-L32C23
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L33
func (s *ComprehensiveMigrationTestSuite) TestPinnedCodeIndexMigration() {
	codeID := uint64(10)

	// Use legacy prefix
	oldKey := v13.GetPinnedCodeIndexPrefixLegacy(codeID)
	oldVal := []byte("creator-index-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	// Use new prefix
	newKey := wasmtypes.GetPinnedCodeIndexPrefix(codeID)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
}

// TestTxCounterPrefixMigration tests the migration of the pinned code indexes
// It stays the same after migration
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L33C2-L33C17
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L34
func (s *ComprehensiveMigrationTestSuite) TestTxCounterPrefixMigration() {
	// Use legacy prefix
	oldKey := v13.LegacyPrefixes.TXCounterPrefix
	oldVal := []byte{10}
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	// Use new prefix
	newKey := wasmtypes.TXCounterPrefix
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
}

// TestContractsByCreatorMigration_LengthPrefixed tests the migration of contracts by creator keys.
// It stays the same after migration.
// https://github.com/CosmWasm/wasmd/blob/v0.46.0/x/wasm/types/keys.go#L52
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L52
func (s *ComprehensiveMigrationTestSuite) TestContractsByCreatorMigration_LengthPrefixed() {
	oldKey := v13.GetContractsByCreatorPrefixLegacy(s.testAddr1)
	oldVal := []byte("creator-index-value")
	s.kvStore.Set(oldKey, oldVal)

	s.runMigration()

	// newKey still length-prefixed
	newKey := wasmtypes.GetContractsByCreatorPrefix(s.testAddr1)
	s.Require().Equal(oldVal, s.kvStore.Get(newKey))
}
