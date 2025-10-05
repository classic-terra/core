package types_test

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/oracle/types"
)

func TestToMap(t *testing.T) {
	tests := struct {
		votes   []types.VoteForTally
		isValid []bool
	}{
		[]types.VoteForTally{
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Denom:        core.MicroKRWDenom,
				ExchangeRate: sdkmath.LegacyNewDec(1600),
				Power:        100,
			},
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Denom:        core.MicroKRWDenom,
				ExchangeRate: sdkmath.LegacyZeroDec(),
				Power:        100,
			},
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Denom:        core.MicroKRWDenom,
				ExchangeRate: sdkmath.LegacyNewDec(1500),
				Power:        100,
			},
		},
		[]bool{true, false, true},
	}

	pb := types.ExchangeRateBallot(tests.votes)
	mapData := pb.ToMap()
	for i, vote := range tests.votes {
		exchangeRate, ok := mapData[string(vote.Voter)]
		if tests.isValid[i] {
			require.True(t, ok)
			require.Equal(t, exchangeRate, vote.ExchangeRate)
		} else {
			require.False(t, ok)
		}
	}
}

func TestToCrossRate(t *testing.T) {
	data := []struct {
		base     sdkmath.LegacyDec
		quote    sdkmath.LegacyDec
		expected sdkmath.LegacyDec
	}{
		{
			base:     sdkmath.LegacyNewDec(1600),
			quote:    sdkmath.LegacyNewDec(100),
			expected: sdkmath.LegacyNewDec(16),
		},
		{
			base:     sdkmath.LegacyNewDec(0),
			quote:    sdkmath.LegacyNewDec(100),
			expected: sdkmath.LegacyNewDec(16),
		},
		{
			base:     sdkmath.LegacyNewDec(1600),
			quote:    sdkmath.LegacyNewDec(0),
			expected: sdkmath.LegacyNewDec(16),
		},
	}

	pbBase := types.ExchangeRateBallot{}
	pbQuote := types.ExchangeRateBallot{}
	cb := types.ExchangeRateBallot{}
	for _, data := range data {
		valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		if !data.base.IsZero() {
			pbBase = append(pbBase, types.NewVoteForTally(data.base, core.MicroKRWDenom, valAddr, 100))
		}

		pbQuote = append(pbQuote, types.NewVoteForTally(data.quote, core.MicroKRWDenom, valAddr, 100))

		if !data.base.IsZero() && !data.quote.IsZero() {
			cb = append(cb, types.NewVoteForTally(data.base.Quo(data.quote), core.MicroKRWDenom, valAddr, 100))
		} else {
			cb = append(cb, types.NewVoteForTally(sdkmath.LegacyZeroDec(), core.MicroKRWDenom, valAddr, 0))
		}
	}

	baseMapBallot := pbBase.ToMap()
	require.Equal(t, cb, pbQuote.ToCrossRate(baseMapBallot))

	sort.Sort(cb)

	require.Equal(t, cb, pbQuote.ToCrossRateWithSort(baseMapBallot))
}

func TestSqrt(t *testing.T) {
	num := sdkmath.LegacyNewDecWithPrec(144, 4)
	floatNum, err := strconv.ParseFloat(num.String(), 64)
	require.NoError(t, err)

	floatNum = math.Sqrt(floatNum)
	num, err = sdkmath.LegacyNewDecFromStr(fmt.Sprintf("%f", floatNum))
	require.NoError(t, err)

	require.Equal(t, sdkmath.LegacyNewDecWithPrec(12, 2), num)
}

func TestPBPower(t *testing.T) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
	_, valAccAddrs, sk := types.GenerateRandomTestCase()
	pb := types.ExchangeRateBallot{}
	ballotPower := int64(0)

	for i := 0; i < len(sk.Validators()); i++ {
		v, _ := sk.Validator(ctx, valAccAddrs[i])
		power := v.GetConsensusPower(sdk.DefaultPowerReduction)
		vote := types.NewVoteForTally(
			sdkmath.LegacyZeroDec(),
			core.MicroSDRDenom,
			valAccAddrs[i],
			power,
		)

		pb = append(pb, vote)

		require.NotEqual(t, int64(0), vote.Power)

		ballotPower += vote.Power
	}

	require.Equal(t, ballotPower, pb.Power())

	// Mix in a fake validator, the total power should not have changed.
	pubKey := secp256k1.GenPrivKey().PubKey()
	faceValAddr := sdk.ValAddress(pubKey.Address())
	fakeVote := types.NewVoteForTally(
		sdkmath.LegacyOneDec(),
		core.MicroSDRDenom,
		faceValAddr,
		0,
	)

	pb = append(pb, fakeVote)
	require.Equal(t, ballotPower, pb.Power())
}

