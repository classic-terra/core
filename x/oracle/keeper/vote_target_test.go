package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestKeeper_GetVoteTargets(t *testing.T) {
	input := CreateTestInput(t)

	input.OracleKeeper.ClearTobinTaxes(input.Ctx)

	expectedTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range expectedTargets {
		input.OracleKeeper.SetTobinTax(input.Ctx, target, sdkmath.LegacyOneDec())
	}

	targets := input.OracleKeeper.GetVoteTargets(input.Ctx)
	require.Equal(t, expectedTargets, targets)
}

func TestKeeper_IsVoteTarget(t *testing.T) {
	input := CreateTestInput(t)

	input.OracleKeeper.ClearTobinTaxes(input.Ctx)

	validTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range validTargets {
		input.OracleKeeper.SetTobinTax(input.Ctx, target, sdkmath.LegacyOneDec())
		require.True(t, input.OracleKeeper.IsVoteTarget(input.Ctx, target))
	}
}
