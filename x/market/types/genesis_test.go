package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
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
