package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
)

func TestGenesisValidation(t *testing.T) {
	genState := DefaultGenesisState()
	require.NoError(t, ValidateGenesis(genState))

	// Error - tax_rate range error
	genState.TaxRate = sdkmath.LegacyNewDec(-1)
	require.Error(t, ValidateGenesis(genState))

	// Valid
	genState.TaxRate = sdkmath.LegacyNewDecWithPrec(1, 2)
	require.NoError(t, ValidateGenesis(genState))

	// Error - reward_weight range error
	genState.RewardWeight = sdkmath.LegacyNewDec(-1)
	require.Error(t, ValidateGenesis(genState))

	// Valid
	genState.RewardWeight = sdkmath.LegacyNewDecWithPrec(5, 2)
	require.NoError(t, ValidateGenesis(genState))

	dummyDec := sdkmath.LegacyNewDec(10)
	dummyInt := sdkmath.NewInt(10)

	genState.EpochStates = []EpochState{
		{
			Epoch:             0,
			TaxReward:         dummyDec,
			SeigniorageReward: dummyDec,
			TotalStakedLuna:   dummyInt,
		},
		{
			Epoch:             1,
			TaxReward:         dummyDec,
			SeigniorageReward: dummyDec,
			TotalStakedLuna:   dummyInt,
		},
	}

	// Valid
	require.NoError(t, ValidateGenesis(genState))
}
