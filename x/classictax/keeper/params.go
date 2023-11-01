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

/*func (k Keeper) GetTaxableMsgTypes(ctx sdk.Context) (ret []string) {
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
*/

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {

	// log ctx.KVStore(k.storeKey) and k.storeKey
	k.Logger(ctx).Info("Fetching Params 1", "storekey", k.storeKey)
	k.Logger(ctx).Info("Fetching Params 2", "kvstore", ctx.KVStore(k.storeKey))
	k.Logger(ctx).Info("Fetching Params 3", "msgtypes", ctx.KVStore(k.storeKey).Get([]byte(types.KeyTaxableMsgTypes)))
	// Logging the store key and param set key
	k.paramSpace.GetParamSet(ctx, &params)

	// Logging the fetched params
	ctx.Logger().Info("Fetched Params", "params", params)
	return params
}

// SetParams sets the total set of market parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	ctx.Logger().Info("Setting Params", "params", params)
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k Keeper) GetTaxableMsgTypes(ctx sdk.Context) (ret []string) {
	storeKey := types.KeyTaxableMsgTypes

	// Logging the store key
	ctx.Logger().Info("Fetching TaxableMsgTypes", "storeKey", storeKey)

	k.paramSpace.Get(ctx, storeKey, &ret)

	// Logging the fetched message types
	ctx.Logger().Info("Fetched TaxableMsgTypes", "msgTypes", ret)

	return ret
}
