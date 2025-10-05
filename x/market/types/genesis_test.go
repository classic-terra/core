package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
)

func TestGenesisValidation(t *testing.T) {
	genState := DefaultGenesisState()
	require.NoError(t, ValidateGenesis(genState))

	genState.Params.BasePool = sdkmath.LegacyNewDec(-1)
	require.Error(t, ValidateGenesis(genState))

	genState = DefaultGenesisState()
	genState.Params.PoolRecoveryPeriod = 0
	require.Error(t, ValidateGenesis(genState))

	genState = DefaultGenesisState()
	genState.Params.MinStabilitySpread = sdkmath.LegacyNewDec(-1)
	require.Error(t, ValidateGenesis(genState))
}
