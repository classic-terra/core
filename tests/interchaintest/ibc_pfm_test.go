package interchaintest

import (
	"context"
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/classic-terra/core/v2/test/interchaintest/helpers"
)

// TestTerraGaiaOsmoPFM setup up a Terra Classic, Osmosis and Gaia network, initializes an IBC connection between them,
// and sends an ICS20 token transfer from Gaia -> Terra Classic -> Osmosis to make sure that the IBC denom not being hashed again.
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

	// Create relayer factory to utilize the go-relayer
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).
		Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(gaia).
		AddChain(osmo).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra,
			Chain2:  gaia,
			Relayer: r,
			Path:    pathTerraGaia,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra,
			Chain2:  osmo,
			Relayer: r,
			Path:    pathTerraOsmo,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  gaia,
			Chain2:  osmo,
			Relayer: r,
			Path:    pathGaiaOsmo,
		})

	// Build interchain
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation:  true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, terra, gaia, osmo)
	terraUser := users[0]
	gaiaUser := users[1]
	osmoUser := users[2]

	terraUserAddr := terraUser.FormattedAddress()
	gaiaUserAddr := gaiaUser.FormattedAddress()
	osmoUserAddr := osmoUser.FormattedAddress()

	err = testutil.WaitForBlocks(ctx, 10, terra, gaia, osmo)
	require.NoError(t, err)

	// rly terra-gaia
	// Generate new path
	err = r.GeneratePath(ctx, eRep, terra.Config().ChainID, gaia.Config().ChainID, pathTerraGaia)
	require.NoError(t, err)
	// Create client
	err = r.CreateClients(ctx, eRep, pathTerraGaia, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, gaia)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, pathTerraGaia)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, gaia)
	require.NoError(t, err)
	// Create channel
	err = r.CreateChannel(ctx, eRep, pathTerraGaia, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, gaia)
	require.NoError(t, err)

	channelsTerra, err := r.GetChannels(ctx, eRep, terra.Config().ChainID)
	require.NoError(t, err)

	channelsGaia, err := r.GetChannels(ctx, eRep, gaia.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsTerra, 1)
	require.Len(t, channelsGaia, 1)

	channelTerraGaia := channelsTerra[0]
	require.NotEmpty(t, channelTerraGaia.ChannelID)
	channelGaiaTerra := channelsGaia[0]
	require.NotEmpty(t, channelGaiaTerra.ChannelID)

	// rly terra-osmo
	// Generate new path
	err = r.GeneratePath(ctx, eRep, terra.Config().ChainID, osmo.Config().ChainID, pathTerraOsmo)
	require.NoError(t, err)
	// Create clients
	err = r.CreateClients(ctx, eRep, pathTerraOsmo, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, osmo)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, pathTerraOsmo)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, osmo)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, pathTerraOsmo, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra, osmo)
	require.NoError(t, err)

	channelsTerra, err = r.GetChannels(ctx, eRep, terra.Config().ChainID)
	require.NoError(t, err)

	channelsOsmo, err := r.GetChannels(ctx, eRep, osmo.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsTerra, 2)
	require.Len(t, channelsOsmo, 1)

	var channelTerraOsmo ibc.ChannelOutput
	for _, chann := range channelsTerra {
		if chann.ChannelID != channelTerraGaia.ChannelID {
			channelTerraOsmo = chann
		}
	}
	require.NotEmpty(t, channelTerraOsmo.ChannelID)

	channelOsmoTerra := channelsOsmo[0]
	require.NotEmpty(t, channelOsmoTerra.ChannelID)

	// rly gaia-osmo
	// Generate new path
	err = r.GeneratePath(ctx, eRep, gaia.Config().ChainID, osmo.Config().ChainID, pathGaiaOsmo)
	require.NoError(t, err)

	// Create clients
	err = r.CreateClients(ctx, eRep, pathGaiaOsmo, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, gaia, osmo)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, pathGaiaOsmo)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, gaia, osmo)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, pathGaiaOsmo, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, gaia, osmo)
	require.NoError(t, err)

	channelsGaia, err = r.GetChannels(ctx, eRep, gaia.Config().ChainID)
	require.NoError(t, err)

	channelsOsmo, err = r.GetChannels(ctx, eRep, osmo.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsGaia, 2)
	require.Len(t, channelsOsmo, 2)

	var channelGaiaOsmo ibc.ChannelOutput
	for _, chann := range channelsGaia {
		if chann.ChannelID != channelGaiaTerra.ChannelID {
			channelGaiaOsmo = chann
		}
	}
	require.NotEmpty(t, channelGaiaOsmo.ChannelID)

	var channelOsmoGaia ibc.ChannelOutput
	for _, chann := range channelsOsmo {
		if chann.ChannelID != channelOsmoTerra.ChannelID {
			channelOsmoGaia = chann
		}
	}
	require.NotEmpty(t, channelOsmoGaia.ChannelID)

	// Start the relayer on both paths
	err = r.StartRelayer(ctx, eRep, pathGaiaOsmo, pathTerraGaia, pathTerraOsmo)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				t.Logf("an error occurred while stopping the relayer: %s", err)
			}
		},
	)

	// Send a transfer from Gaia -> Terra Classic
	transferAmount := math.NewInt(1000)
	transfer := ibc.WalletAmount{
		Address: terraUserAddr,
		Denom:   gaia.Config().Denom,
		Amount:  transferAmount,
	}

	transferTx, err := gaia.SendIBCTransfer(ctx, channelGaiaTerra.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight, gaiaHeight+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, gaia)
	require.NoError(t, err)

	gaiaOnTerraTokenDenom := transfertypes.GetPrefixedDenom(channelGaiaTerra.Counterparty.PortID, channelGaiaTerra.Counterparty.ChannelID, gaia.Config().Denom)
	gaiaOnTerraIBCDenom := transfertypes.ParseDenomTrace(gaiaOnTerraTokenDenom).IBCDenom()

	gaiaOnOsmoTokenDenom := transfertypes.GetPrefixedDenom(channelGaiaOsmo.Counterparty.PortID, channelGaiaOsmo.Counterparty.ChannelID, gaia.Config().Denom)
	gaiaOnOsmoIBCDenom := transfertypes.ParseDenomTrace(gaiaOnOsmoTokenDenom).IBCDenom()

	terraUserUpdateBal, err := terra.GetBalance(ctx, terraUserAddr, gaiaOnTerraIBCDenom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, terraUserUpdateBal)

	// Send a transfer with pfm from Terra Classic -> Osmosis
	// The PacketForwardMiddleware will forward the packet Gaia -> Osmosis
	metadata := &helpers.PacketMetadata{
		Forward: &helpers.ForwardMetadata{
			Receiver: osmoUserAddr,
			Channel:  channelGaiaOsmo.ChannelID,
			Port:     channelGaiaOsmo.PortID,
		},
	}
	transfer = ibc.WalletAmount{
		Address: gaiaUserAddr,
		Denom:   gaiaOnTerraIBCDenom,
		Amount:  transferAmount,
	}

	memo, err := json.Marshal(metadata)
	require.NoError(t, err)

	transferTx, err = terra.SendIBCTransfer(ctx, channelTerraGaia.ChannelID, terraUser.KeyName(), transfer, ibc.TransferOptions{Memo: string(memo)})
	require.NoError(t, err)

	terraHeight, err := terra.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, terra, terraHeight, terraHeight+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, terra)
	require.NoError(t, err)

	// Gaia user send 1000uatom at the begining so the balance should be 1000uatom less
	gaiaUserUpdateBal, err := gaia.GetBalance(ctx, gaiaUserAddr, gaia.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, gaiaUserUpdateBal, genesisWalletAmount.Sub(transferAmount))

	terraUserUpdateBal, err = terra.GetBalance(ctx, terraUserAddr, gaiaOnTerraIBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), terraUserUpdateBal)

	osmoUserUpdateBal, err := osmo.GetBalance(ctx, osmoUserAddr, gaiaOnOsmoIBCDenom)
	require.NoError(t, err)
	require.Equal(t, osmoUserUpdateBal, transferAmount)

	// Check Escrow Balance
	escrowAccount := sdk.MustBech32ifyAddressBytes(gaia.Config().Bech32Prefix, transfertypes.GetEscrowAddress(channelGaiaOsmo.PortID, channelGaiaOsmo.ChannelID))
	escrowBalance, err := gaia.GetBalance(ctx, escrowAccount, gaia.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, escrowBalance)
}
