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

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/classic-terra/core/v3/test/interchaintest/helpers"
)

// TestValidator is a basic test to accrue enough token to join active validator set,
// then ensure a stopped validator gets jailed and has signing info updated.
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
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// let chain produce some blocks
	require.NoError(t, testutil.WaitForBlocks(ctx, 1, terra))

	// stop one validator so it starts missing votes
	require.NoError(t, terra.Validators[1].StopContainer(ctx))

	stdout, _, err := terra.Validators[1].ExecBin(ctx, "status")
	require.Error(t, err)
	require.Empty(t, stdout)

	// wait long enough to trip slashing window
	require.NoError(t, testutil.WaitForBlocks(ctx, 21, terra))

	// --- Query all validators
	stdout, _, err = terra.Validators[0].ExecQuery(
		ctx, "staking", "validators",
		"--output", "json",
	)
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	terraValidators, pubKeys, err := helpers.UnmarshalValidators(*config.EncodingConfig, stdout)
	require.NoError(t, err)
	require.Equal(t, 5, len(terraValidators.Validators))

	// find exactly one jailed validator and capture its consensus pubkey
	var val1PubKey cryptotypes.PubKey
	count := 0
	for i, val := range terraValidators.Validators {
		if val.Jailed {
			count++
			val1PubKey = pubKeys[i]
		}
	}
	require.Equal(t, 1, count)
	require.NotNil(t, val1PubKey)

	// Derive raw consensus address bytes from the pubkey (HRP-agnostic)
	consAddrBytes := sdk.ConsAddress(val1PubKey.Address())

	// --- Get Slashing Params ---
	stdout, _, err = terra.Validators[0].ExecQuery(ctx, "slashing", "params", "--output", "json")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	signedBlocksWindow, err := helpers.GetSignedBlocksWindow(stdout)
	require.NoError(t, err)
	require.Equal(t, int64(20), signedBlocksWindow)

	// --- Get Signing Infos ---
	stdout, _, err = terra.Validators[0].ExecQuery(ctx, "slashing", "signing-infos", "--output", "json")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	var signingInfosResp slashingtypes.QuerySigningInfosResponse
	err = codec.NewLegacyAmino().UnmarshalJSON(stdout, &signingInfosResp)
	require.NoError(t, err)

	count = 0
	defaultTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	for _, info := range signingInfosResp.Info {
		if info.JailedUntil != defaultTime {
			count++
			// Decode whatever HRP the chain used and compare raw bytes
			_, addrBytes, err := bech32.DecodeAndConvert(info.Address)
			require.NoError(t, err)
			require.Equal(t, consAddrBytes, sdk.ConsAddress(addrBytes))
		}
	}
	require.Equal(t, 1, count)
}
