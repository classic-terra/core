package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnRewardDustPreservesFrozenClaims(t *testing.T) {
	ctx, k := newKeeperTest(t)
	p := types.Position{Id: 1, Owner: testAddress(), Principal: sdk.NewInt64Coin(types.BondDenom, 0), Shares: math.ZeroInt(), RewardDebt: math.LegacyZeroDec(), ShareMultiplier: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_WITHDRAWN, ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 5)}
	require.NoError(t, k.SetPosition(ctx, p))
	k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 10)
	server := NewMsgServerImpl(k)
	_, err := server.ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, 6)})
	require.Error(t, err)
	_, err = server.ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, 5)})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5), k.bankKeeper.GetBalance(ctx, nil, types.BondDenom).Amount)
	stored, found := k.GetPosition(ctx, 1)
	require.True(t, found)
	require.Equal(t, p, stored)
}

func TestReturnRewardDustRejectsActiveSharesAndWallets(t *testing.T) {
	ctx, k := newKeeperTest(t)
	server := NewMsgServerImpl(k)
	k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 10)
	_, err := server.ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: testAddress(), Amount: sdk.NewInt64Coin(types.BondDenom, 1)})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	k.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.OneInt()})
	_, err = server.ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, 1)})
	require.Error(t, err)
}

func TestReturnRewardDustValidationAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		coin sdk.Coin
	}{
		{"wrong denom", sdk.NewInt64Coin("uluna", 1)}, {"zero", sdk.NewInt64Coin(types.BondDenom, 0)}, {"nil", sdk.Coin{Denom: types.BondDenom}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, k := newKeeperTest(t)
			_, err := NewMsgServerImpl(k).ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: tc.coin})
			require.Error(t, err)
		})
	}
	ctx, k := newKeeperTest(t)
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 10)
	pool := k.communityPoolKeeper.(*recordingCommunityPoolKeeper)
	pool.err = fmt.Errorf("injected return failure")
	before := k.GetRewardState(ctx)
	_, err := NewMsgServerImpl(k).ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, 10)})
	require.Error(t, err)
	require.Equal(t, sdk.NewInt64Coin(types.BondDenom, 10), bank.rewardPoolBalance)
	require.Equal(t, before, k.GetRewardState(ctx))
	pool.err = nil
	params := k.GetParams(ctx)
	params.Paused = true
	k.SetParams(ctx, params)
	_, err = NewMsgServerImpl(k).ReturnRewardDust(ctx, &types.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, 10)})
	require.NoError(t, err)
	require.True(t, bank.rewardPoolBalance.IsZero())
}

func TestReservedRewardsDerivedStateRebuild(t *testing.T) {
	ctx, k := newKeeperTest(t)
	p := types.Position{Id: 7, Owner: testAddress(), ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 5)}
	require.NoError(t, k.SetPosition(ctx, p))
	require.Equal(t, math.NewInt(5), k.GetReservedRewards(ctx))
	p.ClaimableRewards = sdk.NewInt64Coin(types.BondDenom, 3)
	require.NoError(t, k.SetPosition(ctx, p))
	require.Equal(t, math.NewInt(3), k.GetReservedRewards(ctx))
	k.setReservedRewards(ctx, math.ZeroInt())
	require.NoError(t, k.RebuildDerivedState(ctx))
	require.Equal(t, math.NewInt(3), k.GetReservedRewards(ctx))
	require.NoError(t, k.DeletePosition(ctx, p.Id))
	require.True(t, k.GetReservedRewards(ctx).IsZero())
}
