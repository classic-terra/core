package interchaintest

import (
	"context"
	"fmt"
	"time"
	"testing"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestIBCv2TimeoutOnClose simulates a packet timing out due to channel close (no explicit short timeout required).
// Flow: start relayer to create path -> stop relayer -> send packet -> close channel -> start relayer -> expect timeout_packet on source and refund.
func TestIBCv2TimeoutOnClose(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	numVals := 3
	numFullNodes := 3

	client, network := interchaintest.DockerSetup(t)
	ctx := context.Background()

	cfg, err := createConfig()
	require.NoError(t, err)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   cfg,
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
	terra := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)
	const path = "terra-gaia-timeout-close"
	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(gaia).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: terra, Chain2: gaia, Relayer: r, Path: path})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	// Start and stop relayer so channel/connection are created but packets won't relay
	require.NoError(t, r.StartRelayer(ctx, eRep, path))
	require.NoError(t, testutil.WaitForBlocks(ctx, 5, terra, gaia))
	require.NoError(t, r.StopRelayer(ctx, eRep))

	// Fund users and get channel
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(genesisWalletAmount), terra, gaia)
	terraUser := users[0]
	gaiaUser := users[1]
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, terra, gaia))

	ch, err := ibc.GetTransferChannel(ctx, r, eRep, terra.Config().ChainID, gaia.Config().ChainID)
	require.NoError(t, err)

	// Compute ibc denom on Gaia for assertion
	pref := transfertypes.GetPrefixedDenom(ch.Counterparty.PortID, ch.Counterparty.ChannelID, terra.Config().Denom)
	dt := transfertypes.ParseDenomTrace(pref)
	gaiaIBCDenom := dt.IBCDenom()

	terraBefore, err := terra.GetBalance(ctx, terraUser.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)
	gaiaBefore, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), gaiaIBCDenom)
	require.NoError(t, err)

	amount := math.NewInt(7777)
	amountStr := fmt.Sprintf("%d%s", amount.Int64(), terra.Config().Denom)
	// Use absolute timestamp timeout in nanoseconds (now + 6s). Height-based relative timeouts are not supported by this CLI.
	timeoutNs := time.Now().Add(6 * time.Second).UnixNano()
	// terrad tx ibc-transfer transfer transfer <channel-id> <receiver> <amountDenom> --packet-timeout-timestamp <unix-ns> --absolute-timeouts
	_, err = terra.Validators[0].ExecTx(ctx, terraUser.KeyName(), "ibc-transfer", "transfer", "transfer", ch.ChannelID, gaiaUser.FormattedAddress(), amountStr, "--packet-timeout-timestamp", fmt.Sprintf("%d", timeoutNs), "--absolute-timeouts")
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, terra, gaia))

	terraAfterSend, err := terra.GetBalance(ctx, terraUser.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)
	// Balance should have dropped by at least amount (plus gas)
	require.Less(t, terraAfterSend.Int64(), terraBefore.Sub(amount).Int64())

	// Wait past timeout on destination without relaying, then start relayer to process timeout
	require.NoError(t, testutil.WaitForBlocks(ctx, 8, gaia))

	// Start relayer to relay timeout-on-close
	require.NoError(t, r.StartRelayer(ctx, eRep, path))
	require.NoError(t, testutil.WaitForBlocks(ctx, 8, terra, gaia))

	// Assert timeout_packet observed on source
	terraH, _ := terra.Height(ctx)
	start := terraH - 40
	if start < 1 {
		start = 1
	}
	require.True(t, containsAnyEventInWindow(t, ctx, terra, start, terraH, "timeout_packet"))

	// Gaia should not have received funds
	gaiaAfter, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), gaiaIBCDenom)
	require.NoError(t, err)
	require.Equal(t, gaiaBefore, gaiaAfter)

	// Source balance should be refunded by at least 'amount' relative to post-send balance
	terraAfterTimeout, err := terra.GetBalance(ctx, terraUser.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)
	require.GreaterOrEqual(t, terraAfterTimeout.Int64(), terraAfterSend.Add(amount).Int64())
}
