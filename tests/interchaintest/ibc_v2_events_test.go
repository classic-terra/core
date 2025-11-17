package interchaintest

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Local IBC event types (avoid depending on other test files)
type IBCEventTx struct {
    Data   []byte
    Events []IBCEvent
}

type IBCEvent struct {
    Type       string
    Attributes []IBCEventAttribute
}

type IBCEventAttribute struct {
    Key   string
    Value string
}

// scanBlockEvents converts txs for a given height from a node into the local Tx representation used in oracle_test.go
func scanBlockEvents(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, height int64) []IBCEventTx {
	n := chain.Validators[0]
	txs, err := n.FindTxs(ctx, height)
	require.NoError(t, err)
	convertedTxs := make([]IBCEventTx, len(txs))
	for i, tx := range txs {
		convertedEvents := make([]IBCEvent, len(tx.Events))
		for j, event := range tx.Events {
			convertedEvents[j] = IBCEvent{
				Type:       event.Type,
				Attributes: make([]IBCEventAttribute, len(event.Attributes)),
			}
			for k, attr := range event.Attributes {
				convertedEvents[j].Attributes[k] = IBCEventAttribute{Key: attr.Key, Value: attr.Value}
			}
		}
		convertedTxs[i] = IBCEventTx{Data: tx.Data, Events: convertedEvents}
	}
	return convertedTxs
}

func containsEvent(txs []IBCEventTx, eventType string) bool {
	for _, tx := range txs {
		for _, ev := range tx.Events {
			if strings.EqualFold(ev.Type, eventType) { // be lenient on case
				return true
			}
		}
	}
	return false
}

func containsAnyEventInWindow(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, startHeight, endHeight int64, eventType string) bool {
	for h := startHeight; h <= endHeight; h++ {
		txs := scanBlockEvents(t, ctx, chain, h)
		if containsEvent(txs, eventType) {
			return true
		}
	}
	return false
}

// TestIBCv2HandshakeEvents validates that channel and connection handshake events are emitted during path creation.
func TestIBCv2HandshakeEvents(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	// Do not run in parallel; consumes resources shared with other tests

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

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	terra, gaia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)

	const path = "ibcv2-handshake"

	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(gaia).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: terra, Chain2: gaia, Relayer: r, Path: path})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	// After build, handshake should have completed. Capture current heights and scan a recent window.
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, terra, gaia))
	terraH, err := terra.Height(ctx)
	require.NoError(t, err)
	gaiaH, err := gaia.Height(ctx)
	require.NoError(t, err)

	startTerra := terraH - 50
	if startTerra < 1 { startTerra = 1 }
	startGaia := gaiaH - 50
	if startGaia < 1 { startGaia = 1 }

	// Expected IBC v2 handshake events
	handshakeEvents := []string{
		"connection_open_init",
		"connection_open_try",
		"connection_open_ack",
		"connection_open_confirm",
		"channel_open_init",
		"channel_open_try",
		"channel_open_ack",
		"channel_open_confirm",
	}

	for _, ev := range handshakeEvents {
		found := containsAnyEventInWindow(t, ctx, terra, startTerra, terraH, ev) ||
			containsAnyEventInWindow(t, ctx, gaia, startGaia, gaiaH, ev)
		require.Truef(t, found, "expected to find event %s in recent handshake window", ev)
	}
}

// TestIBCv2TransferEvents validates send, recv, ack events for a standard ICS20 transfer
func TestIBCv2TransferEvents(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

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

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	terra, gaia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)
	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(gaia).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: terra, Chain2: gaia, Relayer: r, Path: pathTerraGaia})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	require.NoError(t, r.StartRelayer(ctx, eRep, pathTerraGaia))
	t.Cleanup(func() { _ = r.StopRelayer(ctx, eRep) })

	// Fund users
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(genesisWalletAmount), terra, gaia)
	terraUser, gaiaUser := users[0], users[1]
	require.NoError(t, testutil.WaitForBlocks(ctx, 5, terra, gaia))

	channel, err := ibc.GetTransferChannel(ctx, r, eRep, terra.Config().ChainID, gaia.Config().ChainID)
	require.NoError(t, err)

	transfer := ibc.WalletAmount{Address: gaiaUser.FormattedAddress(), Denom: terra.Config().Denom, Amount: math.NewInt(1000)}
	transferTx, err := terra.SendIBCTransfer(ctx, channel.ChannelID, terraUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	terraH, err := terra.Height(ctx)
	require.NoError(t, err)

	_, err = testutil.PollForAck(ctx, terra, terraH-5, terraH+25, transferTx.Packet)
	require.NoError(t, err)
	// give relayer time to relay ack
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, terra, gaia))

	// Scan recent window for events
	terraH2, _ := terra.Height(ctx)
	gaiaH2, _ := gaia.Height(ctx)
	startTerra := terraH - 10
	if startTerra < 1 { startTerra = 1 }
	startGaia := gaiaH2 - 30
	if startGaia < 1 { startGaia = 1 }

	require.True(t, containsAnyEventInWindow(t, ctx, terra, startTerra, terraH2, "send_packet"))
	// recv and write_ack occur on destination
	require.True(t, containsAnyEventInWindow(t, ctx, gaia, startGaia, gaiaH2, "recv_packet"))
	require.True(t, containsAnyEventInWindow(t, ctx, gaia, startGaia, gaiaH2, "write_acknowledgement"))
	// acknowledge_packet occurs on source when ack is relayed back
	require.True(t, containsAnyEventInWindow(t, ctx, terra, startTerra, terraH2, "acknowledge_packet"))
}
