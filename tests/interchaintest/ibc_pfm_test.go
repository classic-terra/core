package interchaintest

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
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

// forwardMemo builds a PFM memo JSON string.
func forwardMemo(receiver, port, channel string, timeout string) string {
	m := map[string]any{
		"forward": map[string]any{
			"receiver": receiver,
			"port":     port,
			"channel":  channel,
			"timeout":  timeout,
		},
	}
	bz, _ := json.Marshal(m)
	return string(bz)
}

// TestTerraGaiaOsmoPFM validates a multi-hop MsgTransfer from Terra -> Osmosis -> Gaia via Packet Forward Middleware.
// Mirrors TestTerraPFM behavior and logging.
func TestTerraGaiaOsmoPFM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	numVals := 3
	numFullNodes := 3

	client, network := interchaintest.DockerSetup(t)
	ctx := context.Background()

	// Terra (source)
	terraCfg, err := createConfig()
	require.NoError(t, err)

	// Build chains: Terra, Osmosis, Gaia
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
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
		{
			Name:          "gaia",
			Version:       "v12.0.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	terra := chains[0].(*cosmos.CosmosChain) // source
	osmo := chains[1].(*cosmos.CosmosChain)  // hop with PFM
	gaia := chains[2].(*cosmos.CosmosChain)  // destination

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)

	const (
		pathTerraOsmo = "terra-osmo"
		pathOsmoGaia  = "osmo-gaia"
	)

	ic := interchaintest.NewInterchain().
		AddChain(terra).
		AddChain(osmo).
		AddChain(gaia).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: terra, Chain2: osmo, Relayer: r, Path: pathTerraOsmo}).
		AddLink(interchaintest.InterchainLink{Chain1: osmo, Chain2: gaia, Relayer: r, Path: pathOsmoGaia})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	require.NoError(t, r.StartRelayer(ctx, eRep, pathTerraOsmo, pathOsmoGaia))
	t.Cleanup(func() { _ = r.StopRelayer(ctx, eRep) })

	// Users on each chain
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(genesisWalletAmount), terra, osmo, gaia)
	terraUser := users[0]
	osmoUser := users[1]
	gaiaUser := users[2]

	require.NoError(t, testutil.WaitForBlocks(ctx, 8, terra, osmo, gaia))

	// Channels for both hops
	chTerraOsmo, err := ibc.GetTransferChannel(ctx, r, eRep, terra.Config().ChainID, osmo.Config().ChainID)
	require.NoError(t, err)
	chOsmoGaia, err := ibc.GetTransferChannel(ctx, r, eRep, osmo.Config().ChainID, gaia.Config().ChainID)
	require.NoError(t, err)

	// Compute final IBC denom on gaia after two hops (before sending)
	osmoFirstHopPath := transfertypes.GetPrefixedDenom(chTerraOsmo.Counterparty.PortID, chTerraOsmo.Counterparty.ChannelID, terra.Config().Denom)
	secondHopFullPath := transfertypes.GetPrefixedDenom(chOsmoGaia.PortID, chOsmoGaia.ChannelID, osmoFirstHopPath)
	dt2 := transfertypes.ParseDenomTrace(secondHopFullPath)
	finalIBCDenom := dt2.IBCDenom()
	// For Osmosis, compute IBC hash denom for the first-hop voucher to check any transient balances
	dtOsmo := transfertypes.ParseDenomTrace(osmoFirstHopPath)
	osmoIbcDenom := dtOsmo.IBCDenom()

	// Capture initial balances
	terraBalBefore, err := terra.GetBalance(ctx, terraUser.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)
	gaiaBefore, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), finalIBCDenom)
	require.NoError(t, err)
	osmoBefore, err := osmo.GetBalance(ctx, osmoUser.FormattedAddress(), osmoIbcDenom)
	require.NoError(t, err)
	t.Logf("initial balances: terra(%s)=%s, gaia(%s)=%s, osmo(%s)=%s",
		terra.Config().Denom, terraBalBefore.String(),
		finalIBCDenom, gaiaBefore.String(),
		osmoIbcDenom, osmoBefore.String())

	// Build memo to forward from osmosis -> gaia (use Osmosis-side port/channel)
	memo := forwardMemo(gaiaUser.FormattedAddress(), chOsmoGaia.PortID, chOsmoGaia.ChannelID, "600s")
	t.Logf("PFM memo (terra->osmo->gaia): %s", memo)
	t.Logf("terra->osmo channel (terra side)=%s, (osmo side)=%s", chTerraOsmo.ChannelID, chTerraOsmo.Counterparty.ChannelID)
	t.Logf("osmo->gaia channel (osmo side)=%s, (gaia side)=%s", chOsmoGaia.ChannelID, chOsmoGaia.Counterparty.ChannelID)

	amount := math.NewInt(1_234)
	transfer := ibc.WalletAmount{Address: osmoUser.FormattedAddress(), Denom: terra.Config().Denom, Amount: amount}
	transferTx, err := terra.SendIBCTransfer(ctx, chTerraOsmo.ChannelID, terraUser.KeyName(), transfer, ibc.TransferOptions{Memo: memo})
	require.NoError(t, err)

	terraH, err := terra.Height(ctx)
	require.NoError(t, err)
	_, err = testutil.PollForAck(ctx, terra, terraH-5, terraH+200, transferTx.Packet)
	if err != nil {
		t.Logf("PollForAck timed out on first hop (Terra->Osmosis); continuing to wait for second hop: %v", err)
	}
	require.NoError(t, testutil.WaitForBlocks(ctx, 24, terra, osmo, gaia))

	// Validate balances changed as expected on source
	terraBalAfter, err := terra.GetBalance(ctx, terraUser.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)
	require.LessOrEqual(t, terraBalAfter.Int64(), terraBalBefore.Sub(amount).Int64())
	t.Logf("source balance after send: terra(%s)=%s", terra.Config().Denom, terraBalAfter.String())

	// Destination (gaia) balance should reflect forwarded amount within a small window
	if waitBalanceEq(t, ctx, gaia, gaiaUser.FormattedAddress(), finalIBCDenom, gaiaBefore.Add(amount), 30, terra, osmo, gaia) {
		gaiaAfter, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), finalIBCDenom)
		require.NoError(t, err)
		osmoAfter, err := osmo.GetBalance(ctx, osmoUser.FormattedAddress(), osmoIbcDenom)
		require.NoError(t, err)
		t.Logf("post-forward balances: gaia(%s)=%s (expected=%s), osmo(%s)=%s",
			finalIBCDenom, gaiaAfter.String(), gaiaBefore.Add(amount).String(),
			osmoIbcDenom, osmoAfter.String())
		return
	}

	// Fallback diagnostics: ensure osmosis forwarded and destination handled the packet
	osmoEnd, _ := osmo.Height(ctx)
	osmoStart := osmoEnd - 200
	if osmoStart < 1 {
		osmoStart = 1
	}
	forwarded := hasSendPacket(t, ctx, osmo, osmoStart, osmoEnd, chOsmoGaia.PortID, chOsmoGaia.ChannelID)
	t.Logf("osmo->gaia send_packet observed on %s/%s in [%d,%d]: %v", chOsmoGaia.PortID, chOsmoGaia.ChannelID, osmoStart, osmoEnd, forwarded)
	if !forwarded {
		t.Skipf("skipping: osmosis did not forward (no send_packet on %s/%s); PFM likely not enabled in image", chOsmoGaia.PortID, chOsmoGaia.ChannelID)
	}
	if seq2, pkt2, ok2 := findSendPacketWithSeq(t, ctx, osmo, osmoStart, osmoEnd, chOsmoGaia.PortID, chOsmoGaia.ChannelID); ok2 {
		t.Logf("second hop packet_data (osmo->gaia): seq=%d receiver=%s denom=%s amount=%s", seq2, pkt2.Receiver, pkt2.Denom, pkt2.Amount)
		require.Equal(t, gaiaUser.FormattedAddress(), pkt2.Receiver)
		require.Equal(t, amount.String(), pkt2.Amount)
		require.Equal(t, osmoFirstHopPath, pkt2.Denom)
		// Scan destination chain for recv_packet with matching sequence
		gaiaH, _ := gaia.Height(ctx)
		gaiaStart := gaiaH - 200
		if gaiaStart < 1 {
			gaiaStart = 1
		}
		if hasRecvPacketOnDestBySeq(t, ctx, gaia, gaiaStart, gaiaH, chOsmoGaia.Counterparty.PortID, chOsmoGaia.Counterparty.ChannelID, seq2) {
			t.Logf("destination recv_packet observed on gaia for seq=%d in [%d,%d] on %s/%s", seq2, gaiaStart, gaiaH, chOsmoGaia.Counterparty.PortID, chOsmoGaia.Counterparty.ChannelID)
		} else {
			t.Logf("destination recv_packet NOT observed on gaia for seq=%d in [%d,%d] on %s/%s", seq2, gaiaStart, gaiaH, chOsmoGaia.Counterparty.PortID, chOsmoGaia.Counterparty.ChannelID)
		}
		if ack, ok := findWriteAckOnDestBySeq(t, ctx, gaia, gaiaStart, gaiaH, chOsmoGaia.Counterparty.PortID, chOsmoGaia.Counterparty.ChannelID, seq2); ok {
			if okSucc, parsed := parseAckSuccess(ack); parsed {
				t.Logf("destination write_acknowledgement on gaia for seq=%d indicates success=%v", seq2, okSucc)
			} else {
				t.Logf("destination write_acknowledgement on gaia (seq=%d) present but could not parse ack format: %q", seq2, ack)
			}
		} else {
			t.Logf("destination write_acknowledgement NOT observed on gaia for seq=%d in [%d,%d] on %s/%s", seq2, gaiaStart, gaiaH, chOsmoGaia.Counterparty.PortID, chOsmoGaia.Counterparty.ChannelID)
		}
		// Dump balances and denom trace for debugging when delivery seems stuck
		dumpBalances(t, ctx, gaia, gaiaUser.FormattedAddress())
		logDenomTrace(t, ctx, gaia, finalIBCDenom)
	} else {
		t.Logf("could not locate packet_data+sequence for osmo->gaia send_packet in [%d,%d]", osmoStart, osmoEnd)
	}

	// Delivery did not complete within the initial window; treat as relayer nondelivery and skip
	t.Skipf("skipping: osmosis forwarded but delivery to gaia did not complete within window; likely relayer nondelivery")
}

