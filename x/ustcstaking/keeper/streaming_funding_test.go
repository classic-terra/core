package keeper

import (
	"errors"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestFundRewardsCanPrefundBeforeAnyShares(t *testing.T) {
	ctx, k := newKeeperTest(t)
	err := k.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.NewInt(1000)))
	require.NoError(t, err)
	state := k.GetRewardState(ctx)
	require.Equal(t, math.NewInt(1000), state.UnallocatedRewards)
	require.True(t, state.AllocatedRewards.IsZero())
	require.True(t, state.TotalShares.IsZero())
	require.True(t, state.RewardIndex.IsZero())
	require.Equal(t, math.NewInt(1000), k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance.Amount)
}

func TestFundRewardsSettlesOldReserveBeforeTopUp(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	k.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100))
	state.UnallocatedRewards = math.NewInt(50)
	k.SetRewardState(ctx, state)
	k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(50))

	next := ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))
	require.NoError(t, k.FundRewards(next, sdk.NewCoin(types.BondDenom, math.NewInt(200))))

	got := k.GetRewardState(next)
	require.Equal(t, math.NewInt(200), got.UnallocatedRewards)
	require.Equal(t, math.NewInt(50), got.AllocatedRewards)
	require.True(t, got.RewardIndex.Equal(math.LegacyNewDecWithPrec(5, 1)))
	require.Equal(t, math.NewInt(250), k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance.Amount)
}

func TestFundRewardsWhilePausedAndAfterEmptyInterval(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.Paused = true
	params.RewardRatePerDay = math.NewInt(100)
	k.SetParams(ctx, params)
	next := ctx.WithBlockTime(ctx.BlockTime().Add(30 * 24 * time.Hour))
	server := NewMsgServerImpl(k)
	_, err := server.FundRewards(sdk.WrapSDKContext(next), &types.MsgFundRewards{
		Authority: k.GetAuthority(), Amount: sdk.NewCoin(types.BondDenom, math.NewInt(500)),
	})
	require.NoError(t, err)
	got := k.GetRewardState(next)
	require.Equal(t, math.NewInt(500), got.UnallocatedRewards)
	require.True(t, got.AllocatedRewards.IsZero())
	require.Equal(t, math.LegacyZeroDec(), got.RewardIndex)
	require.Equal(t, next.BlockTime(), *got.LastRewardTime)
}

func TestFundRewardsFailureRollsBackProspectiveSettlement(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	k.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100))
	state.UnallocatedRewards = math.NewInt(100)
	k.SetRewardState(ctx, state)
	k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(100))
	distributor := k.communityPoolKeeper.(*recordingCommunityPoolKeeper)
	distributor.err = errors.New("community pool transfer failed")
	next := ctx.WithBlockTime(ctx.BlockTime().Add(12 * time.Hour))
	before := k.GetRewardState(ctx)

	err := k.FundRewards(next, sdk.NewCoin(types.BondDenom, math.OneInt()))

	require.Error(t, err)
	require.Equal(t, before, k.GetRewardState(ctx))
	require.Empty(t, ctx.EventManager().Events())
}

func TestUpdateParamsSettlesAtOldRateAndKeepsPausedTimeUnpaid(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	k.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100))
	state.UnallocatedRewards = math.NewInt(1000)
	k.SetRewardState(ctx, state)
	k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(1000))
	next := ctx.WithBlockTime(ctx.BlockTime().Add(12 * time.Hour))
	params.RewardRatePerDay = math.NewInt(200)
	params.Paused = true
	server := NewMsgServerImpl(k)
	_, err := server.UpdateParams(sdk.WrapSDKContext(next), &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: params})
	require.NoError(t, err)

	got := k.GetRewardState(next)
	require.Equal(t, math.NewInt(950), got.UnallocatedRewards)
	require.Equal(t, math.NewInt(50), got.AllocatedRewards)
	require.True(t, got.RewardIndex.Equal(math.LegacyNewDecWithPrec(5, 1)))
	require.True(t, got.EmissionRemainder.IsZero())
	require.True(t, k.GetParams(next).Paused)
	err = k.AccrueRewards(next.WithBlockTime(next.BlockTime().Add(48 * time.Hour)))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), k.GetRewardState(next).AllocatedRewards)
}

func TestFundRewardsRequiresGovernanceAuthorityBeforeSettlement(t *testing.T) {
	ctx, k := newKeeperTest(t)
	server := NewMsgServerImpl(k)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt())
	k.SetRewardState(ctx, state)
	next := ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	_, err := server.FundRewards(sdk.WrapSDKContext(next), &types.MsgFundRewards{
		Authority: authtypes.NewModuleAddress("not-gov").String(), Amount: sdk.NewCoin(types.BondDenom, math.OneInt()),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Equal(t, state, k.GetRewardState(ctx))
	require.Empty(t, ctx.EventManager().Events())
}
