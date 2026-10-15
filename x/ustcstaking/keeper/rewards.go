package keeper

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetRewardState(ctx sdk.Context) types.RewardState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RewardStateKey)
	if bz == nil {
		// Genesis and migrations initialize state explicitly. Missing runtime
		// state is corruption and must fail scalar validation, not erase liabilities.
		return types.RewardState{}
	}
	var state types.RewardState
	k.cdc.MustUnmarshal(bz, &state)
	return state
}

func (k Keeper) SetRewardState(ctx sdk.Context, state types.RewardState) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.RewardStateKey, k.cdc.MustMarshal(&state))
}

func (k Keeper) AccruedRewards(position types.Position, state types.RewardState) math.Int {
	return keeperAccruedRewards(position, state)
}

func keeperAccruedRewards(position types.Position, state types.RewardState) math.Int {
	if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE || position.Shares.IsNil() || !position.Shares.IsPositive() {
		if position.ClaimableRewards.Amount.IsNil() {
			return math.ZeroInt()
		}
		return position.ClaimableRewards.Amount
	}

	index := state.RewardIndex
	if index.IsNil() {
		index = math.LegacyZeroDec()
	}
	debt := position.RewardDebt
	if debt.IsNil() {
		debt = math.LegacyZeroDec()
	}
	raw := position.Shares.ToLegacyDec().Mul(index).Sub(debt)
	if raw.IsNegative() {
		return math.ZeroInt()
	}
	return raw.TruncateInt()
}

func (k Keeper) FundRewards(ctx sdk.Context, amount sdk.Coin) error {
	if amount.Denom != types.BondDenom || amount.Amount.IsNil() || !amount.Amount.IsPositive() {
		return types.ErrInvalidDenom.Wrapf("expected positive %s funding amount", types.BondDenom)
	}
	cacheCtx, write := ctx.CacheContext()
	state := k.GetRewardState(cacheCtx)
	if err := validateStreamingScalars(state, k.GetParams(cacheCtx)); err != nil {
		return err
	}
	if err := validateProjectedRewardIndex(state, amount.Amount); err != nil {
		return err
	}
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return err
	}
	state = k.GetRewardState(cacheCtx)
	unallocated, err := checkedRewardAdd(state.UnallocatedRewards, amount.Amount)
	if err != nil {
		return err
	}
	state.UnallocatedRewards = unallocated
	if err := k.communityPoolKeeper.DistributeFromCommunityPoolToModule(cacheCtx, sdk.NewCoins(amount), types.RewardPoolName); err != nil {
		return err
	}
	k.SetRewardState(cacheCtx, state)
	write()
	return nil
}
