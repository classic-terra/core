package wasm

import (
	"context"
	"time"

	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	legacyupgrade "github.com/classic-terra/core/v4/custom/upgrade/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LegacyQueryServer wraps the wasm QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	wasmtypes.QueryServer
	keeper   *wasmkeeper.Keeper
	storeKey storetypes.StoreKey
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance

func NewLegacyQueryServer(
	originalServer wasmtypes.QueryServer,
	keeper *wasmkeeper.Keeper,
	storeKey storetypes.StoreKey,
) wasmtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer: originalServer,
		keeper:      keeper,
		storeKey:    storeKey,
	}
}

func (q *LegacyQueryServer) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	// Defensive: match wasmd behavior
	if req == nil {
		return nil, wasmtypes.ErrEmpty
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	preMigration := isPreWasmKeyMigration(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if !preMigration { // no legacy mapping needed, use upstream implementation directly
		return q.QueryServer.SmartContractState(ctx, req)
	}

	// Wrap context with legacy key-translation store
	ctx, sdkCtx, legacyUsed := q.ensureLegacyWasm(ctx, "SmartContractState")

	// Fix zero/invalid block time to avoid wasm vm time-dependent issues on very old heights.
	hasTimeIssue := sdkCtx.BlockTime().IsZero() || sdkCtx.BlockTime().Unix() <= 0
	effectiveCtx := sdk.UnwrapSDKContext(ctx)
	if hasTimeIssue {
		baseTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		pseudoTime := baseTime.Add(time.Duration(sdkCtx.BlockHeight()) * time.Minute)
		effectiveCtx = effectiveCtx.WithBlockTime(pseudoTime)
	}

	// Execute the smart query directly via keeper to ensure we bypass any post-migration assumptions.
	addr := sdk.MustAccAddressFromBech32(req.Address)
	data, err := q.keeper.QuerySmart(effectiveCtx, addr, req.QueryData)
	if err != nil {
		return nil, err
	}
	if legacyUsed {
		sdkCtx.Logger().Info("legacy wasm smart query succeeded", "contract", req.Address, "legacy", legacyUsed, "height", sdkCtx.BlockHeight())
	}
	return &wasmtypes.QuerySmartContractStateResponse{Data: data}, nil
}

// ensureLegacyWasm wraps the context with the legacy wasm store if the height is pre-migration.
func (q *LegacyQueryServer) ensureLegacyWasm(ctx context.Context, method string) (context.Context, sdk.Context, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	preMigration := isPreWasmKeyMigration(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if preMigration {
		wrappedCtx, ok := prepareLegacyWasmContext(sdkCtx, q.storeKey)
		if ok {
			ctx = sdk.WrapSDKContext(wrappedCtx)
			wrappedCtx.Logger().Info("using legacy wasm key mapping", "method", method, "height", wrappedCtx.BlockHeight(), "chain_id", wrappedCtx.ChainID(), "legacy_mode", legacyMode, "pre_migration", preMigration, "store_wrap", ok)
			return ctx, wrappedCtx, true
		}
		// wrapping failed; log once
		sdkCtx.Logger().Info("legacy wasm key mapping requested but not applied", "method", method, "height", sdkCtx.BlockHeight(), "chain_id", sdkCtx.ChainID(), "legacy_mode", legacyMode, "pre_migration", preMigration, "store_wrap", ok)
		return ctx, sdkCtx, false
	}
	return ctx, sdkCtx, false
}

// Below we wrap additional query methods so that all contract-related historic queries benefit
// from the legacy key translation.

func (q *LegacyQueryServer) ContractInfo(ctx context.Context, req *wasmtypes.QueryContractInfoRequest) (*wasmtypes.QueryContractInfoResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "ContractInfo")
	return q.QueryServer.ContractInfo(ctx, req)
}

func (q *LegacyQueryServer) ContractHistory(ctx context.Context, req *wasmtypes.QueryContractHistoryRequest) (*wasmtypes.QueryContractHistoryResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "ContractHistory")
	return q.QueryServer.ContractHistory(ctx, req)
}

func (q *LegacyQueryServer) ContractsByCode(ctx context.Context, req *wasmtypes.QueryContractsByCodeRequest) (*wasmtypes.QueryContractsByCodeResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "ContractsByCode")
	return q.QueryServer.ContractsByCode(ctx, req)
}

func (q *LegacyQueryServer) AllContractState(ctx context.Context, req *wasmtypes.QueryAllContractStateRequest) (*wasmtypes.QueryAllContractStateResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "AllContractState")
	return q.QueryServer.AllContractState(ctx, req)
}

func (q *LegacyQueryServer) RawContractState(ctx context.Context, req *wasmtypes.QueryRawContractStateRequest) (*wasmtypes.QueryRawContractStateResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "RawContractState")
	return q.QueryServer.RawContractState(ctx, req)
}

func (q *LegacyQueryServer) Code(ctx context.Context, req *wasmtypes.QueryCodeRequest) (*wasmtypes.QueryCodeResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "Code")
	return q.QueryServer.Code(ctx, req)
}

func (q *LegacyQueryServer) Codes(ctx context.Context, req *wasmtypes.QueryCodesRequest) (*wasmtypes.QueryCodesResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "Codes")
	return q.QueryServer.Codes(ctx, req)
}

func (q *LegacyQueryServer) PinnedCodes(ctx context.Context, req *wasmtypes.QueryPinnedCodesRequest) (*wasmtypes.QueryPinnedCodesResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "PinnedCodes")
	return q.QueryServer.PinnedCodes(ctx, req)
}

func (q *LegacyQueryServer) Params(ctx context.Context, req *wasmtypes.QueryParamsRequest) (*wasmtypes.QueryParamsResponse, error) {
	ctx, _, _ = q.ensureLegacyWasm(ctx, "Params")
	return q.QueryServer.Params(ctx, req)
}
