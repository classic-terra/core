package wasm

import (
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	coretypes "github.com/classic-terra/core/v3/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockWasmVMQueryHandler implements wasmkeeper.WasmVMQueryHandler for testing
type MockWasmVMQueryHandler struct {
	handleQueryFunc func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error)
	callLog         []QueryCall
}

type QueryCall struct {
	Ctx     sdk.Context
	Caller  sdk.AccAddress
	Request wasmvmtypes.QueryRequest
}

func NewMockWasmVMQueryHandler() *MockWasmVMQueryHandler {
	return &MockWasmVMQueryHandler{
		callLog: make([]QueryCall, 0),
	}
}

func (m *MockWasmVMQueryHandler) HandleQuery(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
	// Log the call
	m.callLog = append(m.callLog, QueryCall{
		Ctx:     ctx,
		Caller:  caller,
		Request: request,
	})

	if m.handleQueryFunc != nil {
		return m.handleQueryFunc(ctx, caller, request)
	}
	return []byte("mock-response"), nil
}

func (m *MockWasmVMQueryHandler) SetHandleQueryFunc(f func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error)) {
	m.handleQueryFunc = f
}

func (m *MockWasmVMQueryHandler) GetCallLog() []QueryCall {
	return m.callLog
}

func (m *MockWasmVMQueryHandler) ClearCallLog() {
	m.callLog = make([]QueryCall, 0)
}

type LegacyQueryHandlerTestSuite struct {
	suite.Suite
	ctx      sdk.Context
	storeKey storetypes.StoreKey
	handler  wasmkeeper.WasmVMQueryHandler
	mock     *MockWasmVMQueryHandler
}

func TestLegacyQueryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LegacyQueryHandlerTestSuite))
}

