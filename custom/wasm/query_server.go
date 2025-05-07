package wasm

import (
	"context"
	
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// LegacyWasmParams is a wrapper around wasmtypes.Params that implements ParamSet
type LegacyWasmParams struct {
	wasmtypes.Params
}

// ParamSetPairs implements the ParamSet interface
func (p *LegacyWasmParams) ParamSetPairs() paramtypes.ParamSetPairs {
	// This should match the key table used in the legacy subspace
	return paramtypes.ParamSetPairs{
		// Use string literals for the parameter keys as they might have changed in newer versions
		paramtypes.NewParamSetPair([]byte("uploadAccess"), &p.CodeUploadAccess, nil),
		paramtypes.NewParamSetPair([]byte("instantiateAccess"), &p.InstantiateDefaultPermission, nil),
		// In SDK 0.46, this was likely a uint64 value
		// We'll read it but not use it directly since the field structure might have changed
		paramtypes.NewParamSetPair([]byte("maxWasmCodeSize"), new(uint64), nil),
	}
}

// LegacyQueryServer wraps the wasm QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	wasmtypes.QueryServer
	keeper         *wasmkeeper.Keeper
	legacySubspace paramtypes.Subspace
	upgradeHeight  int64
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer wasmtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *wasmkeeper.Keeper,
	upgradeHeight int64,
) wasmtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer:    originalServer,
		keeper:         keeper,
		legacySubspace: legacySubspace,
		upgradeHeight:  upgradeHeight,
	}
}

// ensureLegacyParams ensures that legacy parameters are set for pre-upgrade height queries
func (q *LegacyQueryServer) ensureLegacyParams(ctx context.Context) context.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	
	// Only set legacy params for pre-upgrade heights
	if sdkCtx.BlockHeight() > 0 && sdkCtx.BlockHeight() < q.upgradeHeight {
		legacyParams := &LegacyWasmParams{}
		q.legacySubspace.GetParamSet(sdkCtx, legacyParams)
		
		// Set the params in the keeper if possible
		if q.keeper != nil {
			q.keeper.SetParams(sdkCtx, legacyParams.Params)
		}
		
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
	return q.QueryServer.SmartContractState(q.ensureLegacyParams(ctx), req)
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
	ctx = q.ensureLegacyParams(ctx)
	
	// For pre-upgrade heights, we might want to return the legacy params directly
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() > 0 && sdkCtx.BlockHeight() < q.upgradeHeight {
		legacyParams := &LegacyWasmParams{}
		q.legacySubspace.GetParamSet(sdkCtx, legacyParams)
		return &wasmtypes.QueryParamsResponse{Params: legacyParams.Params}, nil
	}
	
	return q.QueryServer.Params(ctx, req)
}
