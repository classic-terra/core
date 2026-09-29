package cli

import (
	"testing"

	"github.com/classic-terra/core/v4/x/cron/types"
	"github.com/stretchr/testify/require"
)

func TestGetTxCmd(t *testing.T) {
	cmd := GetTxCmd()
	require.Equal(t, types.ModuleName, cmd.Use)
	require.Len(t, cmd.Commands(), 3)
}
