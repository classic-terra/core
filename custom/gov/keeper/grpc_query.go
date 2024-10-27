package keeper

import (
	"context"

	v2lunc1 "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	core "github.com/classic-terra/core/v3/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var _ v2lunc1.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) v2lunc1.QueryServer {
	return queryServer{
		k:              k,
		govQueryServer: govkeeper.NewLegacyQueryServer(k.Keeper)}
}

type queryServer struct {
	k              Keeper
	govQueryServer v1beta1.QueryServer
}

// Params returns params of the mint module.
func (q queryServer) Params(ctx context.Context, _ *v2lunc1.QueryParamsRequest) (*v2lunc1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := q.k.GetParams(sdkCtx)

	return &v2lunc1.QueryParamsResponse{Params: params}, nil
}

// ProposalMinimalLUNCByUusd returns min Usd amount proposal needs to deposit
func (q queryServer) ProposalMinimalLUNCByUusd(ctx context.Context, req *v2lunc1.QueryProposalRequest) (*v2lunc1.QueryMinimalDepositProposalResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	depositAmount := q.k.GetDepositLimitBaseUusd(sdkCtx, req.ProposalId)
	coin := sdk.NewCoin(core.MicroLunaDenom, depositAmount)

	// if no min deposit amount by uusd exists, return default min deposit amount
	if depositAmount.IsZero() {
		coin = q.k.GetParams(sdkCtx).MinDeposit[0]
	}

	return &v2lunc1.QueryMinimalDepositProposalResponse{MinimalDeposit: coin}, nil
}
