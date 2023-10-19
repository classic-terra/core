package keeper

import (
	"encoding/json"

	"github.com/classic-terra/core/v2/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetFreezeAddrs
func (k Keeper) GetFreezeAddrs(ctx sdk.Context) types.FreezeList {

	store := ctx.KVStore(k.storeKey)

	b := store.Get(types.FreezeKey)
	if b == nil {
		return types.NewFreezeList()
	}

	list := types.NewFreezeList()
	err := json.Unmarshal(b, &list)
	if err != nil {
		return types.NewFreezeList()
	}

	return list

}

// SetFreezeAddrs
func (k Keeper) SetFreezeAddrs(ctx sdk.Context, list types.FreezeList) {

	store := ctx.KVStore(k.storeKey)

	bz, err := json.Marshal(list)
	if err != nil {
		return
	}

	store.Set(types.FreezeKey, bz)

}
