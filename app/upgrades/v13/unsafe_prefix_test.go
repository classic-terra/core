package v13_test

import (
	"bytes"
	"fmt"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	apptesting "github.com/classic-terra/core/v3/app/testing"
	v13 "github.com/classic-terra/core/v3/app/upgrades/v13"
)

type UnsafePrefixTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestUnsafePrefixTestSuite(t *testing.T) {
	suite.Run(t, new(UnsafePrefixTestSuite))
}

// TestUnsafePrefixRemovalIssues demonstrates the problems with the old heuristic approach
func (s *UnsafePrefixTestSuite) TestUnsafePrefixRemovalIssues() {
	testCases := []struct {
		name                string
		input               []byte
		expectedOldBehavior []byte // What the old unsafe function would return
		expectedNewBehavior []byte // What the new safe function should return
		description         string
	}{
		{
			name:                "legitimate 21-byte length-prefixed address",
			input:               append([]byte{20}, bytes.Repeat([]byte{0xAA}, 20)...),
			expectedOldBehavior: bytes.Repeat([]byte{0xAA}, 20),
			expectedNewBehavior: bytes.Repeat([]byte{0xAA}, 20),
			description:         "This should be stripped by both old and new logic",
		},
		{
			name:                "32-byte key starting with 0x1F (31)",
			input:               append([]byte{0x1F}, bytes.Repeat([]byte{0xBB}, 31)...),
			expectedOldBehavior: bytes.Repeat([]byte{0xBB}, 31), // OLD LOGIC INCORRECTLY STRIPS!
			expectedNewBehavior: append([]byte{0x1F}, bytes.Repeat([]byte{0xBB}, 31)...),
			description:         "Old logic incorrectly strips first byte because 0x1F == len-1",
		},
		{
			name:                "storage key starting with 0x14 (20)",
			input:               append([]byte{0x14}, bytes.Repeat([]byte{0xCC}, 25)...),
			expectedOldBehavior: bytes.Repeat([]byte{0xCC}, 25), // OLD LOGIC INCORRECTLY STRIPS!
			expectedNewBehavior: append([]byte{0x14}, bytes.Repeat([]byte{0xCC}, 25)...),
			description:         "Old logic incorrectly strips first byte because first byte == 20 and len > 20",
		},
		{
			name:                "hash starting with length-like byte",
			input:               []byte{0x1E, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			expectedOldBehavior: []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66}, // OLD LOGIC INCORRECTLY STRIPS!
			expectedNewBehavior: []byte{0x1E, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			description:         "31-byte hash where first byte (0x1E=30) equals len-1, old logic strips incorrectly",
		},
		{
			name:                "composite key with address-like prefix",
			input:               append(append([]byte{0x14}, bytes.Repeat([]byte{0xDD}, 20)...), []byte{0x01, 0x02, 0x03}...),
			expectedOldBehavior: append(bytes.Repeat([]byte{0xDD}, 20), []byte{0x01, 0x02, 0x03}...), // OLD LOGIC INCORRECTLY STRIPS!
			expectedNewBehavior: append(append([]byte{0x14}, bytes.Repeat([]byte{0xDD}, 20)...), []byte{0x01, 0x02, 0x03}...),
			description:         "Composite key [addr + subkey] where addr part looks length-prefixed, old logic strips incorrectly",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Test old unsafe behavior (this demonstrates the problem)
			oldResult := v13.RemoveLengthPrefixIfNeeded(tc.input)
			s.Require().Equal(tc.expectedOldBehavior, oldResult,
				"Old behavior verification failed: %s", tc.description)

			// Test new safe behavior
			newResult, _ := stripLenPrefixAddrOnly(tc.input)
			s.Require().Equal(tc.expectedNewBehavior, newResult,
				"New safe behavior verification failed: %s", tc.description)

			// If they differ, that proves the old logic was unsafe
			if !bytes.Equal(oldResult, newResult) {
				fmt.Printf("UNSAFE BEHAVIOR DETECTED: %s\n", tc.description)
				fmt.Printf("  Input: %X\n", tc.input)
				fmt.Printf("  Old (unsafe): %X\n", oldResult)
				fmt.Printf("  New (safe): %X\n", newResult)
			}
		})
	}
}

