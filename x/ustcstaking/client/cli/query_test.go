package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseUint64ArgumentRejectsMalformedInput(t *testing.T) {
	for _, value := range []string{"7junk", "1 2", "-1"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseUint64Argument(value, "position id")
			require.Error(t, err)
		})
	}

	value, err := parseUint64Argument("7", "position id")
	require.NoError(t, err)
	require.Equal(t, uint64(7), value)
}

func TestPositionsQueryUsesDocumentedNameAndLegacyAlias(t *testing.T) {
	cmd := positionsByOwnerQueryCmd()
	require.Equal(t, "positions [owner]", cmd.Use)
	require.Contains(t, cmd.Aliases, "positions-by-owner")
}

func TestValidationFailureProducesNonzeroCommandError(t *testing.T) {
	err := validationFailure("reward pool is below protected reserves")
	require.EqualError(t, err, "USTC staking validation failed: reward pool is below protected reserves")
}
