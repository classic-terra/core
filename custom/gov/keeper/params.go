package keeper

import (
	v2lunc1types "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// SetParams sets the gov module's parameters.
func (k Keeper) SetParams(ctx sdk.Context, params v2lunc1types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(govtypes.ParamsKey, bz)

	return nil
}

// GetParams gets the gov module's parameters.
func (k Keeper) GetParams(clientCtx sdk.Context) (params v2lunc1types.Params) {
	store := clientCtx.KVStore(k.storeKey)
	bz := store.Get(govtypes.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}
