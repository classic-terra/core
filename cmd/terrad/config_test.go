package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOverrideConfigCacheSize(t *testing.T) {
	_, cfg := initAppConfig()
	terraCfg, ok := cfg.(TerraAppConfig)
	require.Equal(t, ok, true)
	require.Equal(t, terraCfg.Config.IAVLCacheSize, uint64(DefaultIAVLCacheSize))
	require.Equal(t, terraCfg.Config.IAVLDisableFastNode, IavlDisablefastNodeDefault)
}
