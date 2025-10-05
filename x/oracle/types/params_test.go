package types_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/classic-terra/core/v3/x/oracle/types"

	sdkmath "cosmossdk.io/math"
)

func TestParamsEqual(t *testing.T) {
	p1 := types.DefaultParams()
	err := p1.Validate()
	require.NoError(t, err)

	// minus vote period
	p1.VotePeriod = 0
	err = p1.Validate()
	require.Error(t, err)

	// small vote threshold
	p2 := types.DefaultParams()
	p2.VoteThreshold = sdkmath.LegacyZeroDec()
	err = p2.Validate()
	require.Error(t, err)

	// negative reward band
	p3 := types.DefaultParams()
	p3.RewardBand = sdkmath.LegacyNewDecWithPrec(-1, 2)
	err = p3.Validate()
	require.Error(t, err)

	// negative slash fraction
	p4 := types.DefaultParams()
	p4.SlashFraction = sdkmath.LegacyNewDec(-1)
	err = p4.Validate()
	require.Error(t, err)

	// negative min valid per window
	p5 := types.DefaultParams()
	p5.MinValidPerWindow = sdkmath.LegacyNewDec(-1)
	err = p5.Validate()
	require.Error(t, err)

	// small slash window
	p6 := types.DefaultParams()
	p6.SlashWindow = 0
	err = p6.Validate()
	require.Error(t, err)

	// small distribution window
	p7 := types.DefaultParams()
	p7.RewardDistributionWindow = 0
	err = p7.Validate()
	require.Error(t, err)

	// non-positive tobin tax
	p8 := types.DefaultParams()
	p8.Whitelist[0].Name = ""
	err = p8.Validate()
	require.Error(t, err)

	// invalid name tobin tax
	p9 := types.DefaultParams()
	p9.Whitelist[0].TobinTax = sdkmath.LegacyNewDec(-1)
	err = p9.Validate()
	require.Error(t, err)

	// empty name
	p10 := types.DefaultParams()
	p10.Whitelist[0].Name = ""
	err = p10.Validate()
	require.Error(t, err)

	p11 := types.DefaultParams()
	require.NotNil(t, p11.ParamSetPairs())
	require.NotNil(t, p11.String())
}

func TestValidate(t *testing.T) {
	p1 := types.DefaultParams()
	pairs := p1.ParamSetPairs()
	for _, pair := range pairs {
		switch {
		case bytes.Equal(types.KeyVotePeriod, pair.Key) ||
			bytes.Equal(types.KeyRewardDistributionWindow, pair.Key) ||
			bytes.Equal(types.KeySlashWindow, pair.Key):
			require.NoError(t, pair.ValidatorFn(uint64(1)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(uint64(0)))
		case bytes.Equal(types.KeyVoteThreshold, pair.Key):
			require.NoError(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(33, 2)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(32, 2)))
			require.Error(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(101, 2)))
		case bytes.Equal(types.KeyRewardBand, pair.Key) ||
			bytes.Equal(types.KeySlashFraction, pair.Key) ||
			bytes.Equal(types.KeyMinValidPerWindow, pair.Key):
			require.NoError(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(7, 2)))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(-1, 2)))
			require.Error(t, pair.ValidatorFn(sdkmath.LegacyNewDecWithPrec(101, 2)))
		case bytes.Equal(types.KeyWhitelist, pair.Key):
			require.NoError(t, pair.ValidatorFn(types.DenomList{
				{
					Name:     "denom",
					TobinTax: sdkmath.LegacyNewDecWithPrec(10, 2),
				},
			}))
			require.Error(t, pair.ValidatorFn("invalid"))
			require.Error(t, pair.ValidatorFn(types.DenomList{
				{
					Name:     "",
					TobinTax: sdkmath.LegacyNewDecWithPrec(10, 2),
				},
			}))
			require.Error(t, pair.ValidatorFn(types.DenomList{
				{
					Name:     "denom",
					TobinTax: sdkmath.LegacyNewDecWithPrec(101, 2),
				},
			}))
			require.Error(t, pair.ValidatorFn(types.DenomList{
				{
					Name:     "denom",
					TobinTax: sdkmath.LegacyNewDecWithPrec(-1, 2),
				},
			}))
		}
	}
}
