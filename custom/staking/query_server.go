package staking

import (
	"context"

	legacytypes "github.com/classic-terra/core/v3/custom/staking/types"
	legacyupgrade "github.com/classic-terra/core/v3/custom/upgrade/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// LegacyQueryServer wraps the staking QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	stakingtypes.QueryServer
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer stakingtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *keeper.Keeper,
) stakingtypes.QueryServer {
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
	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	sdkCtx.Logger().Debug("Setting legacy params for pre-upgrade height queries",
		"block_height", sdkCtx.BlockHeight(),
		"legacy_mode", legacyMode,
		"chain_id", sdkCtx.ChainID(),
	)

	if legacyMode == legacyupgrade.LegacyHandlingV1 {
		if !q.legacySubspace.HasKeyTable() {
			q.legacySubspace.WithKeyTable(legacytypes.ParamKeyTable())
		}

		var params legacytypes.LegacyParams
		q.legacySubspace.GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, stakingtypes.Params{
			UnbondingTime:     params.UnbondingTime,
			MaxValidators:     params.MaxValidators,
			MaxEntries:        params.MaxEntries,
			HistoricalEntries: params.HistoricalEntries,
			BondDenom:         params.BondDenom,
			MinCommissionRate: sdk.ZeroDec(),
		})

		// Return updated context
		sdkCtx.Logger().Debug("Legacy params set for pre-upgrade height queries",
			"block_height", sdkCtx.BlockHeight(),
			"chain_id", sdkCtx.ChainID(),
			"params", params,
			"legacy_mode", legacyMode,
			"ctx", sdkCtx,
		)
		return sdk.WrapSDKContext(sdkCtx)
	}

	if legacyMode == legacyupgrade.LegacyHandlingV2 {
		if !q.legacySubspace.HasKeyTable() {
			q.legacySubspace.WithKeyTable(stakingtypes.ParamKeyTable())
		}

		var params stakingtypes.Params
		q.legacySubspace.GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, params)

		// Return updated context
		sdkCtx.Logger().Debug("Legacy params set for pre-upgrade height queries",
			"block_height", sdkCtx.BlockHeight(),
			"chain_id", sdkCtx.ChainID(),
			"params", params,
			"legacy_mode", legacyMode,
			"ctx", sdkCtx,
		)
		return sdk.WrapSDKContext(sdkCtx)
	}

	return ctx
}

// Implement the gRPC query service methods by forwarding to the original server
// after ensuring legacy parameters are set

func (q *LegacyQueryServer) Validators(ctx context.Context, req *stakingtypes.QueryValidatorsRequest) (*stakingtypes.QueryValidatorsResponse, error) {
	return q.QueryServer.Validators(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Validator(ctx context.Context, req *stakingtypes.QueryValidatorRequest) (*stakingtypes.QueryValidatorResponse, error) {
	return q.QueryServer.Validator(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorDelegations(ctx context.Context, req *stakingtypes.QueryValidatorDelegationsRequest) (*stakingtypes.QueryValidatorDelegationsResponse, error) {
	return q.QueryServer.ValidatorDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorUnbondingDelegations(ctx context.Context, req *stakingtypes.QueryValidatorUnbondingDelegationsRequest) (*stakingtypes.QueryValidatorUnbondingDelegationsResponse, error) {
	return q.QueryServer.ValidatorUnbondingDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Delegation(ctx context.Context, req *stakingtypes.QueryDelegationRequest) (*stakingtypes.QueryDelegationResponse, error) {
	return q.QueryServer.Delegation(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) UnbondingDelegation(ctx context.Context, req *stakingtypes.QueryUnbondingDelegationRequest) (*stakingtypes.QueryUnbondingDelegationResponse, error) {
	return q.QueryServer.UnbondingDelegation(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorDelegations(ctx context.Context, req *stakingtypes.QueryDelegatorDelegationsRequest) (*stakingtypes.QueryDelegatorDelegationsResponse, error) {
	return q.QueryServer.DelegatorDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorUnbondingDelegations(ctx context.Context, req *stakingtypes.QueryDelegatorUnbondingDelegationsRequest) (*stakingtypes.QueryDelegatorUnbondingDelegationsResponse, error) {
	return q.QueryServer.DelegatorUnbondingDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Redelegations(ctx context.Context, req *stakingtypes.QueryRedelegationsRequest) (*stakingtypes.QueryRedelegationsResponse, error) {
	return q.QueryServer.Redelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorValidators(ctx context.Context, req *stakingtypes.QueryDelegatorValidatorsRequest) (*stakingtypes.QueryDelegatorValidatorsResponse, error) {
	return q.QueryServer.DelegatorValidators(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorValidator(ctx context.Context, req *stakingtypes.QueryDelegatorValidatorRequest) (*stakingtypes.QueryDelegatorValidatorResponse, error) {
	return q.QueryServer.DelegatorValidator(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) HistoricalInfo(ctx context.Context, req *stakingtypes.QueryHistoricalInfoRequest) (*stakingtypes.QueryHistoricalInfoResponse, error) {
	return q.QueryServer.HistoricalInfo(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Pool(ctx context.Context, req *stakingtypes.QueryPoolRequest) (*stakingtypes.QueryPoolResponse, error) {
	return q.QueryServer.Pool(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Params(ctx context.Context, req *stakingtypes.QueryParamsRequest) (*stakingtypes.QueryParamsResponse, error) {
	return q.QueryServer.Params(q.ensureLegacyParams(ctx), req)
}
