package keeper

import (
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func streamingTestState(now time.Time, rate, shares, reserve math.Int) (types.Params, types.RewardState) {
	params := types.DefaultParams()
	params.RewardRatePerDay = rate
	state := types.RewardState{
		RewardIndex: math.LegacyZeroDec(), TotalShares: shares,
		UnallocatedRewards: reserve, AllocatedRewards: math.ZeroInt(), LastRewardTime: &now,
		EmissionRemainder: math.ZeroInt(), IndexRemainder: math.ZeroInt(),
	}
	return params, state
}

func TestStreamingOneDay(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.NewInt(100), math.NewInt(100), math.NewInt(1000))

	got, released, err := advanceRewardState(state, params, start.Add(24*time.Hour))

	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), released)
	require.Equal(t, math.NewInt(900), got.UnallocatedRewards)
	require.Equal(t, math.NewInt(100), got.AllocatedRewards)
	require.True(t, got.RewardIndex.Equal(math.LegacyOneDec()))
	require.True(t, state.RewardIndex.IsZero(), "input state must not be mutated")
}

func TestStreamingSplitsAndRateChangesUseElapsedConsensusTime(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.NewInt(100), math.NewInt(100), math.NewInt(1000))
	state, first, err := advanceRewardState(state, params, start.Add(12*time.Hour))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), first)

	params.RewardRatePerDay = math.NewInt(200)
	state, second, err := advanceRewardState(state, params, start.Add(24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), second)
	require.Equal(t, math.NewInt(150), state.AllocatedRewards)
	require.Equal(t, math.NewInt(850), state.UnallocatedRewards)
}

func TestStreamingPauseAndEmptyShareIntervalsDoNotBacklog(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.NewInt(100), math.ZeroInt(), math.NewInt(1000))
	state, released, err := advanceRewardState(state, params, start.Add(24*time.Hour))
	require.NoError(t, err)
	require.True(t, released.IsZero())
	require.True(t, state.EmissionRemainder.IsZero())

	state.TotalShares = math.NewInt(100)
	state, released, err = advanceRewardState(state, params, start.Add(36*time.Hour))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), released)

	params.Paused = true
	state, released, err = advanceRewardState(state, params, start.Add(72*time.Hour))
	require.NoError(t, err)
	require.True(t, released.IsZero())
	require.True(t, state.EmissionRemainder.IsZero())

	params.Paused = false
	state, released, err = advanceRewardState(state, params, start.Add(84*time.Hour))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), released)
	require.Equal(t, math.NewInt(100), state.AllocatedRewards)
}

func TestStreamingReserveExhaustionClearsEmissionCarry(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.NewInt(100), math.NewInt(100), math.NewInt(50))
	state.EmissionRemainder = math.NewInt(1)

	got, released, err := advanceRewardState(state, params, start.Add(24*time.Hour))

	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), released)
	require.True(t, got.UnallocatedRewards.IsZero())
	require.True(t, got.EmissionRemainder.IsZero())
	require.Equal(t, math.NewInt(50), got.AllocatedRewards)
}

func TestStreamingHourlyAccrualAndIndexDivisionCarry(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.OneInt(), math.OneInt(), math.OneInt())
	for i := 1; i <= 24; i++ {
		var released math.Int
		var err error
		state, released, err = advanceRewardState(state, params, start.Add(time.Duration(i)*time.Hour))
		require.NoError(t, err)
		if i < 24 {
			require.True(t, released.IsZero())
		} else {
			require.Equal(t, math.OneInt(), released)
		}
	}
	require.True(t, state.RewardIndex.Equal(math.LegacyOneDec()))

	largeShares := math.NewIntFromBigInt(new(big.Int).SetUint64(10_000_000_000_000_000_000))
	params, state = streamingTestState(start, math.NewInt(10), largeShares, math.NewInt(10))
	for i := 1; i <= 10; i++ {
		var released math.Int
		var err error
		state, released, err = advanceRewardState(state, params, start.Add(time.Duration(i)*24*time.Hour/10))
		require.NoError(t, err)
		require.Equal(t, math.OneInt(), released)
	}
	require.True(t, state.RewardIndex.Equal(math.LegacyNewDecWithPrec(1, 18)))
}

func TestStreamingRejectsBackwardsTimeAndUninitializedState(t *testing.T) {
	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(start, math.OneInt(), math.OneInt(), math.OneInt())
	_, _, err := advanceRewardState(state, params, start.Add(-time.Nanosecond))
	require.ErrorIs(t, err, types.ErrInvalidRewardState)

	state.IndexRemainder = math.Int{}
	_, _, err = advanceRewardState(state, params, start.Add(time.Second))
	require.ErrorIs(t, err, types.ErrInvalidRewardState)
}

func TestAccrueRewardsMovesReserveWithoutMovingBankFunds(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	keeper.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100))
	state.UnallocatedRewards = math.NewInt(100)
	keeper.SetRewardState(ctx, state)
	bank := keeper.bankKeeper.(*recordingBankKeeper)
	bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(100))

	next := ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))
	require.NoError(t, keeper.AccrueRewards(next))

	got := keeper.GetRewardState(next)
	require.True(t, got.UnallocatedRewards.IsZero())
	require.Equal(t, math.NewInt(100), got.AllocatedRewards)
	require.Equal(t, sdk.NewCoin(types.BondDenom, math.NewInt(100)), bank.rewardPoolBalance)
	require.Len(t, next.EventManager().Events(), 1)
}

func TestAccrueRewardsRejectsUnbackedReserveWithoutWriting(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100))
	state.UnallocatedRewards = math.NewInt(100)
	keeper.SetRewardState(ctx, state)
	before := keeper.GetRewardState(ctx)

	err := keeper.AccrueRewards(ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour)))

	require.ErrorIs(t, err, types.ErrRewardPoolInsolvent)
	require.Equal(t, before, keeper.GetRewardState(ctx))
	require.Empty(t, ctx.EventManager().Events())
}

func TestStreamingArithmeticBounds(t *testing.T) {
	maxIntBig := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), math.MaxBitLen), big.NewInt(1))
	maxInt, err := checkedRewardInt(maxIntBig)
	require.NoError(t, err)
	require.Equal(t, maxIntBig.String(), maxInt.String())
	_, err = checkedRewardInt(new(big.Int).Lsh(big.NewInt(1), math.MaxBitLen))
	require.ErrorIs(t, err, types.ErrInvalidRewardState)

	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	_, state := streamingTestState(start, math.ZeroInt(), math.OneInt(), math.ZeroInt())
	state.RewardIndex = math.LegacyNewDecFromBigIntWithPrec(legacyDecimalMaxRaw, math.LegacyPrecision)
	require.True(t, state.RewardIndex.IsInValidRange())
	require.ErrorIs(t, validateProjectedRewardIndex(state, math.OneInt()), types.ErrInvalidRewardState)
}
