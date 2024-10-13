package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKeeper(t *testing.T) {
	input := CreateTestInput(t)
	require.NotNil(t, input.AccountKeeper)
	require.NotNil(t, input.BankKeeper)
	require.NotNil(t, input.OracleKeeper)
	require.NotNil(t, input.GovKeeper)
}
