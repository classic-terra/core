package interchaintest

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	"github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/classic-terra/core/v2/test/interchaintest/helpers"
)

// TestValidator is a basic test to accrue enough token to join active validator set, gets slashed for missing or tombstoned for double signing
func TestValidator(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Create chain factory with Terra Classic
	numVals := 5
	numFullNodes := 3

	config, err := createConfig()
	require.NoError(t, err)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "terra",
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	terra := chains[0].(*cosmos.CosmosChain)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
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

		// This can be used to write to the block database which will index all block data e.g. txs, msgs, events, etc.
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})
	err = testutil.WaitForBlocks(ctx, 1, terra)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 1, terra)
	require.NoError(t, err)

	err = terra.Validators[1].StopContainer(ctx)
	require.NoError(t, err)

	stdout, _, err := terra.Validators[1].ExecBin(ctx, "status")
	require.Error(t, err)
	require.Empty(t, stdout)

	err = testutil.WaitForBlocks(ctx, 21, terra)
	require.NoError(t, err)

	// Get all Validators
	stdout, _, err = terra.Validators[0].ExecQuery(ctx, "staking", "validators")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	terraValidators, pubKeys, err := helpers.UnmarshalValidators(*config.EncodingConfig, stdout)
	require.NoError(t, err)
	require.Equal(t, len(terraValidators), 5)

	var val1PubKey cryptotypes.PubKey
	count := 0
	for i, val := range terraValidators {
		if val.Jailed == true {
			count++
			val1PubKey = pubKeys[i]
		}
	}
	require.Equal(t, count, 1)
	bech32Addr, err := bech32.ConvertAndEncode("terravalcons", sdk.ConsAddress(val1PubKey.Address()))
	require.NoError(t, err)

	// Get Slashing Params
	stdout, _, err = terra.Validators[0].ExecQuery(ctx, "slashing", "params")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	signedBlocksWindow, err := helpers.GetSignedBlocksWindow(stdout)
	require.NoError(t, err)
	require.Equal(t, signedBlocksWindow, int64(20))

	// Get SigningInfos
	stdout, _, err = terra.Validators[0].ExecQuery(ctx, "slashing", "signing-infos")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	var signingInfosResp slashingtypes.QuerySigningInfosResponse
	err = codec.NewLegacyAmino().UnmarshalJSON(stdout, &signingInfosResp)
	require.NoError(t, err)

	count = 0
	defaultTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	infos := signingInfosResp.Info
	for _, info := range infos {
		if info.JailedUntil != defaultTime {
			count++
			require.Equal(t, info.Address, bech32Addr)
		}
	}
	require.NoError(t, err)
	require.Equal(t, count, 1)
}
