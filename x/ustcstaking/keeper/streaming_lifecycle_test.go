package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestStreamingStakeUnbondClaimAndWithdrawSequence(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	k.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt())
	state.UnallocatedRewards = math.NewInt(1000)
	k.SetRewardState(ctx, state)
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(1000))
	server := NewMsgServerImpl(k)
	ownerA, ownerB := testAddress(), testAddress()

	responseA, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{Owner: ownerA, Amount: sdk.NewInt64Coin(types.BondDenom, 100), LockTierId: 1})
	require.NoError(t, err)
	dayOne := ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))
	responseB, err := server.Stake(sdk.WrapSDKContext(dayOne), &types.MsgStake{Owner: ownerB, Amount: sdk.NewInt64Coin(types.BondDenom, 100), LockTierId: 1})
	require.NoError(t, err)
	state = k.GetRewardState(dayOne)
	require.Equal(t, math.NewInt(100), state.AllocatedRewards)
	require.Equal(t, math.NewInt(900), state.UnallocatedRewards)
	require.True(t, state.RewardIndex.Equal(math.LegacyOneDec()))

	dayTwo := dayOne.WithBlockTime(dayOne.BlockTime().Add(24 * time.Hour))
	_, err = server.BeginUnbonding(sdk.WrapSDKContext(dayTwo), &types.MsgBeginUnbonding{Owner: ownerA, PositionId: responseA.PositionId})
	require.NoError(t, err)
	positionA, _ := k.GetPosition(dayTwo, responseA.PositionId)
	positionB, _ := k.GetPosition(dayTwo, responseB.PositionId)
	require.Equal(t, math.NewInt(150), positionA.ClaimableRewards.Amount)
	require.True(t, positionA.Shares.IsZero())
	require.True(t, positionB.ClaimableRewards.Amount.IsZero())
	require.Equal(t, math.NewInt(100), k.GetRewardState(dayTwo).TotalShares)

	dayThree := dayTwo.WithBlockTime(dayTwo.BlockTime().Add(24 * time.Hour))
	claim, err := server.ClaimRewards(sdk.WrapSDKContext(dayThree), &types.MsgClaimRewards{Owner: ownerA, PositionId: responseA.PositionId})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(150), claim.Amount.Amount)
	positionA, _ = k.GetPosition(dayThree, responseA.PositionId)
	require.True(t, positionA.ClaimableRewards.Amount.IsZero())
	require.Equal(t, math.NewInt(150), k.GetRewardState(dayThree).AllocatedRewards)

	end := *positionA.UnbondingEndTime
	tooEarly := dayThree.WithBlockTime(end.Add(-time.Nanosecond))
	_, err = server.Withdraw(sdk.WrapSDKContext(tooEarly), &types.MsgWithdraw{Owner: ownerA, PositionId: responseA.PositionId})
	require.ErrorIs(t, err, types.ErrNotMatured)
	mature := tooEarly.WithBlockTime(end)
	_, err = server.Withdraw(sdk.WrapSDKContext(mature), &types.MsgWithdraw{Owner: ownerA, PositionId: responseA.PositionId})
	require.NoError(t, err)
	_, found := k.GetPosition(mature, responseA.PositionId)
	require.False(t, found, "settled withdrawn positions are pruned")
	require.Equal(t, sdk.NewCoin(types.BondDenom, math.NewInt(100)), bank.principalPoolBalance)
}

func TestStreamingRepeatedClaimsPreserveActiveFractionalRewards(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(3)
	k.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt())
	state.UnallocatedRewards = math.NewInt(6)
	k.SetRewardState(ctx, state)
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(6))
	server := NewMsgServerImpl(k)
	ownerA, ownerB := testAddress(), testAddress()
	responseA, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{Owner: ownerA, Amount: sdk.NewInt64Coin(types.BondDenom, 1), LockTierId: 1})
	require.NoError(t, err)
	responseB, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{Owner: ownerB, Amount: sdk.NewInt64Coin(types.BondDenom, 1), LockTierId: 1})
	require.NoError(t, err)

	dayOne := ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))
	first, err := server.ClaimRewards(sdk.WrapSDKContext(dayOne), &types.MsgClaimRewards{Owner: ownerA, PositionId: responseA.PositionId})
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), first.Amount.Amount)
	positionA, _ := k.GetPosition(dayOne, responseA.PositionId)
	require.True(t, positionA.RewardDebt.Equal(math.LegacyOneDec()))

	dayTwo := dayOne.WithBlockTime(dayOne.BlockTime().Add(24 * time.Hour))
	second, err := server.ClaimRewards(sdk.WrapSDKContext(dayTwo), &types.MsgClaimRewards{Owner: ownerA, PositionId: responseA.PositionId})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), second.Amount.Amount)
	other, err := server.ClaimRewards(sdk.WrapSDKContext(dayTwo), &types.MsgClaimRewards{Owner: ownerB, PositionId: responseB.PositionId})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(3), other.Amount.Amount)
	require.Equal(t, math.NewInt(6), math.NewInt(1).Add(second.Amount.Amount).Add(other.Amount.Amount))
	require.True(t, k.GetRewardState(dayTwo).AllocatedRewards.IsZero())
}
