package keeper

import (
	"github.com/classic-terra/core/v2/x/classictax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetBurnTaxRate(ctx sdk.Context) (ret sdk.Dec) {
	k.paramSpace.Get(ctx, types.KeyBurnTaxRate, &ret)
	return ret
}

func (k Keeper) GetGasPrices(ctx sdk.Context) sdk.DecCoins {
	var fetchParam []sdk.DecCoin
	k.paramSpace.Get(ctx, types.KeyGasPrices, &fetchParam)
	return sdk.NewDecCoins(fetchParam...)
}

func (k Keeper) GetTaxableMsgTypes(ctx sdk.Context) (ret []string) {
	k.paramSpace.Get(ctx, types.KeyTaxableMsgTypes, &ret)
	return ret
}

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
