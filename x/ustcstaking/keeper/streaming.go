package keeper

import (
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	rewardDayNanos   int64 = 86_400_000_000_000
	rewardIndexScale int64 = 1_000_000_000_000_000_000
)

var (
	rewardDayNanosBig   = big.NewInt(rewardDayNanos)
	rewardIndexScaleBig = big.NewInt(rewardIndexScale)
	legacyDecimalMaxRaw = func() *big.Int {
		max := new(big.Int).Lsh(big.NewInt(1), math.MaxBitLen)
		max.Mul(max, rewardIndexScaleBig)
		return max.Sub(max, big.NewInt(1))
	}()
)

func advanceRewardState(state types.RewardState, params types.Params, now time.Time) (types.RewardState, math.Int, error) {
	if err := validateStreamingScalars(state, params); err != nil {
		return state, math.Int{}, err
	}
	if state.LastRewardTime == nil {
		return state, math.Int{}, types.ErrInvalidRewardState.Wrap("last reward time is missing")
	}
	if now.Before(*state.LastRewardTime) {
		return state, math.Int{}, types.ErrInvalidRewardState.Wrap("consensus time moved backwards")
	}
	result := cloneRewardState(state)
	if now.Equal(*state.LastRewardTime) {
		return result, math.ZeroInt(), nil
	}
	nowCopy := now
	result.LastRewardTime = &nowCopy

	if params.Paused || params.RewardRatePerDay.IsZero() || state.TotalShares.IsZero() || state.UnallocatedRewards.IsZero() {
		result.EmissionRemainder = math.ZeroInt()
		if state.TotalShares.IsZero() {
			result.IndexRemainder = math.ZeroInt()
		}
		return result, math.ZeroInt(), nil
	}

	elapsed := elapsedNanoseconds(*state.LastRewardTime, now)
	numerator := new(big.Int).Mul(params.RewardRatePerDay.BigInt(), elapsed)
	numerator.Add(numerator, state.EmissionRemainder.BigInt())
	due, nextEmissionRemainder := new(big.Int), new(big.Int)
	due.QuoRem(numerator, rewardDayNanosBig, nextEmissionRemainder)

	released := new(big.Int).Set(due)
	unallocated := state.UnallocatedRewards.BigInt()
	exhausted := released.Cmp(unallocated) >= 0
	if exhausted {
		released.Set(unallocated)
		nextEmissionRemainder.SetInt64(0)
	}

	indexNumerator := new(big.Int).Mul(released, rewardIndexScaleBig)
	indexNumerator.Add(indexNumerator, state.IndexRemainder.BigInt())
	indexIncrement, nextIndexRemainder := new(big.Int), new(big.Int)
	indexIncrement.QuoRem(indexNumerator, state.TotalShares.BigInt(), nextIndexRemainder)
	newIndexScaled := new(big.Int).Add(state.RewardIndex.BigInt(), indexIncrement)
	newIndex := math.LegacyNewDecFromBigIntWithPrec(newIndexScaled, math.LegacyPrecision)
	if !newIndex.IsInValidRange() {
		return state, math.Int{}, types.ErrInvalidRewardState.Wrap("reward index exceeds SDK decimal range")
	}
	if !shareIndexProductInRange(state.TotalShares, newIndex) {
		return state, math.Int{}, types.ErrInvalidRewardState.Wrap("reward debt would exceed SDK decimal range")
	}

	releasedInt, err := checkedRewardInt(released)
	if err != nil {
		return state, math.Int{}, err
	}
	newAllocated, err := checkedRewardInt(new(big.Int).Add(state.AllocatedRewards.BigInt(), released))
	if err != nil {
		return state, math.Int{}, err
	}
	result.RewardIndex = newIndex
	result.UnallocatedRewards = math.NewIntFromBigInt(new(big.Int).Sub(unallocated, released))
	result.AllocatedRewards = newAllocated
	result.EmissionRemainder = math.NewIntFromBigInt(nextEmissionRemainder)
	result.IndexRemainder = math.NewIntFromBigInt(nextIndexRemainder)
	return result, releasedInt, nil
}

