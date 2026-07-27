package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	params := DefaultParams()
	require.NoError(t, params.Validate())

	params = DefaultParams()
	params.TaxPolicy.RateMax = sdkmath.LegacyZeroDec()
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.TaxPolicy.RateMin = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.TaxPolicy.Cap = sdk.Coin{Denom: "foo", Amount: sdkmath.NewInt(-1)}
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.TaxPolicy.ChangeRateMax = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.RewardPolicy.RateMax = sdkmath.LegacyZeroDec()
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.RewardPolicy.ChangeRateMax = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.SeigniorageBurdenTarget = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.MiningIncrement = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.WindowLong = 0
	require.Error(t, params.Validate())

	params = DefaultParams()
	params.RewardPolicy.RateMin = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	require.NotNil(t, params.ParamSetPairs())
	require.NotNil(t, params.String())
}

func TestParams_TaxRedirectRateValidation(t *testing.T) {
	// default should be valid
	params := DefaultParams()
	require.NoError(t, params.Validate())

	// negative invalid
	params = DefaultParams()
	params.TaxRedirectRate = sdkmath.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	// greater than 1 invalid
	params = DefaultParams()
	params.TaxRedirectRate = sdkmath.LegacyMustNewDecFromStr("1.000000000000000001")
	require.Error(t, params.Validate())

	// exactly 0 valid
	params = DefaultParams()
	params.TaxRedirectRate = sdkmath.LegacyZeroDec()
	require.NoError(t, params.Validate())

	// exactly 1 valid
	params = DefaultParams()
	params.TaxRedirectRate = sdkmath.LegacyOneDec()
	require.NoError(t, params.Validate())
}