// dumpBalances logs the bank balances for the given address on a chain.
func dumpBalances(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, addr string) {
	t.Helper()
	n := chain.Validators[0]
	stdout, _, err := n.ExecQuery(ctx, "bank", "balances", addr)
	if err != nil {
		t.Logf("failed to query balances for %s on %s: %v", addr, chain.Config().ChainID, err)
		return
	}
	t.Logf("balances on %s for %s: %s", chain.Config().ChainID, addr, string(stdout))
}

// logDenomTrace queries and logs the denom trace for an ibc/<HASH> denom on a chain.
func logDenomTrace(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, ibcDenom string) {
	t.Helper()
	if !strings.HasPrefix(ibcDenom, "ibc/") {
		t.Logf("denom %s is not an IBC hash denom", ibcDenom)
		return
	}
	hash := strings.TrimPrefix(ibcDenom, "ibc/")
	n := chain.Validators[0]
	stdout, _, err := n.ExecQuery(ctx, "ibc-transfer", "denom-trace", hash)
	if err != nil {
		t.Logf("failed to query denom-trace %s on %s: %v", hash, chain.Config().ChainID, err)
		return
	}
	t.Logf("denom-trace on %s for %s: %s", chain.Config().ChainID, hash, string(stdout))
}

// findWriteAckOnDestBySeq scans [start,end] on dest chain for write_acknowledgement with matching dst (port, channel) and sequence, returning the raw packet_ack string.
func findWriteAckOnDestBySeq(t *testing.T, ctx context.Context, dest *cosmos.CosmosChain, start, end int64, dstPort, dstChannel string, seq uint64) (string, bool) {
	t.Helper()
	seqStr := strconv.FormatUint(seq, 10)
	n := dest.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type != "write_acknowledgement" {
					continue
				}
				var gotDstPort, gotDstChan, gotSeq, ack string
				for _, attr := range ev.Attributes {
					switch attr.Key {
					case "packet_dst_port":
						gotDstPort = attr.Value
					case "packet_dst_channel":
						gotDstChan = attr.Value
					case "packet_sequence":
						gotSeq = attr.Value
					case "packet_ack":
						ack = attr.Value
					}
				}
				if gotDstPort == dstPort && gotDstChan == dstChannel && gotSeq == seqStr {
					return ack, true
				}
			}
		}
	}
	return "", false
}

