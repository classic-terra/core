package oracle_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/classic-terra/core/v3/x/oracle/keeper"
	"github.com/classic-terra/core/v3/x/oracle/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	stakingAmt = sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)

	randomExchangeRate = sdkmath.LegacyNewDec(1700)
)

func setupWithSmallVotingPower(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.VotePeriod = 1
	params.SlashWindow = 100
	params.RewardDistributionWindow = 100
	input.OracleKeeper.SetParams(input.Ctx, params)
	h := keeper.NewMsgServerImpl(input.OracleKeeper)

	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err := stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[0], keeper.ValPubKeys[0], sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)))
	require.NoError(t, err)

	// mark validator as bonded for tests (no EndBlocker here)
	val, err := input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	require.NoError(t, err)
	val.Status = stakingtypes.Bonded
	input.StakingKeeper.SetValidator(input.Ctx, val)

	return input, h
}

func setup(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.VotePeriod = 1
	params.SlashWindow = 100
	params.RewardDistributionWindow = 100
	input.OracleKeeper.SetParams(input.Ctx, params)
	h := keeper.NewMsgServerImpl(input.OracleKeeper)

	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)

	// Validator created
	_, err := stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[0], keeper.ValPubKeys[0], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[1], keeper.ValPubKeys[1], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[2], keeper.ValPubKeys[2], stakingAmt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(input.Ctx)

	return input, h
}

func setupVal5(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.VotePeriod = 1
	params.SlashWindow = 100
	params.RewardDistributionWindow = 100
	input.OracleKeeper.SetParams(input.Ctx, params)
	h := keeper.NewMsgServerImpl(input.OracleKeeper)

	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)

	// Validator created
	_, err := stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[0], keeper.ValPubKeys[0], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[1], keeper.ValPubKeys[1], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[2], keeper.ValPubKeys[2], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[3], keeper.ValPubKeys[3], stakingAmt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, keeper.NewTestMsgCreateValidator(keeper.ValAddrs[4], keeper.ValPubKeys[4], stakingAmt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(input.Ctx)

	return input, h
}
