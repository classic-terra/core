package interchaintest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"

	"github.com/classic-terra/core/v2/test/interchaintest/helpers"
)

// TestTerraGaiaOsmoPFM setup up a Terra Classic, Osmosis and Gaia network, initializes an IBC connection between them,
// and sends an ICS20 token transfer from Terra Classic -> Gaia -> Osmosis to make sure that the IBC denom not being hashed again.
func TestTerraGaiaOsmoPFM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Create chain factory with Terra Classic
	numVals := 3
	numFullNodes := 3

	client, network := interchaintest.DockerSetup(t)

	ctx := context.Background()

	config, err := createConfig()
	require.NoError(t, err)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "gaia",
			Version:       "v12.0.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
			ChainConfig: ibc.ChainConfig{
				GasPrices: "0.0uatom",
			},
		},
		{
			Name:          "osmosis",
			Version:       "v18.0.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
			ChainConfig: ibc.ChainConfig{
				GasPrices: "0.005uosmo",
			},
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra, gaia, osmo := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)

	// Start the test flow
	helpers.PFMTestFlow(t, ctx, terra, gaia, osmo, client, network, pathTerraGaia, pathTerraOsmo, pathGaiaOsmo, genesisWalletAmount)
}