func validateStreamingScalars(state types.RewardState, params types.Params) error {
	if state.LastRewardTime == nil {
		return types.ErrInvalidRewardState.Wrap("last reward time is not initialized")
	}
	if params.RewardRatePerDay.IsNil() || params.RewardRatePerDay.IsNegative() || params.RewardRatePerDay.BigInt().BitLen() > math.MaxBitLen {
		return types.ErrInvalidRewardRate.Wrap("rate must be an initialized, non-negative SDK integer")
	}
	if state.RewardIndex.IsNil() || !state.RewardIndex.IsInValidRange() || state.RewardIndex.IsNegative() ||
		state.TotalShares.IsNil() || state.TotalShares.IsNegative() || state.TotalShares.BigInt().BitLen() > math.MaxBitLen ||
		state.UnallocatedRewards.IsNil() || state.UnallocatedRewards.IsNegative() || state.UnallocatedRewards.BigInt().BitLen() > math.MaxBitLen ||
		state.AllocatedRewards.IsNil() || state.AllocatedRewards.IsNegative() || state.AllocatedRewards.BigInt().BitLen() > math.MaxBitLen ||
		state.EmissionRemainder.IsNil() || state.EmissionRemainder.IsNegative() ||
		state.IndexRemainder.IsNil() || state.IndexRemainder.IsNegative() {
		return types.ErrInvalidRewardState.Wrap("streaming state contains uninitialized, negative, or out-of-range scalars")
	}
	if state.EmissionRemainder.GTE(math.NewInt(rewardDayNanos)) {
		return types.ErrInvalidRewardState.Wrap("emission remainder is outside its denominator")
	}
	if state.TotalShares.IsZero() && !state.IndexRemainder.IsZero() ||
		state.TotalShares.IsPositive() && state.IndexRemainder.GTE(state.TotalShares) {
		return types.ErrInvalidRewardState.Wrap("index remainder is outside its share denominator")
	}
	if err := timestamppb.New(*state.LastRewardTime).CheckValid(); err != nil {
		return types.ErrInvalidRewardState.Wrapf("last reward time is invalid: %s", err)
	}
	return nil
}

func elapsedNanoseconds(from, to time.Time) *big.Int {
	seconds := new(big.Int).Sub(big.NewInt(to.Unix()), big.NewInt(from.Unix()))
	seconds.Mul(seconds, big.NewInt(1_000_000_000))
	nanos := int64(to.Nanosecond() - from.Nanosecond())
	return seconds.Add(seconds, big.NewInt(nanos))
}

func checkedRewardInt(value *big.Int) (math.Int, error) {
	if value == nil || value.Sign() < 0 || value.BitLen() > math.MaxBitLen {
		return math.Int{}, types.ErrInvalidRewardState.Wrap("reward accounting exceeds SDK integer range")
	}
	return math.NewIntFromBigInt(value), nil
}

func cloneRewardState(state types.RewardState) types.RewardState {
	cloneInt := func(value math.Int) math.Int {
		if value.IsNil() {
			return math.Int{}
		}
		return math.NewIntFromBigInt(value.BigInt())
	}
	clone := state
	if !state.RewardIndex.IsNil() {
		clone.RewardIndex = state.RewardIndex.Add(math.LegacyZeroDec())
	}
	clone.TotalShares = cloneInt(state.TotalShares)
	clone.UnallocatedRewards = cloneInt(state.UnallocatedRewards)
	clone.AllocatedRewards = cloneInt(state.AllocatedRewards)
	clone.EmissionRemainder = cloneInt(state.EmissionRemainder)
	clone.IndexRemainder = cloneInt(state.IndexRemainder)
	if state.LastRewardTime != nil {
		last := *state.LastRewardTime
		clone.LastRewardTime = &last
	}
	return clone
}

