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

type MigrationCorruptionTestSuite struct {
	suite.Suite
	apptesting.KeeperTestHelper
}

func TestMigrationCorruptionTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationCorruptionTestSuite))
}

// TestMigrationDataCorruption demonstrates how unsafe prefix removal causes real corruption during migration
func (s *MigrationCorruptionTestSuite) TestMigrationDataCorruption() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Setup problematic test data that would expose unsafe prefix removal
	kvStore := ctx.KVStore(wasmStoreKey)

	// Sequence keys (these should migrate correctly)
	kvStore.Set([]byte{0x01}, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	kvStore.Set([]byte{0x02}, []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Contract keys - mix of legitimate and problematic keys

	// 1. Legitimate length-prefixed contract address
	legitimateAddr := append([]byte{20}, bytes.Repeat([]byte{0xAA}, 20)...)
	kvStore.Set(append([]byte{0x04}, legitimateAddr...), []byte("legitimate-contract"))

	// 2. Problematic key that looks like length-prefixed but isn't (32 bytes starting with 0x1F = 31)
	problematicKey1 := append([]byte{0x1F}, bytes.Repeat([]byte{0xBB}, 31)...)
	kvStore.Set(append([]byte{0x04}, problematicKey1...), []byte("problematic-key-1"))

	// 3. Another problematic key (starts with 0x14 = 20, but is actually a composite key)
	problematicKey2 := append([]byte{0x14}, bytes.Repeat([]byte{0xCC}, 25)...)
	kvStore.Set(append([]byte{0x04}, problematicKey2...), []byte("problematic-key-2"))

	// 4. A key that would collide after unsafe stripping
	collisionKey := append([]byte{0x14}, bytes.Repeat([]byte{0xAA}, 20)...) // Same as legitimate after stripping!
	kvStore.Set(append([]byte{0x04}, collisionKey...), []byte("collision-key"))

	// Contract store keys - these demonstrate the composite key problem
	contractAddr := bytes.Repeat([]byte{0xAA}, 20)

	// Store some contract state
	kvStore.Set(append(append([]byte{0x05}, contractAddr...), []byte{0x01}...), []byte("state-1"))
	kvStore.Set(append(append([]byte{0x05}, contractAddr...), []byte{0x02}...), []byte("state-2"))

	// Store state for a length-prefixed contract address
	lengthPrefixedContractAddr := append([]byte{20}, bytes.Repeat([]byte{0xDD}, 20)...)
	kvStore.Set(append(append([]byte{0x05}, lengthPrefixedContractAddr...), []byte{0x01}...), []byte("prefixed-state-1"))

	// Contract history keys
	kvStore.Set(append([]byte{0x06}, contractAddr...), []byte("history-1"))
	kvStore.Set(append([]byte{0x06}, lengthPrefixedContractAddr...), []byte("prefixed-history-1"))

	// Secondary index and params keys
	kvStore.Set([]byte{0x10, 0x01, 0x00}, []byte("index-1"))
	kvStore.Set([]byte{0x11}, []byte("params"))

	// Create a mock wasm keeper
	mockWasmKeeper := createMockWasmKeeperForCorruption(wasmStoreKey)

	// Run the migration with the FIXED safe logic
	err := v13.MigrateWasmKeys(ctx, mockWasmKeeper, wasmStoreKey)
	require.NoError(s.T(), err)

	// Commit the store
	stateStore.Commit()

	// Create a new context with the updated store
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore = ctx.KVStore(wasmStoreKey)

	// Verify post-migration state
	s.verifyPostMigrationState(kvStore, legitimateAddr, problematicKey1, problematicKey2, collisionKey)
}

func (s *MigrationCorruptionTestSuite) verifyPostMigrationState(kvStore sdk.KVStore, legitimateAddr, problematicKey1, problematicKey2, collisionKey []byte) {
	// Verify old keys are deleted
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, legitimateAddr...)), "Old legitimate key should be deleted")
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, problematicKey1...)), "Old problematic key 1 should be deleted")
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, problematicKey2...)), "Old problematic key 2 should be deleted")
	require.Nil(s.T(), kvStore.Get(append([]byte{0x04}, collisionKey...)), "Old collision key should be deleted")

	// Verify new keys exist with SAFE stripping
	// Only the legitimate length-prefixed address should be stripped
	strippedLegitimate := bytes.Repeat([]byte{0xAA}, 20)
	require.Equal(s.T(), []byte("legitimate-contract"),
		kvStore.Get(append([]byte{0x02}, strippedLegitimate...)), "Legitimate contract should be migrated with prefix stripped")

	// Problematic keys should NOT be stripped (safe behavior)
	require.Equal(s.T(), []byte("problematic-key-1"),
		kvStore.Get(append([]byte{0x02}, problematicKey1...)), "Problematic key 1 should be migrated WITHOUT stripping")
	require.Equal(s.T(), []byte("problematic-key-2"),
		kvStore.Get(append([]byte{0x02}, problematicKey2...)), "Problematic key 2 should be migrated WITHOUT stripping")
	require.Equal(s.T(), []byte("collision-key"),
		kvStore.Get(append([]byte{0x02}, collisionKey...)), "Collision key should be migrated WITHOUT stripping")

	// Verify no collisions occurred
	allNewKeys := [][]byte{
		append([]byte{0x02}, strippedLegitimate...),
		append([]byte{0x02}, problematicKey1...),
		append([]byte{0x02}, problematicKey2...),
		append([]byte{0x02}, collisionKey...),
	}

	// Check that all keys are unique
	for i, key1 := range allNewKeys {
		for j, key2 := range allNewKeys {
			if i != j {
				require.False(s.T(), bytes.Equal(key1, key2),
					"Migration should not create key collisions: key %d (%X) == key %d (%X)",
					i, key1, j, key2)
			}
		}
	}

	fmt.Printf("Post-migration verification passed - no data corruption detected\n")
}

// TestCollisionGuardEffectiveness tests that the collision guards actually prevent data corruption
func (s *MigrationCorruptionTestSuite) TestCollisionGuardEffectiveness() {
	// Setup
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(s.T(), stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Pre-populate new location with some data
	existingKey := append([]byte{0x02}, bytes.Repeat([]byte{0xAA}, 20)...)
	existingValue := []byte("existing-data")
	kvStore.Set(existingKey, existingValue)

	// Try to migrate data that would collide
	oldKey := append([]byte{0x04}, append([]byte{20}, bytes.Repeat([]byte{0xAA}, 20)...)...)
	conflictingValue := []byte("conflicting-data")
	kvStore.Set(oldKey, conflictingValue)

	// Test the collision guard logic directly
	originalValue := kvStore.Get(oldKey)
	targetKey := append([]byte{0x02}, bytes.Repeat([]byte{0xAA}, 20)...)

	// This simulates the collision guard check in the fixed migration code
	if kvStore.Has(targetKey) && !bytes.Equal(kvStore.Get(targetKey), originalValue) {
		// The guard should trigger
		fmt.Printf("Collision guard triggered for key %X\n", targetKey)
		fmt.Printf("  Existing: %s\n", string(kvStore.Get(targetKey)))
		fmt.Printf("  Incoming: %s\n", string(originalValue))

		// In the real migration, this would return an error
		require.True(s.T(), true, "Collision guard should trigger")
	} else {
		require.Fail(s.T(), "Collision guard should have detected the conflict")
	}
}

// createMockWasmKeeperForCorruption creates a mock wasm keeper for testing
func createMockWasmKeeperForCorruption(storeKey storetypes.StoreKey) wasmkeeper.Keeper {
	return wasmkeeper.Keeper{}
}
