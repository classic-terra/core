package interchaintest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"

	"github.com/classic-terra/core/v2/test/interchaintest/helpers"
)

// TestTerraPFM setup up 3 Terra Classic networks, initializes an IBC connection between them,
// and sends an ICS20 token transfer among them to make sure that the IBC denom not being hashed again.
func TestTerraPFM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Create chain factory with Terra Classic
	numVals := 3
	numFullNodes := 3

	client, network := interchaintest.DockerSetup(t)

	ctx := context.Background()

	config1, err := createConfig()
	require.NoError(t, err)
	config2 := config1.Clone()
	config2.ChainID = "core-2"
	config3 := config1.Clone()
	config3.ChainID = "core-3"

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   config1,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "terra",
			ChainConfig:   config2,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "terra",
			ChainConfig:   config3,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	const (
		path1 = "ibc-path-1"
		path2 = "ibc-path-2"
		path3 = "ibc-path-3"
	)

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra1, terra2, terra3 := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)

	// Start the test flow
	helpers.PFMTestFlow(t, ctx, terra1, terra2, terra3, client, network, path1, path2, path3, genesisWalletAmount)
}
