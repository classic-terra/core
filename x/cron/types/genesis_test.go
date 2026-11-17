package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesisValidate(t *testing.T) {
	require.NoError(t, DefaultGenesisState().Validate())

	err := GenesisState{
		Params: Params{Limit: 0},
	}.Validate()
	require.Error(t, err)
}