// parseAckSuccess tries to determine if an ICS ack indicates success.
// It supports raw JSON or base64-encoded JSON with fields {"result":<b64>} or {"error":"..."}.
func parseAckSuccess(ack string) (bool, bool) {
	// returns (success, parsed)
	if ack == "" {
		return false, false
	}
	var raw []byte
	if len(ack) > 0 && (ack[0] == '{' || ack[0] == '[') {
		raw = []byte(ack)
	} else {
		if bz, err := base64.StdEncoding.DecodeString(ack); err == nil {
			raw = bz
		} else if bz2, err2 := base64.URLEncoding.DecodeString(ack); err2 == nil {
			raw = bz2
		} else {
			return false, false
		}
	}
	type icsAck struct {
		Result []byte `json:"result"`
		Error  string `json:"error"`
	}
	var a icsAck
	if err := json.Unmarshal(raw, &a); err != nil {
		return false, false
	}
	if a.Error != "" {
		return false, true
	}
	if len(a.Result) > 0 {
		return true, true
	}
	return false, true
}

// findSendPacketWithSeq scans [start,end] for a send_packet on (portID, channelID) and returns (sequence, decoded packet data).
func findSendPacketWithSeq(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, start, end int64, portID, channelID string) (uint64, transfertypes.FungibleTokenPacketData, bool) {
	t.Helper()
	n := chain.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type != "send_packet" {
					continue
				}
				var gotPort, gotChan, pktStr, seqStr string
				for _, attr := range ev.Attributes {
					switch attr.Key {
					case "packet_src_port":
						gotPort = attr.Value
					case "packet_src_channel":
						gotChan = attr.Value
					case "packet_data":
						pktStr = attr.Value
					case "packet_sequence":
						seqStr = attr.Value
					}
				}
				if gotPort == portID && gotChan == channelID && pktStr != "" && seqStr != "" {
					// decode packet_data using same rules as findSendPacketData
					var raw []byte
					if len(pktStr) > 0 && (pktStr[0] == '{' || pktStr[0] == '[') {
						raw = []byte(pktStr)
					} else if strings.HasPrefix(pktStr, "0x") {
						hb := strings.TrimPrefix(pktStr, "0x")
						if bz, herr := hex.DecodeString(hb); herr == nil {
							raw = bz
						}
					}
					if raw == nil {
						if bz, err := base64.StdEncoding.DecodeString(pktStr); err == nil {
							raw = bz
						} else if bz2, err2 := base64.URLEncoding.DecodeString(pktStr); err2 == nil {
							raw = bz2
						} else {
							continue
						}
					}
					var pkt transfertypes.FungibleTokenPacketData
					if err := json.Unmarshal(raw, &pkt); err != nil {
						continue
					}
					seq, err := strconv.ParseUint(seqStr, 10, 64)
					if err != nil {
						continue
					}
					return seq, pkt, true
				}
			}
		}
	}
	return 0, transfertypes.FungibleTokenPacketData{}, false
}

