package wasm

import (
	"context"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	legacyupgrade "github.com/classic-terra/core/v3/custom/upgrade/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	LegacyParamStoreKeyUploadAccess      = []byte("uploadAccess")
	LegacyParamStoreKeyInstantiateAccess = []byte("instantiateAccess")
)

// LegacyWasmParams is a wrapper around wasmtypes.Params that implements ParamSet
type LegacyWasmParams struct {
	wasmtypes.Params
}

// LegacyQueryServer wraps the wasm QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	wasmtypes.QueryServer
	keeper *wasmkeeper.Keeper
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer wasmtypes.QueryServer,
	keeper *wasmkeeper.Keeper,
) wasmtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer: originalServer,
		keeper:      keeper,
	}
}

func (q *LegacyQueryServer) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if legacyMode == legacyupgrade.LegacyHandlingNone {
		return q.QueryServer.SmartContractState(ctx, req)
	}

	var result []byte
	var queryErr error

	hasTimeIssue := sdkCtx.BlockTime().IsZero() || sdkCtx.BlockTime().Unix() <= 0
	modifiedCtx := sdk.UnwrapSDKContext(ctx)

	// If we fixed the block time, apply it to the new context, it is not the correct historic time
	if hasTimeIssue {
		baseTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		defaultTime := baseTime.Add(time.Duration(sdkCtx.BlockHeight()) * time.Minute)
		modifiedCtx = modifiedCtx.WithBlockTime(defaultTime)
	}

	// Use direct query with keeper for all pre-upgrade heights
	result, queryErr = q.keeper.QuerySmart(modifiedCtx, sdk.MustAccAddressFromBech32(req.Address), req.QueryData)
	// If the direct query was successful, return the result
	if queryErr == nil {
		return &wasmtypes.QuerySmartContractStateResponse{Data: result}, nil
	}

	return nil, queryErr
}
