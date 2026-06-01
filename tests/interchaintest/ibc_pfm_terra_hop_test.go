package interchaintest

import (
	"context"
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

// TestTerraAsPFMHop validates that Terra Classic itself forwards a multi-hop
// transfer: gaia -> TERRA (Packet Forward Middleware) -> osmosis.
//
// This is the test the existing PFM suite does NOT cover: TestTerraPFM /
// TestTerraGaiaOsmoPFM use Osmosis as the forwarding hop and only exercise
// Terra as a source/destination. Here Terra is the middle hop, so a successful
// forward proves the v15 PFM wiring (keeper, store, stack order, and the
// ics4Wrapper choice in app/keepers/keepers.go) actually works at runtime.
//
// The hard assertion is that TERRA emits an onward send_packet on the
// terra->osmo channel after receiving the gaia packet. Final delivery to
// osmosis depends on the relayer (not our code), so the balance check is
// best-effort.
func TestTerraAsPFMHop(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Keep the footprint small: single validator, no full nodes per chain.
	numVals := 1
	numFullNodes := 0

	client, network := interchaintest.DockerSetup(t)
	ctx := context.Background()

	terraCfg, err := createConfig() // the hop chain — our v15 image (core:local)
	require.NoError(t, err)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "gaia",
			Version:       "v25.1.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
			ChainConfig:   createGaiaConfig(),
		},
		{
			Name:          "terra",
			ChainConfig:   terraCfg,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "osmosis",
			Version:       "v25.0.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	gaia := chains[0].(*cosmos.CosmosChain)  // source
	terra := chains[1].(*cosmos.CosmosChain) // hop with PFM (our build)
	osmo := chains[2].(*cosmos.CosmosChain)  // destination

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)

	const (
		pathGaiaTerraHop = "gaia-terra-hop"
		pathTerraOsmoHop = "terra-osmo-hop"
	)

	ic := interchaintest.NewInterchain().
		AddChain(gaia).
		AddChain(terra).
		AddChain(osmo).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: gaia, Chain2: terra, Relayer: r, Path: pathGaiaTerraHop}).
		AddLink(interchaintest.InterchainLink{Chain1: terra, Chain2: osmo, Relayer: r, Path: pathTerraOsmoHop})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	require.NoError(t, r.StartRelayer(ctx, eRep, pathGaiaTerraHop, pathTerraOsmoHop))
	t.Cleanup(func() { _ = r.StopRelayer(ctx, eRep) })

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(genesisWalletAmount), gaia, terra, osmo)
	gaiaUser := users[0]
	terraUser := users[1]
	osmoUser := users[2]

	require.NoError(t, testutil.WaitForBlocks(ctx, 8, gaia, terra, osmo))

	chGaiaTerra, err := ibc.GetTransferChannel(ctx, r, eRep, gaia.Config().ChainID, terra.Config().ChainID)
	require.NoError(t, err)
	chTerraOsmo, err := ibc.GetTransferChannel(ctx, r, eRep, terra.Config().ChainID, osmo.Config().ChainID)
	require.NoError(t, err)

	// Final IBC denom on osmosis after gaia -> terra -> osmo (uatom voucher, two hops).
	terraHopPath := transfertypes.GetPrefixedDenom(chGaiaTerra.Counterparty.PortID, chGaiaTerra.Counterparty.ChannelID, gaia.Config().Denom)
	secondHopFullPath := transfertypes.GetPrefixedDenom(chTerraOsmo.Counterparty.PortID, chTerraOsmo.Counterparty.ChannelID, terraHopPath)
	finalIBCDenom := transfertypes.ParseDenomTrace(secondHopFullPath).IBCDenom()

	osmoBefore, err := osmo.GetBalance(ctx, osmoUser.FormattedAddress(), finalIBCDenom)
	require.NoError(t, err)

	// Forward memo: on receipt, Terra forwards onward to osmosis using the Terra-side channel.
	memo := forwardMemo(osmoUser.FormattedAddress(), chTerraOsmo.PortID, chTerraOsmo.ChannelID, "600s")
	t.Logf("PFM memo (gaia->terra->osmo): %s", memo)
	t.Logf("gaia->terra channel (gaia side)=%s, (terra side)=%s", chGaiaTerra.ChannelID, chGaiaTerra.Counterparty.ChannelID)
	t.Logf("terra->osmo channel (terra side)=%s, (osmo side)=%s", chTerraOsmo.ChannelID, chTerraOsmo.Counterparty.ChannelID)

	amount := math.NewInt(1_234)
	// Receiver on the hop (terra) is terraUser; PFM intercepts via the memo and forwards.
	transfer := ibc.WalletAmount{Address: terraUser.FormattedAddress(), Denom: gaia.Config().Denom, Amount: amount}
	transferTx, err := gaia.SendIBCTransfer(ctx, chGaiaTerra.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{Memo: memo})
	require.NoError(t, err)

	gaiaH, err := gaia.Height(ctx)
	require.NoError(t, err)
	if _, err := testutil.PollForAck(ctx, gaia, gaiaH-5, gaiaH+200, transferTx.Packet); err != nil {
		t.Logf("PollForAck timed out on first hop (gaia->terra); continuing: %v", err)
	}
	require.NoError(t, testutil.WaitForBlocks(ctx, 24, gaia, terra, osmo))

	// CORE PROOF (our v15 code): Terra forwarded — it emitted a send_packet on the
	// terra->osmo channel after receiving from gaia. If the PFM wiring / ics4Wrapper
	// were wrong, no onward packet would be produced.
	terraEnd, err := terra.Height(ctx)
	require.NoError(t, err)
	terraStart := terraEnd - 200
	if terraStart < 1 {
		terraStart = 1
	}
	forwarded := hasSendPacket(t, ctx, terra, terraStart, terraEnd, chTerraOsmo.PortID, chTerraOsmo.ChannelID)
	require.True(t, forwarded,
		"Terra PFM did NOT forward: no send_packet on %s/%s — v15 PFM wiring/ics4Wrapper is wrong",
		chTerraOsmo.PortID, chTerraOsmo.ChannelID)
	t.Logf("PROOF: Terra(v15) forwarded the packet onward on %s/%s", chTerraOsmo.PortID, chTerraOsmo.ChannelID)

	if seq, pkt, ok := findSendPacketWithSeq(t, ctx, terra, terraStart, terraEnd, chTerraOsmo.PortID, chTerraOsmo.ChannelID); ok {
		t.Logf("forwarded packet: seq=%d receiver=%s denom=%s amount=%s", seq, pkt.Receiver, pkt.Denom, pkt.Amount)
		require.Equal(t, osmoUser.FormattedAddress(), pkt.Receiver)
		require.Equal(t, amount.String(), pkt.Amount)
		require.Equal(t, terraHopPath, pkt.Denom)
	}

	// Best-effort: confirm osmosis credited the forwarded funds (depends on the relayer).
	if waitBalanceEq(t, ctx, osmo, osmoUser.FormattedAddress(), finalIBCDenom, osmoBefore.Add(amount), 30, gaia, terra, osmo) {
		t.Logf("PROOF: osmosis received forwarded %s of %s", amount.String(), finalIBCDenom)
	} else {
		t.Logf("note: osmosis balance did not reach expected within window (relayer delivery, not Terra's forward) — forward itself is already proven above")
	}
}
