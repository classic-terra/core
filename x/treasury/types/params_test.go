package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
