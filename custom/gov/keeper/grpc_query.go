package keeper

import (
	"context"

	v2custom "github.com/classic-terra/core/v3/custom/gov/types/v2custom"
	core "github.com/classic-terra/core/v3/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var _ v2custom.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) v2custom.QueryServer {
	return queryServer{
		k:              k,
		govQueryServer: govkeeper.NewLegacyQueryServer(k.Keeper),
	}
}

type queryServer struct {
	k              Keeper
	govQueryServer v1beta1.QueryServer
}

// Params returns params of the mint module.
func (q queryServer) Params(ctx context.Context, _ *v2custom.QueryParamsRequest) (*v2custom.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := q.k.GetParams(sdkCtx)

	return &v2custom.QueryParamsResponse{Params: params}, nil
}

// ProposalMinimalLUNCByUstc returns min Usd amount proposal needs to deposit
func (q queryServer) ProposalMinimalLUNCByUstc(ctx context.Context, req *v2custom.QueryProposalRequest) (*v2custom.QueryMinimalDepositProposalResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	depositAmount := q.k.GetDepositLimitBaseUstc(sdkCtx, req.ProposalId)
	coin := sdk.NewCoin(core.MicroLunaDenom, depositAmount)

	// if no min deposit amount by usd exists, return default min deposit amount
	if depositAmount.IsZero() {
		coin = q.k.GetParams(sdkCtx).MinDeposit[0]
	}

	return &v2custom.QueryMinimalDepositProposalResponse{MinimalDeposit: coin}, nil
}
