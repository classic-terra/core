package keeper

import (
	"github.com/classic-terra/core/v2/x/classictax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetBurnTaxRate(ctx sdk.Context) (ret sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyBurnTaxRate, &ret)
	return ret
}

func (k Keeper) GetGasFee(ctx sdk.Context) (ret sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyGasFee, &ret)
	return ret
}

func (k Keeper) GetTaxableMsgTypes(ctx sdk.Context) (ret []string) {
	k.paramSpace.Get(ctx, types.KeyTaxableMsgTypes, &ret)
	return ret
}

// GetParams returns the total set of market parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of market parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
