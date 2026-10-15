package keeper

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// TestStreamingAccrualMatchesIndependentReferenceModel exercises many
// settlement boundaries against a rational model that does not call the
// production accrual helper.
func TestStreamingAccrualMatchesIndependentReferenceModel(t *testing.T) {
	rng := rand.New(rand.NewSource(20261014))
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	params, state := streamingTestState(now, math.NewInt(173), math.NewInt(997), math.NewInt(1_000_000))

	day := big.NewInt(rewardDayNanos)
	scale := big.NewInt(rewardIndexScale)
	shares := big.NewInt(997)
	reserve := big.NewInt(1_000_000)
	allocated := new(big.Int)
	indexRaw := new(big.Int)
	emissionFraction := new(big.Rat)
	indexRemainder := new(big.Int)

	for step := 0; step < 3000; step++ {
		// Funding and governance parameter changes happen between settlements.
		if rng.Intn(11) == 0 {
			funding := big.NewInt(int64(rng.Intn(20_000) + 1))
			reserve.Add(reserve, funding)
			state.UnallocatedRewards = math.NewIntFromBigInt(reserve)
		}
		if rng.Intn(7) == 0 {
			params.RewardRatePerDay = math.NewInt(int64(rng.Intn(500)))
		}
		if rng.Intn(13) == 0 {
			params.Paused = !params.Paused
		}

		delta := time.Duration(rng.Intn(int(48*time.Hour)) + 1)
		now = now.Add(delta)
		active := !params.Paused && params.RewardRatePerDay.IsPositive() && reserve.Sign() > 0
		var expectedRelease *big.Int
		if active {
			fraction := new(big.Rat).SetInt(params.RewardRatePerDay.BigInt())
			fraction.Mul(fraction, new(big.Rat).SetInt64(int64(delta)))
			fraction.Quo(fraction, new(big.Rat).SetInt(day))
			fraction.Add(fraction, emissionFraction)
			expectedRelease = new(big.Int).Quo(fraction.Num(), fraction.Denom())
			emissionFraction.Sub(fraction, new(big.Rat).SetInt(expectedRelease))
			if expectedRelease.Cmp(reserve) >= 0 {
				expectedRelease.Set(reserve)
				emissionFraction.SetInt64(0)
			}
		} else {
			expectedRelease = new(big.Int)
			emissionFraction.SetInt64(0)
		}
		reserve.Sub(reserve, expectedRelease)
		allocated.Add(allocated, expectedRelease)
		indexNumerator := new(big.Int).Mul(expectedRelease, scale)
		indexNumerator.Add(indexNumerator, indexRemainder)
		increment, nextRemainder := new(big.Int), new(big.Int)
		increment.QuoRem(indexNumerator, shares, nextRemainder)
		indexRaw.Add(indexRaw, increment)
		indexRemainder.Set(nextRemainder)

		var gotRelease math.Int
		var err error
		state, gotRelease, err = advanceRewardState(state, params, now)
		require.NoError(t, err, "step %d", step)
		require.Equal(t, expectedRelease.String(), gotRelease.String(), "released at step %d", step)
		require.Equal(t, reserve.String(), state.UnallocatedRewards.String(), "reserve at step %d", step)
		require.Equal(t, allocated.String(), state.AllocatedRewards.String(), "allocated at step %d", step)
		require.Equal(t, indexRaw.String(), state.RewardIndex.BigInt().String(), "index at step %d", step)
		emissionCarry := new(big.Int).Mul(emissionFraction.Num(), day)
		emissionCarry.Quo(emissionCarry, emissionFraction.Denom())
		require.Equal(t, emissionCarry.String(), state.EmissionRemainder.String(), "emission carry at step %d", step)
		require.Equal(t, indexRemainder.String(), state.IndexRemainder.String(), "index carry at step %d", step)
	}
}
