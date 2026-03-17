package keeper

import (
	"context"

	"github.com/classic-terra/core/v4/x/tax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	return &types.QueryBurnTaxRateResponse{TaxRate: k.GetBurnTaxRate(ctx)}, nil
}