func (k Keeper) AccrueRewards(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	state := k.GetRewardState(ctx)
	if err := validateStreamingScalars(state, params); err != nil {
		return err
	}
	if state.LastRewardTime == nil {
		return types.ErrInvalidRewardState.Wrap("last reward time is missing")
	}
	pool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	backing, err := checkedRewardAdd(state.UnallocatedRewards, state.AllocatedRewards)
	if err != nil {
		return err
	}
	if pool.Amount.IsNil() || pool.Amount.IsNegative() || pool.Amount.LT(backing) {
		return types.ErrRewardPoolInsolvent.Wrapf("reward pool balance %s is below protected reserve %s", pool, backing)
	}
	if ctx.BlockTime().Equal(*state.LastRewardTime) {
		return nil
	}
	next, released, err := advanceRewardState(state, params, ctx.BlockTime())
	if err != nil {
		return err
	}
	k.SetRewardState(ctx, next)
	if released.IsPositive() {
		ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeRewardsAccrued,
			sdk.NewAttribute("released", sdk.NewCoin(types.BondDenom, released).String()),
			sdk.NewAttribute("timestamp", ctx.BlockTime().UTC().Format(time.RFC3339Nano)),
			sdk.NewAttribute("reward_index", next.RewardIndex.String()),
			sdk.NewAttribute("unallocated_rewards", next.UnallocatedRewards.String()),
			sdk.NewAttribute("allocated_rewards", next.AllocatedRewards.String()),
		))
	}
	return nil
}

func validateProjectedRewardIndex(state types.RewardState, extra math.Int) error {
	if state.RewardIndex.IsNil() || !state.RewardIndex.IsInValidRange() || state.TotalShares.IsNil() || state.TotalShares.IsNegative() ||
		state.UnallocatedRewards.IsNil() || state.UnallocatedRewards.IsNegative() || state.AllocatedRewards.IsNil() || state.AllocatedRewards.IsNegative() ||
		extra.IsNil() || extra.IsNegative() {
		return types.ErrInvalidRewardState.Wrap("projected reserve must be non-negative")
	}
	maximumRelease := new(big.Int).Add(state.UnallocatedRewards.BigInt(), state.AllocatedRewards.BigInt())
	maximumRelease.Add(maximumRelease, extra.BigInt())
	maximumIndexIncrement := new(big.Int).Mul(maximumRelease, rewardIndexScaleBig)
	maximumIndex := new(big.Int).Add(state.RewardIndex.BigInt(), maximumIndexIncrement)
	if !math.LegacyNewDecFromBigIntWithPrec(maximumIndex, math.LegacyPrecision).IsInValidRange() {
		return types.ErrInvalidRewardState.Wrap("projected reward index exceeds SDK decimal range")
	}
	if !shareIndexProductInRange(state.TotalShares, math.LegacyNewDecFromBigIntWithPrec(maximumIndex, math.LegacyPrecision)) {
		return types.ErrInvalidRewardState.Wrap("projected reward debt exceeds SDK decimal range")
	}
	return nil
}

func shareIndexProductInRange(shares math.Int, index math.LegacyDec) bool {
	if shares.IsNil() || shares.IsNegative() || index.IsNil() || index.IsNegative() {
		return false
	}
	rawProduct := new(big.Int).Mul(shares.BigInt(), index.BigInt())
	return rawProduct.Cmp(legacyDecimalMaxRaw) <= 0
}

func checkedRewardAdd(left, right math.Int) (math.Int, error) {
	if left.IsNil() || right.IsNil() || left.IsNegative() || right.IsNegative() {
		return math.Int{}, types.ErrInvalidRewardState.Wrap("reward addition requires initialized non-negative values")
	}
	value := new(big.Int).Add(left.BigInt(), right.BigInt())
	result, err := checkedRewardInt(value)
	if err != nil {
		return math.Int{}, fmt.Errorf("checked reward addition: %w", err)
	}
	return result, nil
}
