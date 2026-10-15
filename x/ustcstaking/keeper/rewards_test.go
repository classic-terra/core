package keeper

import (
	"errors"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFundRewardsWaitsInReserveUntilElapsedTime(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(1000)
	keeper.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(1000))
	keeper.SetRewardState(ctx, state)
	position := types.Position{
		Id:         1,
		Shares:     math.OneInt(),
		RewardDebt: math.LegacyZeroDec(),
		Status:     types.PositionStatus_POSITION_STATUS_ACTIVE,
	}

	require.NoError(t, keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.NewInt(1000))))
	require.True(t, keeper.GetRewardState(ctx).RewardIndex.IsZero())
	require.Equal(t, math.NewInt(1000), keeper.GetRewardState(ctx).UnallocatedRewards)
	next := ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))
	require.NoError(t, keeper.AccrueRewards(next))

	require.Equal(t, math.OneInt(), keeper.AccruedRewards(position, keeper.GetRewardState(next)))
	distributor := keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper)
	require.Equal(t, 1, distributor.calls)
	require.Equal(t, types.RewardPoolName, distributor.recipientModule)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(types.BondDenom, math.NewInt(1000))), distributor.amount)
}

func TestFundingWithNoActiveSharesPrefundsWithoutAllocation(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	require.NoError(t, keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.OneInt())))

	require.Equal(t, math.OneInt(), keeper.GetRewardState(ctx).UnallocatedRewards)
	require.True(t, keeper.GetRewardState(ctx).RewardIndex.IsZero())
	require.Equal(t, 1, keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper).calls)
}

func TestFundRewardsLeavesIndexUnchangedWhenCommunityPoolFails(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	initial := newTestRewardState(ctx, math.LegacyOneDec(), math.NewInt(100))
	keeper.SetRewardState(ctx, initial)
	keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper).err = errors.New("insufficient community pool")

	err := keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.NewInt(40)))

	require.Error(t, err)
	require.Equal(t, initial, keeper.GetRewardState(ctx))
}