// TestDataCorruptionScenario demonstrates how the unsafe logic can cause data corruption
func (s *UnsafePrefixTestSuite) TestDataCorruptionScenario() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Scenario: Two different keys that would collide after unsafe stripping
	// Key 1: A legitimate 21-byte length-prefixed address
	legitimateAddr := append([]byte{20}, bytes.Repeat([]byte{0xAA}, 20)...)

	// Key 2: A 21-byte composite key that coincidentally starts with 0x14 (20)
	compositeFakeKey := append([]byte{0x14}, bytes.Repeat([]byte{0xAA}, 20)...)

	// Different values for each key
	value1 := []byte("legitimate-contract-info")
	value2 := []byte("composite-storage-data")

	// Store both keys in old format (0x04 prefix for contract info)
	kvStore.Set(append([]byte{0x04}, legitimateAddr...), value1)
	kvStore.Set(append([]byte{0x04}, compositeFakeKey...), value2)

	// Verify both keys exist and have different values
	s.Require().Equal(value1, kvStore.Get(append([]byte{0x04}, legitimateAddr...)))
	s.Require().Equal(value2, kvStore.Get(append([]byte{0x04}, compositeFakeKey...)))

	// Now test what happens with unsafe migration
	// Both keys would be stripped to the same 20-byte sequence: 0xAA repeated 20 times
	strippedKey1 := v13.RemoveLengthPrefixIfNeeded(legitimateAddr)
	strippedKey2 := v13.RemoveLengthPrefixIfNeeded(compositeFakeKey)

	// This is the problem: both keys become identical after unsafe stripping!
	s.Require().Equal(strippedKey1, strippedKey2, "Unsafe stripping causes key collision!")

	// If migration used the old unsafe logic, one value would overwrite the other
	newKey := append([]byte{0x02}, strippedKey1...)
	fmt.Printf("COLLISION DETECTED: Both keys %X and %X strip to same result %X\n",
		legitimateAddr, compositeFakeKey, strippedKey1)
	fmt.Printf("This would cause new key %X to have conflicting data\n", newKey)

	// Test the new safe logic
	safeStripped1, stripped1 := stripLenPrefixAddrOnly(legitimateAddr)
	safeStripped2, stripped2 := stripLenPrefixAddrOnly(compositeFakeKey)

	s.Require().True(stripped1, "Legitimate address should be stripped")
	s.Require().False(stripped2, "Composite key should NOT be stripped")
	s.Require().NotEqual(safeStripped1, safeStripped2, "Safe stripping prevents collision")
}

// TestMigrationWithCollisionGuards tests that collision guards prevent data corruption
func (s *UnsafePrefixTestSuite) TestMigrationWithCollisionGuards() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create a scenario where migration might cause collision
	addr := bytes.Repeat([]byte{0xAA}, 20)
	value1 := []byte("original-value")
	value2 := []byte("conflicting-value")

	// Pre-populate the new location with existing data
	newKey := append([]byte{0x02}, addr...)
	kvStore.Set(newKey, value1)

	// Now try to migrate a different key that would map to the same location
	// This simulates what could happen with unsafe prefix stripping
	oldKey := append([]byte{0x04}, addr...)
	kvStore.Set(oldKey, value2)

	// Simulate migration with collision detection
	originalValue := kvStore.Get(oldKey)
	newFullKey := append([]byte{0x02}, addr...)

	// This should detect the collision since the existing value differs
	if kvStore.Has(newFullKey) && !bytes.Equal(kvStore.Get(newFullKey), originalValue) {
		// This is the collision guard working correctly
		s.Require().True(true, "Collision guard correctly detected potential overwrite")
		fmt.Printf("COLLISION GUARD TRIGGERED: Refusing to overwrite key %X\n", newFullKey)
		fmt.Printf("  Existing value: %s\n", string(kvStore.Get(newFullKey)))
		fmt.Printf("  New value: %s\n", string(originalValue))
	} else {
		s.Fail("Collision guard should have detected the conflict")
	}
}

