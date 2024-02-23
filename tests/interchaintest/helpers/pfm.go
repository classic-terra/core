package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/docker/docker/client"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
)

func PFMTestFlow(t *testing.T, ctx context.Context, chain1, chain2, chain3 *cosmos.CosmosChain, client *client.Client, network string, path1, path2, path3 string, genesisWalletAmount math.Int) {
	// Create relayer factory to utilize the go-relayer
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).
		Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(chain1).
		AddChain(chain2).
		AddChain(chain3).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain1,
			Chain2:  chain2,
			Relayer: r,
			Path:    path1,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain2,
			Chain2:  chain3,
			Relayer: r,
			Path:    path2,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain1,
			Chain2:  chain3,
			Relayer: r,
			Path:    path3,
		})

	// Build interchain
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
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

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, chain1, chain2, chain3)
	chain1User := users[0]
	chain2User := users[1]
	chain3User := users[2]

	chain1UserAddr := chain1User.FormattedAddress()
	chain2UserAddr := chain2User.FormattedAddress()
	chain3UserAddr := chain3User.FormattedAddress()

	err = testutil.WaitForBlocks(ctx, 10, chain1, chain2, chain3)
	require.NoError(t, err)

	// rly chain1-chain2
	// Generate new path
	err = r.GeneratePath(ctx, eRep, chain1.Config().ChainID, chain2.Config().ChainID, path1)
	require.NoError(t, err)
	// Create client
	err = r.CreateClients(ctx, eRep, path1, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain2)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path1)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain2)
	require.NoError(t, err)
	// Create channel
	err = r.CreateChannel(ctx, eRep, path1, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain2)
	require.NoError(t, err)

	channelsChain1, err := r.GetChannels(ctx, eRep, chain1.Config().ChainID)
	require.NoError(t, err)

	channelsChain2, err := r.GetChannels(ctx, eRep, chain2.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsChain1, 1)
	require.Len(t, channelsChain2, 1)

	channelChain1Chain2 := channelsChain1[0]
	require.NotEmpty(t, channelChain1Chain2.ChannelID)
	channelChain2Chain1 := channelsChain2[0]
	require.NotEmpty(t, channelChain2Chain1.ChannelID)

	// rly chain1-chain3
	// Generate new path
	err = r.GeneratePath(ctx, eRep, chain1.Config().ChainID, chain3.Config().ChainID, path3)
	require.NoError(t, err)
	// Create clients
	err = r.CreateClients(ctx, eRep, path3, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain3)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path3)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain3)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, path3, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain1, chain3)
	require.NoError(t, err)

	channelsChain1, err = r.GetChannels(ctx, eRep, chain1.Config().ChainID)
	require.NoError(t, err)

	channelsChain3, err := r.GetChannels(ctx, eRep, chain3.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsChain1, 2)
	require.Len(t, channelsChain3, 1)

	var channelChain1Chain3 ibc.ChannelOutput
	for _, chann := range channelsChain1 {
		if chann.ChannelID != channelChain1Chain2.ChannelID {
			channelChain1Chain3 = chann
		}
	}
	require.NotEmpty(t, channelChain1Chain3.ChannelID)

	channelChain3Chain1 := channelsChain3[0]
	require.NotEmpty(t, channelChain3Chain1.ChannelID)

	// rly chain2-chain3
	// Generate new path
	err = r.GeneratePath(ctx, eRep, chain2.Config().ChainID, chain3.Config().ChainID, path2)
	require.NoError(t, err)

	// Create clients
	err = r.CreateClients(ctx, eRep, path2, ibc.DefaultClientOpts())
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain2, chain3)
	require.NoError(t, err)

	// Create connection
	err = r.CreateConnections(ctx, eRep, path2)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain2, chain3)
	require.NoError(t, err)

	// Create channel
	err = r.CreateChannel(ctx, eRep, path2, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	})
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, chain2, chain3)
	require.NoError(t, err)

	channelsChain2, err = r.GetChannels(ctx, eRep, chain2.Config().ChainID)
	require.NoError(t, err)

	channelsChain3, err = r.GetChannels(ctx, eRep, chain3.Config().ChainID)
	require.NoError(t, err)

	require.Len(t, channelsChain2, 2)
	require.Len(t, channelsChain3, 2)

	var channelChain2Chain3 ibc.ChannelOutput
	for _, chann := range channelsChain2 {
		if chann.ChannelID != channelChain2Chain1.ChannelID {
			channelChain2Chain3 = chann
		}
	}
	require.NotEmpty(t, channelChain2Chain3.ChannelID)

	var channelChain3Chain2 ibc.ChannelOutput
	for _, chann := range channelsChain3 {
		if chann.ChannelID != channelChain3Chain1.ChannelID {
			channelChain3Chain2 = chann
		}
	}
	require.NotEmpty(t, channelChain3Chain2.ChannelID)

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

	// Send a transfer from Chain 2 -> Chain 1
	transferAmount := math.NewInt(1000)
	transfer := ibc.WalletAmount{
		Address: chain1UserAddr,
		Denom:   chain2.Config().Denom,
		Amount:  transferAmount,
	}

	transferTx, err := chain2.SendIBCTransfer(ctx, channelChain2Chain1.ChannelID, chain2User.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	chain2Height, err := chain2.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, chain2, chain2Height, chain2Height+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, chain2)
	require.NoError(t, err)

	chain2Onchain1TokenDenom := transfertypes.GetPrefixedDenom(channelChain2Chain1.Counterparty.PortID, channelChain2Chain1.Counterparty.ChannelID, chain2.Config().Denom)
	chain2Onchain1IBCDenom := transfertypes.ParseDenomTrace(chain2Onchain1TokenDenom).IBCDenom()

	chain2Onchain3TokenDenom := transfertypes.GetPrefixedDenom(channelChain2Chain3.Counterparty.PortID, channelChain2Chain3.Counterparty.ChannelID, chain2.Config().Denom)
	chain2Onchain3IBCDenom := transfertypes.ParseDenomTrace(chain2Onchain3TokenDenom).IBCDenom()

	chain1UserUpdateBal, err := chain1.GetBalance(ctx, chain1UserAddr, chain2Onchain1IBCDenom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, chain1UserUpdateBal)

	// Send a transfer with pfm from Chain 1 -> Chain 3
	// The PacketForwardMiddleware will forward the packet Chain 2 -> Chain 3
	metadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: chain3UserAddr,
			Channel:  channelChain2Chain3.ChannelID,
			Port:     channelChain2Chain3.PortID,
		},
	}
	transfer = ibc.WalletAmount{
		Address: chain2UserAddr,
		Denom:   chain2Onchain1IBCDenom,
		Amount:  transferAmount,
	}

	memo, err := json.Marshal(metadata)
	require.NoError(t, err)

	transferTx, err = chain1.SendIBCTransfer(ctx, channelChain1Chain2.ChannelID, chain1User.KeyName(), transfer, ibc.TransferOptions{Memo: string(memo)})
	require.NoError(t, err)

	chain1Height, err := chain1.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, chain1, chain1Height, chain1Height+30, transferTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 5, chain1)
	require.NoError(t, err)

	// chain2 user send 1000uatom at the begining so the balance should be 1000uatom less
	chain2UserUpdateBal, err := chain2.GetBalance(ctx, chain2UserAddr, chain2.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, chain2UserUpdateBal, genesisWalletAmount.Sub(transferAmount))

	chain1UserUpdateBal, err = chain1.GetBalance(ctx, chain1UserAddr, chain2Onchain1IBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), chain1UserUpdateBal)

	chain3UserUpdateBal, err := chain3.GetBalance(ctx, chain3UserAddr, chain2Onchain3IBCDenom)
	require.NoError(t, err)
	require.Equal(t, chain3UserUpdateBal, transferAmount)

	// Check Escrow Balance
	escrowAccount := sdk.MustBech32ifyAddressBytes(chain2.Config().Bech32Prefix, transfertypes.GetEscrowAddress(channelChain2Chain3.PortID, channelChain2Chain3.ChannelID))
	escrowBalance, err := chain2.GetBalance(ctx, escrowAccount, chain2.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, transferAmount, escrowBalance)

}