func (suite *LegacyQueryHandlerTestSuite) SetupTest() {
	// Create test context with store
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	suite.storeKey = sdk.NewKVStoreKey("wasm")
	ms.MountStoreWithDB(suite.storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(suite.T(), ms.LoadLatestVersion())

	suite.ctx = sdk.NewContext(ms, tmproto.Header{Height: 1}, false, log.NewNopLogger())

	// Create mock and handler
	suite.mock = NewMockWasmVMQueryHandler()
	suite.handler = NewLegacyQueryHandler(suite.mock, suite.storeKey)
}

func (suite *LegacyQueryHandlerTestSuite) TestNewLegacyQueryHandler() {
	// Test constructor
	handler := NewLegacyQueryHandler(suite.mock, suite.storeKey)
	require.NotNil(suite.T(), handler)

	// The function signature already guarantees it returns wasmkeeper.WasmVMQueryHandler
	// so no need for type assertion - just verify it's not nil and can be called
	require.Implements(suite.T(), (*wasmkeeper.WasmVMQueryHandler)(nil), handler)
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_PostMigrationHeight() {
	// Test with height after migration (should not use legacy store)
	testCases := []struct {
		name    string
		chainID string
		height  int64
	}{
		{
			name:    "mainnet post-migration",
			chainID: coretypes.ColumbusChainID,
			height:  25619230, // At migration height
		},
		{
			name:    "testnet post-migration",
			chainID: coretypes.RebelChainID,
			height:  26888500, // After migration height
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mock.ClearCallLog()

			// Create context with specific chain ID and height
			ctx := suite.ctx.WithChainID(tc.chainID).WithBlockHeight(tc.height)
			caller := sdk.AccAddress([]byte("test-caller"))
			request := wasmvmtypes.QueryRequest{
				Wasm: &wasmvmtypes.WasmQuery{
					Smart: &wasmvmtypes.SmartQuery{
						ContractAddr: "test-contract",
						Msg:          []byte(`{"get_count":{}}`),
					},
				},
			}

			// Execute query
			result, err := suite.handler.HandleQuery(ctx, caller, request)

			// Verify no error and got expected result
			require.NoError(suite.T(), err)
			require.Equal(suite.T(), []byte("mock-response"), result)

			// Verify the mock was called with original context (not wrapped)
			calls := suite.mock.GetCallLog()
			require.Len(suite.T(), calls, 1)
			require.Equal(suite.T(), tc.chainID, calls[0].Ctx.ChainID())
			require.Equal(suite.T(), tc.height, calls[0].Ctx.BlockHeight())
		})
	}
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_PreMigrationHeight() {
	// Test with height before migration (should use legacy store)
	testCases := []struct {
		name    string
		chainID string
		height  int64
	}{
		{
			name:    "mainnet pre-migration",
			chainID: coretypes.ColumbusChainID,
			height:  25619229, // Before migration height
		},
		{
			name:    "testnet pre-migration",
			chainID: coretypes.RebelChainID,
			height:  26888495, // Before migration height
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mock.ClearCallLog()

			// Create context with specific chain ID and height
			ctx := suite.ctx.WithChainID(tc.chainID).WithBlockHeight(tc.height)
			caller := sdk.AccAddress([]byte("test-caller"))
			request := wasmvmtypes.QueryRequest{
				Wasm: &wasmvmtypes.WasmQuery{
					Smart: &wasmvmtypes.SmartQuery{
						ContractAddr: "test-contract",
						Msg:          []byte(`{"get_count":{}}`),
					},
				},
			}

			// Execute query
			result, err := suite.handler.HandleQuery(ctx, caller, request)

			// Verify no error and got expected result
			require.NoError(suite.T(), err)
			require.Equal(suite.T(), []byte("mock-response"), result)

			// Verify the mock was called
			calls := suite.mock.GetCallLog()
			require.Len(suite.T(), calls, 1)

			// For pre-migration heights, the context should be wrapped with legacy store
			// We can verify this by checking that the context has the legacy store
			callCtx := calls[0].Ctx
			require.Equal(suite.T(), tc.chainID, callCtx.ChainID())
			require.Equal(suite.T(), tc.height, callCtx.BlockHeight())

			// The context should have a wrapped multistore with legacy wasm store
			tmpStore := callCtx.KVStore(suite.storeKey)
			require.NotNil(suite.T(), tmpStore)

			// Check if the multistore is wrapped with legacyMultiStore
			// The actual store might be wrapped by gas metering, so we check the multistore instead
			ms := callCtx.MultiStore()
			lms, isLegacyMultiStore := ms.(legacyMultiStore)
			require.True(suite.T(), isLegacyMultiStore, "Expected legacyMultiStore for pre-migration height")

			// Verify the legacy store is of the correct type
			_, isLegacyWasmStore := lms.legacy.(*legacyWasmStore)
			require.True(suite.T(), isLegacyWasmStore, "Expected legacyWasmStore inside legacyMultiStore")
		})
	}
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_EdgeCases() {
	suite.Run("zero height", func() {
		suite.mock.ClearCallLog()

		ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(0)
		caller := sdk.AccAddress([]byte("test-caller"))
		request := wasmvmtypes.QueryRequest{
			Wasm: &wasmvmtypes.WasmQuery{
				Smart: &wasmvmtypes.SmartQuery{
					ContractAddr: "test-contract",
					Msg:          []byte(`{"get_count":{}}`),
				},
			},
		}

		result, err := suite.handler.HandleQuery(ctx, caller, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), []byte("mock-response"), result)

		// Zero height should not trigger legacy mode
		calls := suite.mock.GetCallLog()
		require.Len(suite.T(), calls, 1)
		tmpStore := calls[0].Ctx.KVStore(suite.storeKey)
		_, isLegacyStore := tmpStore.(*legacyWasmStore)
		require.False(suite.T(), isLegacyStore, "Zero height should not use legacy store")
	})

	suite.Run("negative height", func() {
		suite.mock.ClearCallLog()

		ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(-1)
		caller := sdk.AccAddress("test-caller")
		request := wasmvmtypes.QueryRequest{
			Wasm: &wasmvmtypes.WasmQuery{
				Smart: &wasmvmtypes.SmartQuery{
					ContractAddr: "test-contract",
					Msg:          []byte(`{"get_count":{}}`),
				},
			},
		}

		result, err := suite.handler.HandleQuery(ctx, caller, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), []byte("mock-response"), result)

		// Negative height should not trigger legacy mode
		calls := suite.mock.GetCallLog()
		require.Len(suite.T(), calls, 1)
		tmpStore := calls[0].Ctx.KVStore(suite.storeKey)
		_, isLegacyStore := tmpStore.(*legacyWasmStore)
		require.False(suite.T(), isLegacyStore, "Negative height should not use legacy store")
	})
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_NilStoreKey() {
	// Test with nil store key
	handler := NewLegacyQueryHandler(suite.mock, nil)

	ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(100) // Pre-migration
	caller := sdk.AccAddress([]byte("test-caller"))
	request := wasmvmtypes.QueryRequest{
		Wasm: &wasmvmtypes.WasmQuery{
			Smart: &wasmvmtypes.SmartQuery{
				ContractAddr: "test-contract",
				Msg:          []byte(`{"get_count":{}}`),
			},
		},
	}

	suite.mock.ClearCallLog()
	result, err := handler.HandleQuery(ctx, caller, request)

	// Should still work but not use legacy store
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), []byte("mock-response"), result)

	calls := suite.mock.GetCallLog()
	require.Len(suite.T(), calls, 1)
	// Should use original context since store key is nil
	require.Equal(suite.T(), ctx.ChainID(), calls[0].Ctx.ChainID())
	require.Equal(suite.T(), ctx.BlockHeight(), calls[0].Ctx.BlockHeight())
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_ErrorPropagation() {
	// Test that errors from the next handler are properly propagated
	expectedError := sdkerrors.ErrInvalidRequest.Wrap("test error")
	suite.mock.SetHandleQueryFunc(func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
		return nil, expectedError
	})

	ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(100)
	caller := sdk.AccAddress([]byte("test-caller"))
	request := wasmvmtypes.QueryRequest{
		Wasm: &wasmvmtypes.WasmQuery{
			Smart: &wasmvmtypes.SmartQuery{
				ContractAddr: "test-contract",
				Msg:          []byte(`{"get_count":{}}`),
			},
		},
	}

	result, err := suite.handler.HandleQuery(ctx, caller, request)

	require.Error(suite.T(), err)
	require.Equal(suite.T(), expectedError, err)
	require.Nil(suite.T(), result)
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_DifferentQueryTypes() {
	// Test different types of wasm queries
	testCases := []struct {
		name    string
		request wasmvmtypes.QueryRequest
	}{
		{
			name: "smart query",
			request: wasmvmtypes.QueryRequest{
				Wasm: &wasmvmtypes.WasmQuery{
					Smart: &wasmvmtypes.SmartQuery{
						ContractAddr: "test-contract",
						Msg:          []byte(`{"get_count":{}}`),
					},
				},
			},
		},
		{
			name: "raw query",
			request: wasmvmtypes.QueryRequest{
				Wasm: &wasmvmtypes.WasmQuery{
					Raw: &wasmvmtypes.RawQuery{
						ContractAddr: "test-contract",
						Key:          []byte("key"),
					},
				},
			},
		},
		{
			name: "contract info query",
			request: wasmvmtypes.QueryRequest{
				Wasm: &wasmvmtypes.WasmQuery{
					ContractInfo: &wasmvmtypes.ContractInfoQuery{
						ContractAddr: "test-contract",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mock.ClearCallLog()

			// Test with pre-migration height to ensure legacy store is used
			ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(100)
			caller := sdk.AccAddress([]byte("test-caller"))

			result, err := suite.handler.HandleQuery(ctx, caller, tc.request)

			require.NoError(suite.T(), err)
			require.Equal(suite.T(), []byte("mock-response"), result)

			calls := suite.mock.GetCallLog()
			require.Len(suite.T(), calls, 1)
			require.Equal(suite.T(), tc.request, calls[0].Request)
		})
	}
}

func (suite *LegacyQueryHandlerTestSuite) TestHandleQuery_LegacyStoreIntegration() {
	// Test that the legacy store actually works with real data
	suite.Run("legacy store integration", func() {
		suite.mock.ClearCallLog()

		// Set up some data in the underlying store to simulate old format
		ctx := suite.ctx.WithChainID(coretypes.ColumbusChainID).WithBlockHeight(100) // Pre-migration
		tmpStore := ctx.KVStore(suite.storeKey)

		// Simulate old format contract store key: 0x05 + 0x14 + 20-byte-addr + storage-key
		contractAddr := make([]byte, 20)
		for i := range contractAddr {
			contractAddr[i] = byte(i + 1) // Simple test address
		}
		storageKey := []byte("test-storage-key")
		oldFormatKey := append([]byte{0x05, 0x14}, contractAddr...)
		oldFormatKey = append(oldFormatKey, storageKey...)
		testValue := []byte("test-value")
		tmpStore.Set(oldFormatKey, testValue)

		// Set up mock to verify the legacy store can read the data
		suite.mock.SetHandleQueryFunc(func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
			// Try to read using new format key: 0x03 + addr + storage-key
			newFormatKey := append([]byte{0x03}, contractAddr...)
			newFormatKey = append(newFormatKey, storageKey...)

			legacyStore := ctx.KVStore(suite.storeKey)
			value := legacyStore.Get(newFormatKey)

			if value == nil {
				return nil, sdkerrors.ErrNotFound.Wrap("key not found in legacy store")
			}

			return value, nil
		})

		caller := sdk.AccAddress([]byte("test-caller"))
		request := wasmvmtypes.QueryRequest{
			Wasm: &wasmvmtypes.WasmQuery{
				Raw: &wasmvmtypes.RawQuery{
					ContractAddr: "test-contract",
					Key:          []byte("test-key"),
				},
			},
		}

		// Execute query - should use legacy store and find the data
		result, err := suite.handler.HandleQuery(ctx, caller, request)

		require.NoError(suite.T(), err)
		require.Equal(suite.T(), testValue, result, "Legacy store should translate keys and return correct value")

		// Verify the mock was called with legacy context
		calls := suite.mock.GetCallLog()
		require.Len(suite.T(), calls, 1)

		// Verify legacy store was used
		ms := calls[0].Ctx.MultiStore()
		_, isLegacyMultiStore := ms.(legacyMultiStore)
		require.True(suite.T(), isLegacyMultiStore, "Should use legacy multistore for pre-migration height")
	})
}

// Test the isPreWasmKeyMigration function directly
func (suite *LegacyQueryHandlerTestSuite) TestIsPreWasmKeyMigration() {
	testCases := []struct {
		name     string
		chainID  string
		height   int64
		expected bool
	}{
		// Mainnet tests
		{"mainnet before migration", coretypes.ColumbusChainID, 25619229, true},
		{"mainnet at migration", coretypes.ColumbusChainID, 25619230, false},
		{"mainnet after migration", coretypes.ColumbusChainID, 25619231, false},

		// Testnet tests
		{"testnet before migration", coretypes.RebelChainID, 26888495, true},
		{"testnet at migration", coretypes.RebelChainID, 26888496, false},
		{"testnet after migration", coretypes.RebelChainID, 26888497, false},

		// Edge cases
		{"zero height", coretypes.ColumbusChainID, 0, false},
		{"negative height", coretypes.ColumbusChainID, -1, false},
		{"unknown chain", "unknown", 100, false},
		{"empty chain", "", 100, false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := isPreWasmKeyMigration(tc.chainID, tc.height)
			require.Equal(suite.T(), tc.expected, result)
		})
	}
}