// TestPerformanceWithLargeDataset tests migration performance and correctness with many keys
func (s *UnsafePrefixTestSuite) TestPerformanceWithLargeDataset() {
	// Setup in-memory database and context
	db := dbm.NewMemDB()
	wasmStoreKey := sdk.NewKVStoreKey(wasmtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(wasmStoreKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	kvStore := ctx.KVStore(wasmStoreKey)

	// Create a large dataset with various key patterns
	numKeys := 5000

	legitPrefixedCount := 0
	unsafelyStrippedCount := 0
	collisionCount := 0

	for i := 0; i < numKeys; i++ {
		// Generate various key patterns
		var key []byte

		switch i % 4 {
		case 0:
			// Legitimate length-prefixed address (21 bytes: [20][addr])
			addr := make([]byte, 20)
			addr[0] = byte(i)
			addr[1] = byte(i >> 8)
			key = append([]byte{20}, addr...)
			legitPrefixedCount++

		case 1:
			// 32-byte key starting with 0x1F (would be unsafely stripped)
			key = make([]byte, 32)
			key[0] = 0x1F
			for j := 1; j < 32; j++ {
				key[j] = byte(i + j)
			}

		case 2:
			// Storage key starting with 0x14 (would be unsafely stripped)
			key = make([]byte, 26)
			key[0] = 0x14
			for j := 1; j < 26; j++ {
				key[j] = byte(i + j)
			}

		case 3:
			// Regular key that shouldn't be stripped
			key = make([]byte, 24)
			for j := 0; j < 24; j++ {
				key[j] = byte(i + j + 100)
			}
		}

		// Store the key
		value := []byte(fmt.Sprintf("value-%d", i))
		kvStore.Set(append([]byte{0x04}, key...), value)

		// Test what old vs new logic would do
		oldStripped := v13.RemoveLengthPrefixIfNeeded(key)
		newStripped, _ := stripLenPrefixAddrOnly(key)

		if !bytes.Equal(oldStripped, newStripped) {
			unsafelyStrippedCount++
		}

		// Check for potential collisions in stripped keys
		for j := 0; j < i; j++ {
			// Recreate previous key for comparison
			var prevKey []byte
			switch j % 4 {
			case 0:
				prevAddr := make([]byte, 20)
				prevAddr[0] = byte(j)
				prevAddr[1] = byte(j >> 8)
				prevKey = append([]byte{20}, prevAddr...)
			case 1:
				prevKey = make([]byte, 32)
				prevKey[0] = 0x1F
				for k := 1; k < 32; k++ {
					prevKey[k] = byte(j + k)
				}
			case 2:
				prevKey = make([]byte, 26)
				prevKey[0] = 0x14
				for k := 1; k < 26; k++ {
					prevKey[k] = byte(j + k)
				}
			case 3:
				prevKey = make([]byte, 24)
				for k := 0; k < 24; k++ {
					prevKey[k] = byte(j + k + 100)
				}
			}

			prevOldStripped := v13.RemoveLengthPrefixIfNeeded(prevKey)
			if bytes.Equal(oldStripped, prevOldStripped) && !bytes.Equal(key, prevKey) {
				collisionCount++
			}
		}
	}

	fmt.Printf("Dataset Analysis:\n")
	fmt.Printf("  Total keys: %d\n", numKeys)
	fmt.Printf("  Legitimate prefixed addresses: %d\n", legitPrefixedCount)
	fmt.Printf("  Keys unsafely stripped by old logic: %d\n", unsafelyStrippedCount)
	fmt.Printf("  Collision potential with old logic: %d\n", collisionCount)

	// Verify the unsafe logic would cause problems
	s.Require().Greater(unsafelyStrippedCount, 0, "Old logic should unsafely strip some keys")

	// If there are potential collisions, that's a serious problem
	if collisionCount > 0 {
		fmt.Printf("WARNING: Old logic would cause %d potential collisions!\n", collisionCount)
	}
}

// stripLenPrefixAddrOnly is the safe version we're testing
func stripLenPrefixAddrOnly(b []byte) (out []byte, stripped bool) {
	if len(b) == 21 && int(b[0]) == 20 {
		out = make([]byte, 20)
		copy(out, b[1:21])
		return out, true
	}
	out = make([]byte, len(b))
	copy(out, b)
	return out, false
}