func TestPBWeightedMedian(t *testing.T) {
	tests := []struct {
		inputs      []int64
		weights     []int64
		isValidator []bool
		median      sdkmath.LegacyDec
		panic       bool
	}{
		{
			// Supermajority one number
			[]int64{1, 2, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(10),
			false,
		},
		{
			// Adding fake validator doesn't change outcome
			[]int64{1, 2, 10, 100000, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdkmath.LegacyNewDec(10),
			false,
		},
		{
			// Tie votes
			[]int64{1, 2, 3, 4},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(2),
			false,
		},
		{
			// No votes
			[]int64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(0),
			false,
		},
		{
			// not sorted panic
			[]int64{2, 1, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(10),
			true,
		},
	}

	for _, tc := range tests {
		pb := types.ExchangeRateBallot{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewVoteForTally(
				sdkmath.LegacyNewDec(input),
				core.MicroSDRDenom,
				valAddr,
				power,
			)

			pb = append(pb, vote)
		}

		if tc.panic {
			require.Panics(t, func() {
				if !sort.IsSorted(pb) {
					panic("ballot must be sorted")
				}
			})
		} else {
			require.Equal(t, tc.median, pb.WeightedMedian())
		}
	}
}

func TestPBStandardDeviation(t *testing.T) {
	tests := []struct {
		inputs            []float64
		weights           []int64
		isValidator       []bool
		standardDeviation sdkmath.LegacyDec
	}{
		{
			// Supermajority one number
			[]float64{1.0, 2.0, 10.0, 100000.0},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDecWithPrec(4999500036300, types.OracleDecPrecision),
		},
		{
			// Adding fake validator doesn't change outcome
			[]float64{1.0, 2.0, 10.0, 100000.0, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdkmath.LegacyNewDecWithPrec(447213595075100600, types.OracleDecPrecision),
		},
		{
			// Tie votes
			[]float64{1.0, 2.0, 3.0, 4.0},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDecWithPrec(122474500, types.OracleDecPrecision),
		},
		{
			// No votes
			[]float64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDecWithPrec(0, 0),
		},
	}

	base := math.Pow10(types.OracleDecPrecision)
	for _, tc := range tests {
		pb := types.ExchangeRateBallot{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewVoteForTally(
				sdkmath.LegacyNewDecWithPrec(int64(input*base), int64(types.OracleDecPrecision)),
				core.MicroSDRDenom,
				valAddr,
				power,
			)

			pb = append(pb, vote)
		}

		require.Equal(t, tc.standardDeviation, pb.StandardDeviation(pb.WeightedMedian()))
	}
}

func TestPBStandardDeviationOverflow(t *testing.T) {
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	exchangeRate, err := sdkmath.LegacyNewDecFromStr("100000000000000000000000000000000000000000000000000000000.0")
	require.NoError(t, err)

	pb := types.ExchangeRateBallot{types.NewVoteForTally(
		sdkmath.LegacyZeroDec(),
		core.MicroSDRDenom,
		valAddr,
		2,
	), types.NewVoteForTally(
		exchangeRate,
		core.MicroSDRDenom,
		valAddr,
		1,
	)}

	require.Equal(t, sdkmath.LegacyZeroDec(), pb.StandardDeviation(pb.WeightedMedian()))
}

func TestNewClaim(t *testing.T) {
	power := int64(10)
	weight := int64(11)
	winCount := int64(1)
	addr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes())
	claim := types.NewClaim(power, weight, winCount, addr)
	require.Equal(t, types.Claim{
		Power:     power,
		Weight:    weight,
		WinCount:  winCount,
		Recipient: addr,
	}, claim)
}
