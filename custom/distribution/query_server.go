package distribution

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// LegacyQueryServer wraps the distribution QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	distrtypes.QueryServer
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
	upgradeHeight  int64
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer distrtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *keeper.Keeper,
	upgradeHeight int64,
) distrtypes.QueryServer {
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
		var params distrtypes.Params
		q.legacySubspace.GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, params)

		// Return updated context
		return sdk.WrapSDKContext(sdkCtx)
	}

	return ctx
}

// Implement the gRPC query service methods by forwarding to the original server
// after ensuring legacy parameters are set

func (q *LegacyQueryServer) Params(ctx context.Context, req *distrtypes.QueryParamsRequest) (*distrtypes.QueryParamsResponse, error) {
	return q.QueryServer.Params(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorOutstandingRewards(ctx context.Context, req *distrtypes.QueryValidatorOutstandingRewardsRequest) (*distrtypes.QueryValidatorOutstandingRewardsResponse, error) {
	return q.QueryServer.ValidatorOutstandingRewards(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorCommission(ctx context.Context, req *distrtypes.QueryValidatorCommissionRequest) (*distrtypes.QueryValidatorCommissionResponse, error) {
	return q.QueryServer.ValidatorCommission(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorSlashes(ctx context.Context, req *distrtypes.QueryValidatorSlashesRequest) (*distrtypes.QueryValidatorSlashesResponse, error) {
	return q.QueryServer.ValidatorSlashes(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegationRewards(ctx context.Context, req *distrtypes.QueryDelegationRewardsRequest) (*distrtypes.QueryDelegationRewardsResponse, error) {
	return q.QueryServer.DelegationRewards(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegationTotalRewards(ctx context.Context, req *distrtypes.QueryDelegationTotalRewardsRequest) (*distrtypes.QueryDelegationTotalRewardsResponse, error) {
	return q.QueryServer.DelegationTotalRewards(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorValidators(ctx context.Context, req *distrtypes.QueryDelegatorValidatorsRequest) (*distrtypes.QueryDelegatorValidatorsResponse, error) {
	return q.QueryServer.DelegatorValidators(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorWithdrawAddress(ctx context.Context, req *distrtypes.QueryDelegatorWithdrawAddressRequest) (*distrtypes.QueryDelegatorWithdrawAddressResponse, error) {
	return q.QueryServer.DelegatorWithdrawAddress(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) CommunityPool(ctx context.Context, req *distrtypes.QueryCommunityPoolRequest) (*distrtypes.QueryCommunityPoolResponse, error) {
	return q.QueryServer.CommunityPool(q.ensureLegacyParams(ctx), req)
}
