package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type Keeper struct {
	cdc                 codec.BinaryCodec
	storeKey            storetypes.StoreKey
	bankKeeper          types.BankKeeper
	communityPoolKeeper types.CommunityPoolKeeper
}

func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, bankKeeper types.BankKeeper, communityPoolKeeper types.CommunityPoolKeeper) Keeper {
	return Keeper{cdc: cdc, storeKey: storeKey, bankKeeper: bankKeeper, communityPoolKeeper: communityPoolKeeper}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority is immutable; persisted params cannot delegate governance powers.
func (k Keeper) GetAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}

func (k Keeper) GetNextPositionID(ctx sdk.Context) uint64 {
	bz := ctx.KVStore(k.storeKey).Get(types.NextPositionIDKey)
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

func (k Keeper) SetNextPositionID(ctx sdk.Context, id uint64) {
	ctx.KVStore(k.storeKey).Set(types.NextPositionIDKey, sdk.Uint64ToBigEndian(id))
}
