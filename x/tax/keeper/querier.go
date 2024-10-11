package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/tax/types"
)

var _ types.QueryServer = Keeper{}

// Params queries params of tax module
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// BurnTaxRate queries burn tax rate of tax module
func (k Keeper) BurnTaxRate(c context.Context, _ *types.QueryBurnTaxRateRequest) (*types.QueryBurnTaxRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryBurnTaxRateResponse{BurnTaxRate: k.GetBurnTaxRate(ctx)}, nil
}
