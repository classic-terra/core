package cli

import (
	"testing"

	"github.com/classic-terra/core/v4/x/cron/types"
	"github.com/stretchr/testify/require"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := GetQueryCmd()
	require.Equal(t, types.ModuleName, cmd.Use)
	require.Len(t, cmd.Commands(), 3)
}
