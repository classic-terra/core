package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetPosition(ctx sdk.Context, id uint64) (types.Position, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	bz := store.Get(sdk.Uint64ToBigEndian(id))
	if bz == nil {
		return types.Position{}, false
	}
	var position types.Position
	k.cdc.MustUnmarshal(bz, &position)
	return position, true
}

func (k Keeper) SetPosition(ctx sdk.Context, position types.Position) error {
	if !position.ClaimableRewards.Amount.IsNil() && position.ClaimableRewards.Amount.IsNegative() {
		return types.ErrInvalidPosition.Wrap("negative claimable rewards")
	}
	owner, err := sdk.AccAddressFromBech32(position.Owner)
	if err != nil {
		return types.ErrInvalidPosition.Wrapf("position %d owner: %s", position.Id, err)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	reserved, err := k.GetReservedRewardsChecked(ctx)
	if err != nil {
		return err
	}
	if existing, found := k.GetPosition(ctx, position.Id); found {
		if existing.Owner != position.Owner {
			return types.ErrPositionOwner
		}
		reserved = reserved.Sub(claimableAmount(existing))
	}
	reserved = reserved.Add(claimableAmount(position))
	if reserved.IsNegative() {
		return types.ErrInvalidRewardState.Wrap("negative reserved rewards")
	}
	store.Set(sdk.Uint64ToBigEndian(position.Id), k.cdc.MustMarshal(&position))
	k.setReservedRewards(ctx, reserved)
	ownerIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.OwnerPositionKeyPrefix)
	prefix.NewStore(ownerIndex, types.OwnerPositionPrefix(owner)).Set(sdk.Uint64ToBigEndian(position.Id), []byte{1})
	return nil
}

func (k Keeper) DeletePosition(ctx sdk.Context, id uint64) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	position, found := k.GetPosition(ctx, id)
	if !found {
		return nil
	}
	owner, err := sdk.AccAddressFromBech32(position.Owner)
	if err != nil {
		return types.ErrInvalidPosition.Wrapf("position %d owner: %s", id, err)
	}
	reserved, err := k.GetReservedRewardsChecked(ctx)
	if err != nil {
		return err
	}
	reserved = reserved.Sub(claimableAmount(position))
	if reserved.IsNegative() {
		return types.ErrInvalidRewardState.Wrap("negative reserved rewards")
	}
	store.Delete(sdk.Uint64ToBigEndian(id))
	k.setReservedRewards(ctx, reserved)
	ownerIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.OwnerPositionKeyPrefix)
	prefix.NewStore(ownerIndex, types.OwnerPositionPrefix(owner)).Delete(sdk.Uint64ToBigEndian(id))
	return nil
}

func claimableAmount(p types.Position) math.Int {
	if p.ClaimableRewards.Amount.IsNil() {
		return math.ZeroInt()
	}
	return p.ClaimableRewards.Amount
}

// GetReservedRewards is a derived aggregate of integer claimable obligations.
// Position writes maintain it; genesis imports rebuild it from position records.
func (k Keeper) GetReservedRewards(ctx sdk.Context) math.Int {
	amount, err := k.GetReservedRewardsChecked(ctx)
	if err != nil {
		panic(err)
	}
	return amount
}

func (k Keeper) GetReservedRewardsChecked(ctx sdk.Context) (math.Int, error) {
	bz := ctx.KVStore(k.storeKey).Get(types.ReservedRewardsKey)
	if bz == nil {
		return math.ZeroInt(), nil
	}
	amount, ok := math.NewIntFromString(string(bz))
	if !ok || amount.IsNegative() {
		return math.Int{}, types.ErrInvalidRewardState.Wrap("invalid stored reserved reward total")
	}
	return amount, nil
}

func (k Keeper) setReservedRewards(ctx sdk.Context, amount math.Int) {
	ctx.KVStore(k.storeKey).Set(types.ReservedRewardsKey, []byte(amount.String()))
}

// RebuildDerivedState reconstructs the owner index and claim reserve on import.
// Neither derived representation is independently serialized in genesis.
func (k Keeper) RebuildDerivedState(ctx sdk.Context) error {
	index := prefix.NewStore(ctx.KVStore(k.storeKey), types.OwnerPositionKeyPrefix)
	iter := index.Iterator(nil, nil)
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, append([]byte(nil), iter.Key()...))
	}
	iter.Close()
	for _, key := range keys {
		index.Delete(key)
	}
	reserved := math.ZeroInt()
	var err error
	k.IteratePositions(ctx, func(p types.Position) bool {
		var owner sdk.AccAddress
		owner, err = sdk.AccAddressFromBech32(p.Owner)
		if err != nil {
			return true
		}
		index.Set(types.OwnerPositionKey(owner, p.Id), []byte{1})
		reserved = reserved.Add(claimableAmount(p))
		return false
	})
	if err != nil {
		return err
	}
	k.setReservedRewards(ctx, reserved)
	return nil
}

func (k Keeper) saveOrPruneSettledPosition(ctx sdk.Context, position types.Position) error {
	if position.Status == types.PositionStatus_POSITION_STATUS_WITHDRAWN &&
		!position.Principal.Amount.IsNil() && position.Principal.Amount.IsZero() &&
		!position.Shares.IsNil() && position.Shares.IsZero() &&
		!position.RewardDebt.IsNil() && position.RewardDebt.IsZero() &&
		!position.ClaimableRewards.Amount.IsNil() && position.ClaimableRewards.Amount.IsZero() {
		return k.DeletePosition(ctx, position.Id)
	}
	return k.SetPosition(ctx, position)
}

func (k Keeper) IteratePositions(ctx sdk.Context, callback func(types.Position) bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		k.cdc.MustUnmarshal(iterator.Value(), &position)
		if callback(position) {
			return
		}
	}
}
