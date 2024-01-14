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

	// Create relayer factory to utilize the go-relayer
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).
		Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(terra1).
		AddChain(terra2).
		AddChain(terra3).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra1,
			Chain2:  terra2,
			Relayer: r,
			Path:    path1,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra2,
			Chain2:  terra3,
			Relayer: r,
			Path:    path2,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra1,
			Chain2:  terra3,
			Relayer: r,
			Path:    path3,
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

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, terra1, terra2, terra3)
	terra1User := users[0]
	terra2User := users[1]
	terra3User := users[2]

	terra1UserAddr := terra1User.FormattedAddress()
	terra2UserAddr := terra2User.FormattedAddress()
	terra3UserAddr := terra3User.FormattedAddress()

	err = testutil.WaitForBlocks(ctx, 10, terra1, terra2, terra3)
	require.NoError(t, err)

	// rly terra1-terra2
	// Generate new path
	err = r.GeneratePath(ctx, eRep, terra1.Config().ChainID, terra2.Config().ChainID, path1)
	require.NoError(t, err)
	// Create client
	err = r.CreateClients(ctx, eRep, path1, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra2)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path1)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra2)
	require.NoError(t, err)
	// Create channel
	err = r.CreateChannel(ctx, eRep, path1, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra2)
	require.NoError(t, err)

	channelsTerra1, err := r.GetChannels(ctx, eRep, terra1.Config().ChainID)
	require.NoError(t, err)

	channelsTerra2, err := r.GetChannels(ctx, eRep, terra2.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsTerra1, 1)
	require.Len(t, channelsTerra2, 1)

	channelTerra1Terra2 := channelsTerra1[0]
	require.NotEmpty(t, channelTerra1Terra2.ChannelID)
	channelTerra2Terra1 := channelsTerra2[0]
	require.NotEmpty(t, channelTerra2Terra1.ChannelID)

	// rly terra1-terra3
	// Generate new path
	err = r.GeneratePath(ctx, eRep, terra1.Config().ChainID, terra3.Config().ChainID, path3)
	require.NoError(t, err)
	// Create clients
	err = r.CreateClients(ctx, eRep, path3, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra3)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path3)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra3)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, path3, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra1, terra3)
	require.NoError(t, err)

	channelsTerra1, err = r.GetChannels(ctx, eRep, terra1.Config().ChainID)
	require.NoError(t, err)

	channelsTerra3, err := r.GetChannels(ctx, eRep, terra3.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsTerra1, 2)
	require.Len(t, channelsTerra3, 1)

	var channelTerra1Terra3 ibc.ChannelOutput
	for _, chann := range channelsTerra1 {
		if chann.ChannelID != channelTerra1Terra2.ChannelID {
			channelTerra1Terra3 = chann
		}
	}
	require.NotEmpty(t, channelTerra1Terra3.ChannelID)

	channelTerra3Terra1 := channelsTerra3[0]
	require.NotEmpty(t, channelTerra3Terra1.ChannelID)

	// rly terra2-terra3
	// Generate new path
	err = r.GeneratePath(ctx, eRep, terra2.Config().ChainID, terra3.Config().ChainID, path2)
	require.NoError(t, err)

	// Create clients
	err = r.CreateClients(ctx, eRep, path2, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra2, terra3)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path2)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra2, terra3)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, path2, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, terra2, terra3)
	require.NoError(t, err)

	channelsTerra2, err = r.GetChannels(ctx, eRep, terra2.Config().ChainID)
	require.NoError(t, err)

	channelsTerra3, err = r.GetChannels(ctx, eRep, terra3.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsTerra2, 2)
	require.Len(t, channelsTerra3, 2)

	var channelTerra2Terra3 ibc.ChannelOutput
	for _, chann := range channelsTerra2 {
		if chann.ChannelID != channelTerra2Terra1.ChannelID {
			channelTerra2Terra3 = chann
		}
	}
	require.NotEmpty(t, channelTerra2Terra3.ChannelID)

	var channelTerra3Terra2 ibc.ChannelOutput
	for _, chann := range channelsTerra3 {
		if chann.ChannelID != channelTerra3Terra1.ChannelID {
			channelTerra3Terra2 = chann
		}
	}
	require.NotEmpty(t, channelTerra3Terra2.ChannelID)

	// Start the relayer on both paths
	err = r.StartRelayer(ctx, eRep, path1, path2, path3)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				t.Logf("an error occurred while stopping the relayer: %s", err)
			}
		},
	)

	// Send a transfer from Terra 2 -> Terra 1
	transferAmount := math.NewInt(1000)
	transfer := ibc.WalletAmount{
		Address: terra1UserAddr,
		Denom:   terra2.Config().Denom,
		Amount:  transferAmount,
	}

	transferTx, err := terra2.SendIBCTransfer(ctx, channelTerra2Terra1.ChannelID, terra2User.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	terra2Height, err := terra2.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, terra2, terra2Height, terra2Height+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, terra2)
	require.NoError(t, err)

	terra2OnTerra1TokenDenom := transfertypes.GetPrefixedDenom(channelTerra2Terra1.Counterparty.PortID, channelTerra2Terra1.Counterparty.ChannelID, terra2.Config().Denom)
	terra2OnTerra1IBCDenom := transfertypes.ParseDenomTrace(terra2OnTerra1TokenDenom).IBCDenom()

	terra2OnTerra3TokenDenom := transfertypes.GetPrefixedDenom(channelTerra2Terra3.Counterparty.PortID, channelTerra2Terra3.Counterparty.ChannelID, terra2.Config().Denom)
	terra2OnTerra3IBCDenom := transfertypes.ParseDenomTrace(terra2OnTerra3TokenDenom).IBCDenom()

	terra1UserUpdateBal, err := terra1.GetBalance(ctx, terra1UserAddr, terra2OnTerra1IBCDenom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, terra1UserUpdateBal)

	// Send a transfer with pfm from Terra 1 -> Terra 3
	// The PacketForwardMiddleware will forward the packet Terra 2 -> Terra 3
	metadata := &helpers.PacketMetadata{
		Forward: &helpers.ForwardMetadata{
			Receiver: terra3UserAddr,
			Channel:  channelTerra2Terra3.ChannelID,
			Port:     channelTerra2Terra3.PortID,
		},
	}
	transfer = ibc.WalletAmount{
		Address: terra2UserAddr,
		Denom:   terra2OnTerra1IBCDenom,
		Amount:  transferAmount,
	}

	memo, err := json.Marshal(metadata)
	require.NoError(t, err)

	transferTx, err = terra1.SendIBCTransfer(ctx, channelTerra1Terra2.ChannelID, terra1User.KeyName(), transfer, ibc.TransferOptions{Memo: string(memo)})
	require.NoError(t, err)

	terraHeight, err := terra1.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, terra1, terraHeight, terraHeight+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, terra1)
	require.NoError(t, err)

	// terra2 user send 1000uatom at the begining so the balance should be 1000uatom less
	terra2UserUpdateBal, err := terra2.GetBalance(ctx, terra2UserAddr, terra2.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, terra2UserUpdateBal, genesisWalletAmount.Sub(transferAmount))

	terra1UserUpdateBal, err = terra1.GetBalance(ctx, terra1UserAddr, terra2OnTerra1IBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), terra1UserUpdateBal)

	terra3UserUpdateBal, err := terra3.GetBalance(ctx, terra3UserAddr, terra2OnTerra3IBCDenom)
	require.NoError(t, err)
	require.Equal(t, terra3UserUpdateBal, transferAmount)

	// Check Escrow Balance
	escrowAccount := sdk.MustBech32ifyAddressBytes(terra2.Config().Bech32Prefix, transfertypes.GetEscrowAddress(channelTerra2Terra3.PortID, channelTerra2Terra3.ChannelID))
	escrowBalance, err := terra2.GetBalance(ctx, escrowAccount, terra2.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, escrowBalance)
}
