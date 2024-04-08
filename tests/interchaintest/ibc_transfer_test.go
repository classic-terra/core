package interchaintest

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTerraGaiaIBCTranfer spins up a Terra Classic and Gaia network, initializes an IBC connection between them,
// and sends an ICS20 token transfer from Terra Classic -> Gaia and then back from Gaia -> Terra Classic.
func TestTerraGaiaIBCTranfer(t *testing.T) {
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
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra, gaia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	// Create relayer factory to utilize the go-relayer
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).
		Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(gaia).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra,
			Chain2:  gaia,
			Relayer: r,
			Path:    pathTerraGaia,
		})

	// Build interchain
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer and set the cleanup function.
	require.NoError(t, r.StartRelayer(ctx, eRep, pathTerraGaia))
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				panic(fmt.Errorf("an error occurred while stopping the relayer: %s", err))
			}
		},
	)

	// Create and Fund User Wallets
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", genesisWalletAmount, terra, gaia)
	terraUser := users[0]
	gaiaUser := users[1]

	terraUserAddr := terraUser.FormattedAddress()
	gaiaUserAddr := gaiaUser.FormattedAddress()

	err = testutil.WaitForBlocks(ctx, 10, terra, gaia)
	require.NoError(t, err)

	terraUserInitialBal, err := terra.GetBalance(ctx, terraUserAddr, terra.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, genesisWalletBalance, terraUserInitialBal)

	gaiaUserInitialBal, err := gaia.GetBalance(ctx, gaiaUserAddr, gaia.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, genesisWalletBalance, gaiaUserInitialBal)

	// Compose an IBC transfer and send from Terra Classic -> Gaia
	transferAmount := math.NewInt(1000)
	transfer := ibc.WalletAmount{
		Address: gaiaUserAddr,
		Denom:   terra.Config().Denom,
		Amount:  transferAmount,
	}

	// Query for the newly created channel
	terraChannels, err := r.GetChannels(ctx, eRep, terra.Config().ChainID)
	require.NoError(t, err)

	transferTx, err := terra.SendIBCTransfer(ctx, terraChannels[0].ChannelID, terraUserAddr, transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	terraHeight, err := terra.Height(ctx)
	require.NoError(t, err)

	// Poll for the ack to know the transfer was successful
	_, err = testutil.PollForAck(ctx, terra, terraHeight, terraHeight+10, transferTx.Packet)
	require.NoError(t, err)

	// Get the IBC denom for uluna on Gaia
	terraTokenDenom := transfertypes.GetPrefixedDenom(terraChannels[0].Counterparty.PortID, terraChannels[0].Counterparty.ChannelID, terra.Config().Denom)
	terraIBCDenom := transfertypes.ParseDenomTrace(terraTokenDenom).IBCDenom()

	// Assert that the funds are no longer present in user acc on Terra Classic and are in the user acc on Gaia
	terraUserUpdateBal, err := terra.GetBalance(ctx, terraUserAddr, terra.Config().Denom)
	require.NoError(t, err)
	require.True(t, terraUserUpdateBal.Equal(terraUserInitialBal.Sub(transferAmount)))

	gaiaUserUpdateBal, err := gaia.GetBalance(ctx, gaiaUserAddr, terraIBCDenom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, gaiaUserUpdateBal)

	// Compose an IBC transfer and send from Gaia -> Terra Classic
	transfer = ibc.WalletAmount{
		Address: terraUserAddr,
		Denom:   terraIBCDenom,
		Amount:  transferAmount,
	}

	transferTx, err = gaia.SendIBCTransfer(ctx, terraChannels[0].Counterparty.ChannelID, gaiaUserAddr, transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)

	// Poll for the ack to know the transfer was successful
	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight, gaiaHeight+10, transferTx.Packet)
	require.NoError(t, err)

	// Assert that the funds are now back on Terra Classic and not on Gaia
	terraUserUpdateBal, err = terra.GetBalance(ctx, terraUserAddr, terra.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, terraUserInitialBal, terraUserUpdateBal)

	gaiaUserUpdateBal, err = gaia.GetBalance(ctx, gaiaUserAddr, terraIBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(0), gaiaUserUpdateBal)
}
