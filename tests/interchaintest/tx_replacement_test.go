package interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTxReplacementAtCommittedSequence(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	numVals := 1
	numFullNodes := 0

	config, err := createConfig()
	require.NoError(t, err)
	config.ConfigFileOverrides = map[string]any{
		"config/config.toml": testutil.Toml{
			"consensus": testutil.Toml{
				"timeout_commit": "20s",
			},
		},
	}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra := chains[0].(*cosmos.CosmosChain)
	ic := interchaintest.NewInterchain().AddChain(terra)

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	require.NoError(t, testutil.WaitForBlocks(ctx, 2, terra))

	users := interchaintest.GetAndFundTestUsers(
		t,
		ctx,
		"replacement",
		sdkmath.NewInt(genesisWalletAmount),
		terra,
		terra,
	)
	sender := users[0]
	receiver := users[1]

	require.NoError(t, testutil.WaitForBlocks(ctx, 2, terra))

	accountInfo, err := terra.AuthQueryAccountInfo(ctx, sender.FormattedAddress())
	require.NoError(t, err)
	committedSeq := accountInfo.GetSequence()
	accountNumber := accountInfo.GetAccountNumber()

	initialReceiverBalance, err := terra.GetBalance(ctx, receiver.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)

	node := terra.GetNode()
	for seq := committedSeq; seq <= committedSeq+10; seq++ {
		require.NoError(t, broadcastSyncBankSendAtSequence(
			ctx,
			node,
			sender.KeyName(),
			receiver.FormattedAddress(),
			terra.Config().Denom,
			sdkmath.OneInt(),
			accountNumber,
			seq,
			fmt.Sprintf("queued-%d", seq),
		))
	}

	replacementAmount := sdkmath.NewInt(9)
	require.NoError(t, broadcastSyncBankSendAtSequence(
		ctx,
		node,
		sender.KeyName(),
		receiver.FormattedAddress(),
		terra.Config().Denom,
		replacementAmount,
		accountNumber,
		committedSeq,
		"replacement-at-committed-sequence",
	))

	require.NoError(t, testutil.WaitForBlocks(ctx, 2, terra))

	expectedFinalSeq := committedSeq + 11
	require.Eventually(t, func() bool {
		accountInfo, err := terra.AuthQueryAccountInfo(ctx, sender.FormattedAddress())
		return err == nil && accountInfo.GetSequence() == expectedFinalSeq
	}, time.Minute, 2*time.Second)

	finalReceiverBalance, err := terra.GetBalance(ctx, receiver.FormattedAddress(), terra.Config().Denom)
	require.NoError(t, err)

	// The original tx at committedSeq sent 1uluna. If it was not replaced, the
	// receiver would only gain 11uluna. The replacement sends 9uluna, followed
	// by the ten queued txs at committedSeq+1..+10.
	require.Equal(t, initialReceiverBalance.Add(sdkmath.NewInt(19)), finalReceiverBalance)
}

func broadcastSyncBankSendAtSequence(
	ctx context.Context,
	node *cosmos.ChainNode,
	keyName string,
	toAddress string,
	denom string,
	amount sdkmath.Int,
	accountNumber uint64,
	sequence uint64,
	memo string,
) error {
	stdout, _, err := node.Exec(ctx, node.TxCommand(
		keyName,
		"bank", "send",
		keyName,
		toAddress,
		fmt.Sprintf("%s%s", amount.String(), denom),
		"--account-number", fmt.Sprintf("%d", accountNumber),
		"--sequence", fmt.Sprintf("%d", sequence),
		"--offline",
		"--broadcast-mode", "sync",
		"--fees", fmt.Sprintf("6000000%s", denom),
		"--gas", "200000",
		"--note", memo,
	), node.Chain.Config().Env)
	if err != nil {
		return err
	}

	var txResp struct {
		Code   uint32 `json:"code"`
		RawLog string `json:"raw_log"`
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(stdout, &txResp); err != nil {
		return err
	}
	if txResp.Code != 0 {
		return fmt.Errorf("sync broadcast failed with code %d: %s", txResp.Code, txResp.RawLog)
	}
	if txResp.TxHash == "" {
		return fmt.Errorf("sync broadcast returned empty txhash")
	}
	return nil
}
