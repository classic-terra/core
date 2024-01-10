package interchaintest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v2/test/interchaintest/helpers"
	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestIBCHooks ensures the ibc-hooks middleware from osmosis works.
func TestTerraIBCHooks(t *testing.T) {
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
	config2.Name = "core-counterparty"
	config2.ChainID = "core-counterparty-1"

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
	})

	const (
		path = "ibc-path"
	)

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra, terra2 := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	// Create relayer factory to utilize the go-relayer
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).
		Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(terra2).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  terra,
			Chain2:  terra2,
			Relayer: r,
			Path:    path,
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
	require.NoError(t, r.StartRelayer(ctx, eRep, path))
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				panic(fmt.Errorf("an error occurred while stopping the relayer: %s", err))
			}
		},
	)

	// Create and Fund User Wallets
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", genesisWalletAmount, terra, terra2)
	terraUser, terra2User := users[0], users[1]

	terraUserAddr := terraUser.FormattedAddress()

	// Wait a few blocks for relayer to start and for user accounts to be created
	err = testutil.WaitForBlocks(ctx, 5, terra, terra2)
	require.NoError(t, err)

	channel, err := ibc.GetTransferChannel(ctx, r, eRep, terra.Config().ChainID, terra2.Config().ChainID)
	require.NoError(t, err)

	_, contractAddr := helpers.SetupContract(t, ctx, terra2, terra2User.KeyName(), "bytecode/counter.wasm", `{"count":0}`)

	transfer := ibc.WalletAmount{
		Address: contractAddr,
		Denom:   terra.Config().Denom,
		Amount:  math.OneInt(),
	}

	memo := ibc.TransferOptions{
		Memo: fmt.Sprintf(`{"wasm":{"contract":"%s","msg":%s}}`, contractAddr, `{"increment":{}}`),
	}

	// Initial transfer. Account is created by the wasm execute is not so we must do this twice to properly set up
	transferTx, err := terra.SendIBCTransfer(ctx, channel.ChannelID, terraUser.KeyName(), transfer, memo)
	require.NoError(t, err)
	terraHeight, err := terra.Height(ctx)
	require.NoError(t, err)

	// TODO: Remove when the relayer is fixed
	r.Flush(ctx, eRep, path, channel.ChannelID)
	_, err = testutil.PollForAck(ctx, terra, terraHeight-5, terraHeight+25, transferTx.Packet)
	require.NoError(t, err)

	// Second time, this will make the counter == 1 since the account is now created.
	transferTx, err = terra.SendIBCTransfer(ctx, channel.ChannelID, terraUser.KeyName(), transfer, memo)
	require.NoError(t, err)
	terraHeight, err = terra.Height(ctx)
	require.NoError(t, err)

	// TODO: Remove when the relayer is fixed
	r.Flush(ctx, eRep, path, channel.ChannelID)
	_, err = testutil.PollForAck(ctx, terra, terraHeight-5, terraHeight+25, transferTx.Packet)
	require.NoError(t, err)

	// Get the address on the other chain's side
	addr := helpers.GetIBCHooksUserAddress(t, ctx, terra, channel.ChannelID, terraUserAddr)
	require.NotEmpty(t, addr)

	// Get funds on the receiving chain
	funds := helpers.GetIBCHookTotalFunds(t, ctx, terra2, contractAddr, addr)
	require.Equal(t, int(1), len(funds.Data.TotalFunds))

	var ibcDenom string
	for _, coin := range funds.Data.TotalFunds {
		if strings.HasPrefix(coin.Denom, "ibc/") {
			ibcDenom = coin.Denom
			break
		}
	}
	require.NotEmpty(t, ibcDenom)

	// ensure the count also increased to 1 as expected.
	count := helpers.GetIBCHookCount(t, ctx, terra2, contractAddr, addr)
	require.Equal(t, int64(1), count.Data.Count)
}