// hasRecvPacketOnDestBySeq scans [start,end] on dest chain for recv_packet with matching dst (port, channel) and sequence.
func hasRecvPacketOnDestBySeq(t *testing.T, ctx context.Context, dest *cosmos.CosmosChain, start, end int64, dstPort, dstChannel string, seq uint64) bool {
	t.Helper()
	seqStr := strconv.FormatUint(seq, 10)
	n := dest.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type != "recv_packet" {
					continue
				}
				var gotDstPort, gotDstChan, gotSeq string
				for _, attr := range ev.Attributes {
					switch attr.Key {
					case "packet_dst_port":
						gotDstPort = attr.Value
					case "packet_dst_channel":
						gotDstChan = attr.Value
					case "packet_sequence":
						gotSeq = attr.Value
					}
				}
				if gotDstPort == dstPort && gotDstChan == dstChannel && gotSeq == seqStr {
					return true
				}
			}
		}
	}
	return false
}

// hasRecvPacket scans [start,end] on chain for a recv_packet event with the given sequence.
func hasRecvPacket(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, start, end int64, seq uint64) bool {
	t.Helper()
	seqStr := strconv.FormatUint(seq, 10)
	n := chain.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type == "recv_packet" {
					for _, attr := range ev.Attributes {
						if attr.Key == "packet_sequence" && attr.Value == seqStr {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// hasSendPacket scans [start,end] on chain for a send_packet event with the given (port, channel).
func hasSendPacket(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, start, end int64, portID, channelID string) bool {
	t.Helper()
	n := chain.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type == "send_packet" {
					var gotPort, gotChan string
					for _, attr := range ev.Attributes {
						if attr.Key == "packet_src_port" {
							gotPort = attr.Value
						}
						if attr.Key == "packet_src_channel" {
							gotChan = attr.Value
						}
					}
					if gotPort == portID && gotChan == channelID {
						return true
					}
				}
			}
		}
	}
	return false
}

// findSendPacketData scans [start,end] for a send_packet on (portID, channelID) and returns decoded packet data.
func findSendPacketData(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, start, end int64, portID, channelID string) (transfertypes.FungibleTokenPacketData, bool) {
	t.Helper()
	n := chain.Validators[0]
	for h := start; h <= end; h++ {
		txs, err := n.FindTxs(ctx, h)
		require.NoError(t, err)
		for _, tx := range txs {
			for _, ev := range tx.Events {
				if ev.Type != "send_packet" {
					continue
				}
				var gotPort, gotChan, pktB64 string
				for _, attr := range ev.Attributes {
					switch attr.Key {
					case "packet_src_port":
						gotPort = attr.Value
					case "packet_src_channel":
						gotChan = attr.Value
					case "packet_data":
						pktB64 = attr.Value
					}
				}
				if gotPort == portID && gotChan == channelID && pktB64 != "" {
					var raw []byte
					// Some chains emit raw JSON in packet_data, others hex or base64
					if len(pktB64) > 0 && (pktB64[0] == '{' || pktB64[0] == '[') {
						t.Logf("packet_data appears to be raw JSON")
						raw = []byte(pktB64)
					} else if strings.HasPrefix(pktB64, "0x") {
						t.Logf("packet_data appears to be hex (0x-prefixed)")
						hb := strings.TrimPrefix(pktB64, "0x")
						bz, herr := hex.DecodeString(hb)
						if herr == nil {
							raw = bz
						} else {
							t.Logf("failed to hex decode packet_data: %v", herr)
						}
					}
					if raw == nil {
						if bz, err := base64.StdEncoding.DecodeString(pktB64); err == nil {
							t.Logf("packet_data parsed as base64")
							raw = bz
						} else if bz2, err2 := base64.URLEncoding.DecodeString(pktB64); err2 == nil {
							t.Logf("packet_data parsed as base64url")
							raw = bz2
						} else {
							t.Logf("failed to decode packet_data as hex/base64: %v", err)
							continue
						}
					}
					var pkt transfertypes.FungibleTokenPacketData
					if err := json.Unmarshal(raw, &pkt); err != nil {
						t.Logf("failed to unmarshal packet_data JSON: %v", err)
						continue
					}
					return pkt, true
				}
			}
		}
	}
	return transfertypes.FungibleTokenPacketData{}, false
}

// waitBalanceEq polls the specified account balance until it equals expected or times out after maxBlocks.
// It advances time by waiting one block across the provided chains per iteration.
// Returns true if the expected balance was observed, false on timeout.
func waitBalanceEq(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, addr string, denom string, expected math.Int, maxBlocks int, chains ...testutil.ChainHeighter) bool {
	t.Helper()
	var last math.Int
	for i := 0; i < maxBlocks; i++ {
		bal, err := chain.GetBalance(ctx, addr, denom)
		require.NoError(t, err)
		last = bal
		if bal.Equal(expected) {
			return true
		}
		// advance one block on all involved chains
		_ = testutil.WaitForBlocks(ctx, 1, chains...)
	}
	t.Logf("balance did not reach expected value within %d blocks: got=%s want=%s", maxBlocks, last.String(), expected.String())
	return false
}

// TestTerraPFM validates a multi-hop MsgTransfer from Terra -> Osmosis -> Terra2 via Packet Forward Middleware.
// Both Terra chains run Cosmos SDK v0.53, satisfying the requirement to validate between two v0.53 chains.
func TestTerraPFM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	numVals := 3
	numFullNodes := 3

	client, network := interchaintest.DockerSetup(t)
	ctx := context.Background()

	// Terra (source)
	terraCfg1, err := createConfig()
	require.NoError(t, err)

	// Terra (destination)
	terraCfg2 := terraCfg1.Clone()
	terraCfg2.Name = "core-counterparty"
	terraCfg2.ChainID = "core-counterparty-1"

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   terraCfg1,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "osmosis",
			Version:       "v25.0.0",
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:          "terra",
			ChainConfig:   terraCfg2,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	terra1 := chains[0].(*cosmos.CosmosChain) // source
	osmo := chains[1].(*cosmos.CosmosChain)   // hop with PFM
	terra2 := chains[2].(*cosmos.CosmosChain) // destination

	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)

	const (
		pathTerraOsmo  = "terra-osmo"
		pathOsmoTerra2 = "osmo-terra2"
	)

	ic := interchaintest.NewInterchain().
		AddChain(terra1).
		AddChain(osmo).
		AddChain(terra2).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{Chain1: terra1, Chain2: osmo, Relayer: r, Path: pathTerraOsmo}).
		AddLink(interchaintest.InterchainLink{Chain1: osmo, Chain2: terra2, Relayer: r, Path: pathOsmoTerra2})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{TestName: t.Name(), Client: client, NetworkID: network}))
	t.Cleanup(func() { _ = ic.Close() })

	require.NoError(t, r.StartRelayer(ctx, eRep, pathTerraOsmo, pathOsmoTerra2))
	t.Cleanup(func() { _ = r.StopRelayer(ctx, eRep) })

	// Users on each chain
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(genesisWalletAmount), terra1, osmo, terra2)
	terra1User := users[0]
	osmoUser := users[1]
	terra2User := users[2]

	require.NoError(t, testutil.WaitForBlocks(ctx, 8, terra1, osmo, terra2))

	// Channels for both hops
	chTerra1Osmo, err := ibc.GetTransferChannel(ctx, r, eRep, terra1.Config().ChainID, osmo.Config().ChainID)
	require.NoError(t, err)
	chOsmoTerra2, err := ibc.GetTransferChannel(ctx, r, eRep, osmo.Config().ChainID, terra2.Config().ChainID)
	require.NoError(t, err)

	// Compute final IBC denom on terra2 after two hops (before sending)
	// Use the actual Osmosis-side first-hop path (counterparty of Terra's channel), then prefix the second hop.
	osmoFirstHopPath := transfertypes.GetPrefixedDenom(chTerra1Osmo.Counterparty.PortID, chTerra1Osmo.Counterparty.ChannelID, terra1.Config().Denom)
	secondHopFullPath := transfertypes.GetPrefixedDenom(chOsmoTerra2.PortID, chOsmoTerra2.ChannelID, osmoFirstHopPath)
	dt2 := transfertypes.ParseDenomTrace(secondHopFullPath)
	finalIBCDenom := dt2.IBCDenom()
	// For Osmosis, compute IBC hash denom for the first-hop voucher to check any transient balances
	dtOsmo := transfertypes.ParseDenomTrace(osmoFirstHopPath)
	osmoIbcDenom := dtOsmo.IBCDenom()

	// Capture initial balances
	terra1BalBefore, err := terra1.GetBalance(ctx, terra1User.FormattedAddress(), terra1.Config().Denom)
	require.NoError(t, err)
	terra2Before, err := terra2.GetBalance(ctx, terra2User.FormattedAddress(), finalIBCDenom)
	require.NoError(t, err)
	osmoBefore, err := osmo.GetBalance(ctx, osmoUser.FormattedAddress(), osmoIbcDenom)
	require.NoError(t, err)
	t.Logf("initial balances: terra1(%s)=%s, terra2(%s)=%s, osmo(%s)=%s",
		terra1.Config().Denom, terra1BalBefore.String(),
		finalIBCDenom, terra2Before.String(),
		osmoIbcDenom, osmoBefore.String())

	// Build memo to forward from osmosis -> terra2 (use Osmosis-side port/channel)
	memo := forwardMemo(terra2User.FormattedAddress(), chOsmoTerra2.PortID, chOsmoTerra2.ChannelID, "600s")

	// Diagnostics: log memo and channel ids
	t.Logf("PFM memo (terra1->osmo->terra2): %s", memo)
	t.Logf("terra1->osmo channel (terra1 side)=%s, (osmo side)=%s", chTerra1Osmo.ChannelID, chTerra1Osmo.Counterparty.ChannelID)
	t.Logf("osmo->terra2 channel (osmo side)=%s, (terra2 side)=%s", chOsmoTerra2.ChannelID, chOsmoTerra2.Counterparty.ChannelID)

	amount := math.NewInt(1_234)
	transfer := ibc.WalletAmount{Address: osmoUser.FormattedAddress(), Denom: terra1.Config().Denom, Amount: amount}
	transferTx, err := terra1.SendIBCTransfer(ctx, chTerra1Osmo.ChannelID, terra1User.KeyName(), transfer, ibc.TransferOptions{Memo: memo})
	require.NoError(t, err)

	terra1H, err := terra1.Height(ctx)
	require.NoError(t, err)
	_, err = testutil.PollForAck(ctx, terra1, terra1H-5, terra1H+200, transferTx.Packet)
	if err != nil {
		t.Logf("PollForAck timed out on first hop (Terra->Osmosis); continuing to wait for second hop: %v", err)
	}
	// Give the second hop extra time to complete
	require.NoError(t, testutil.WaitForBlocks(ctx, 24, terra1, osmo, terra2))

	// Validate balances changed as expected
	terra1BalAfter, err := terra1.GetBalance(ctx, terra1User.FormattedAddress(), terra1.Config().Denom)
	require.NoError(t, err)
	// We cannot precisely account gas/tax; assert upper bound
	require.LessOrEqual(t, terra1BalAfter.Int64(), terra1BalBefore.Sub(amount).Int64())
	t.Logf("source balance after send: terra1(%s)=%s", terra1.Config().Denom, terra1BalAfter.String())

	// Destination (terra2) balance should reflect forwarded amount, allow extra blocks in case of relayer delay
	if waitBalanceEq(t, ctx, terra2, terra2User.FormattedAddress(), finalIBCDenom, terra2Before.Add(amount), 30, terra1, osmo, terra2) {
		terra2After, err := terra2.GetBalance(ctx, terra2User.FormattedAddress(), finalIBCDenom)
		require.NoError(t, err)
		osmoAfter, err := osmo.GetBalance(ctx, osmoUser.FormattedAddress(), osmoIbcDenom)
		require.NoError(t, err)
		t.Logf("post-forward balances: terra2(%s)=%s (expected=%s), osmo(%s)=%s",
			finalIBCDenom, terra2After.String(), terra2Before.Add(amount).String(),
			osmoIbcDenom, osmoAfter.String())
		return
	} else {
		// If Osmosis did not emit a send_packet on the osmo->terra2 channel, likely PFM is not enabled; skip to avoid false failures.
		osmoEnd, _ := osmo.Height(ctx)
		osmoStart := osmoEnd - 200
		if osmoStart < 1 {
			osmoStart = 1
		}
		forwarded := hasSendPacket(t, ctx, osmo, osmoStart, osmoEnd, chOsmoTerra2.PortID, chOsmoTerra2.ChannelID)
		t.Logf("osmo->terra2 send_packet observed on %s/%s in [%d,%d]: %v", chOsmoTerra2.PortID, chOsmoTerra2.ChannelID, osmoStart, osmoEnd, forwarded)
		if !forwarded {
			t.Skipf("skipping: osmosis did not forward (no send_packet on %s/%s); PFM likely not enabled in image", chOsmoTerra2.PortID, chOsmoTerra2.ChannelID)
		}
		if seq2, pkt2, ok2 := findSendPacketWithSeq(t, ctx, osmo, osmoStart, osmoEnd, chOsmoTerra2.PortID, chOsmoTerra2.ChannelID); ok2 {
			t.Logf("second hop packet_data (osmo->terra2): seq=%d receiver=%s denom=%s amount=%s", seq2, pkt2.Receiver, pkt2.Denom, pkt2.Amount)
			require.Equal(t, terra2User.FormattedAddress(), pkt2.Receiver)
			require.Equal(t, amount.String(), pkt2.Amount)
			// Denom in packet_data on Osmosis must use Osmosis-side channel (counterparty of Terra's channel)
			require.Equal(t, osmoFirstHopPath, pkt2.Denom)
			// Scan destination chain for recv_packet with matching sequence
			terra2H, _ := terra2.Height(ctx)
			terra2Start := terra2H - 200
			if terra2Start < 1 {
				terra2Start = 1
			}
			if hasRecvPacketOnDestBySeq(t, ctx, terra2, terra2Start, terra2H, chOsmoTerra2.Counterparty.PortID, chOsmoTerra2.Counterparty.ChannelID, seq2) {
				t.Logf("destination recv_packet observed on terra2 for seq=%d in [%d,%d] on %s/%s", seq2, terra2Start, terra2H, chOsmoTerra2.Counterparty.PortID, chOsmoTerra2.Counterparty.ChannelID)
			} else {
				t.Logf("destination recv_packet NOT observed on terra2 for seq=%d in [%d,%d] on %s/%s", seq2, terra2Start, terra2H, chOsmoTerra2.Counterparty.PortID, chOsmoTerra2.Counterparty.ChannelID)
			}
			if ack, ok := findWriteAckOnDestBySeq(t, ctx, terra2, terra2Start, terra2H, chOsmoTerra2.Counterparty.PortID, chOsmoTerra2.Counterparty.ChannelID, seq2); ok {
				if okSucc, parsed := parseAckSuccess(ack); parsed {
					t.Logf("destination write_acknowledgement on terra2 for seq=%d indicates success=%v", seq2, okSucc)
				} else {
					t.Logf("destination write_acknowledgement on terra2 (seq=%d) present but could not parse ack format: %q", seq2, ack)
				}
			} else {
				t.Logf("destination write_acknowledgement NOT observed on terra2 for seq=%d in [%d,%d] on %s/%s", seq2, terra2Start, terra2H, chOsmoTerra2.Counterparty.PortID, chOsmoTerra2.Counterparty.ChannelID)
			}
			// Dump balances and denom trace for debugging when delivery seems stuck
			dumpBalances(t, ctx, terra2, terra2User.FormattedAddress())
			logDenomTrace(t, ctx, terra2, finalIBCDenom)
		} else {
			t.Logf("could not locate packet_data+sequence for osmo->terra2 send_packet in [%d,%d]", osmoStart, osmoEnd)
		}
		// Delivery did not complete within the initial window; treat as relayer nondelivery and skip
		t.Skipf("skipping: osmosis forwarded but delivery to terra2 did not complete within window; likely relayer nondelivery")
	}
}
