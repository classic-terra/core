package wasm

import (
	"context"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	legacyupgrade "github.com/classic-terra/core/v3/custom/upgrade/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	LegacyParamStoreKeyUploadAccess      = []byte("uploadAccess")
	LegacyParamStoreKeyInstantiateAccess = []byte("instantiateAccess")
)

// LegacyWasmParams is a wrapper around wasmtypes.Params that implements ParamSet
type LegacyWasmParams struct {
	wasmtypes.Params
}

// ParamKeyTable returns the parameter key table for wasm module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&LegacyWasmParams{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
func (p *LegacyWasmParams) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(LegacyParamStoreKeyUploadAccess, &p.Params.CodeUploadAccess, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair(LegacyParamStoreKeyInstantiateAccess, &p.Params.InstantiateDefaultPermission, func(i interface{}) error { return nil }),
	}
}

// LegacyQueryServer wraps the wasm QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	wasmtypes.QueryServer
	keeper         *wasmkeeper.Keeper
	legacySubspace paramtypes.Subspace
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer wasmtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *wasmkeeper.Keeper,
) wasmtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer:    originalServer,
		keeper:         keeper,
		legacySubspace: legacySubspace,
	}
}

// ensureLegacyParams ensures that legacy parameters are set for pre-upgrade height queries
func (q *LegacyQueryServer) ensureLegacyParams(ctx context.Context) context.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Only set legacy params for pre-upgrade heights
	upgradeHeight := legacyupgrade.GetUpgradeHeight(sdkCtx.ChainID())
	if sdkCtx.BlockHeight() > 0 && sdkCtx.BlockHeight() < upgradeHeight && q.keeper != nil {
		// Try to get params from legacy subspace
		var legacyParams LegacyWasmParams
		q.legacySubspace.GetParamSet(sdkCtx, &legacyParams)

		// Set the params in the keeper
		sdkCtx.Logger().Debug("Using params for historic block",
			"upload_access", legacyParams.Params.CodeUploadAccess.Permission,
			"instantiate_permission", legacyParams.Params.InstantiateDefaultPermission)
		q.keeper.SetParams(sdkCtx, legacyParams.Params)

		// Return updated context
		return sdk.WrapSDKContext(sdkCtx)
	}

	return ctx
}

// Implement the gRPC query service methods by forwarding to the original server
// after ensuring legacy parameters are set

func (q *LegacyQueryServer) ContractInfo(ctx context.Context, req *wasmtypes.QueryContractInfoRequest) (*wasmtypes.QueryContractInfoResponse, error) {
	return q.QueryServer.ContractInfo(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ContractHistory(ctx context.Context, req *wasmtypes.QueryContractHistoryRequest) (*wasmtypes.QueryContractHistoryResponse, error) {
	return q.QueryServer.ContractHistory(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ContractsByCode(ctx context.Context, req *wasmtypes.QueryContractsByCodeRequest) (*wasmtypes.QueryContractsByCodeResponse, error) {
	return q.QueryServer.ContractsByCode(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) AllContractState(ctx context.Context, req *wasmtypes.QueryAllContractStateRequest) (*wasmtypes.QueryAllContractStateResponse, error) {
	return q.QueryServer.AllContractState(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) RawContractState(ctx context.Context, req *wasmtypes.QueryRawContractStateRequest) (*wasmtypes.QueryRawContractStateResponse, error) {
	return q.QueryServer.RawContractState(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	upgradeHeight := legacyupgrade.GetUpgradeHeight(sdkCtx.ChainID())
	if sdkCtx.BlockHeight() == 0 || sdkCtx.BlockHeight() >= upgradeHeight {
		return q.QueryServer.SmartContractState(ctx, req)
	}

	var result []byte
	var queryErr error

	hasTimeIssue := sdkCtx.BlockTime().IsZero() || sdkCtx.BlockTime().Unix() <= 0

	// Set legacy parameters
	ctx = q.ensureLegacyParams(ctx)
	// Update the modified context with the legacy params
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

func (q *LegacyQueryServer) Code(ctx context.Context, req *wasmtypes.QueryCodeRequest) (*wasmtypes.QueryCodeResponse, error) {
	return q.QueryServer.Code(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Codes(ctx context.Context, req *wasmtypes.QueryCodesRequest) (*wasmtypes.QueryCodesResponse, error) {
	return q.QueryServer.Codes(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) PinnedCodes(ctx context.Context, req *wasmtypes.QueryPinnedCodesRequest) (*wasmtypes.QueryPinnedCodesResponse, error) {
	return q.QueryServer.PinnedCodes(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Params(ctx context.Context, req *wasmtypes.QueryParamsRequest) (*wasmtypes.QueryParamsResponse, error) {
	return q.QueryServer.Params(q.ensureLegacyParams(ctx), req)
}
