package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	secp256k1 "github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestStakeTransfersOnlyUSTCToPrincipalPool(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	server := NewMsgServerImpl(keeper)

	response, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(100)), LockTierId: 1,
	})

	require.NoError(t, err)
	require.Equal(t, uint64(1), response.PositionId)
	position, found := keeper.GetPosition(ctx, response.PositionId)
	require.True(t, found)
	require.Equal(t, math.NewInt(150), position.Shares)
	require.Equal(t, types.PositionStatus_POSITION_STATUS_ACTIVE, position.Status)

	bank := keeper.bankKeeper.(*recordingBankKeeper)
	require.Equal(t, types.PrincipalPoolName, bank.lastRecipientModule)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(types.BondDenom, math.NewInt(100))), bank.lastAccountToModuleAmount)
}

func TestStakeEmitsModuleEvent(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err := NewMsgServerImpl(keeper).Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(100)), LockTierId: 1,
	})

	require.NoError(t, err)
	event := findModuleEvent(t, ctx, "ustcstaking_stake")
	require.Equal(t, owner, eventAttribute(event, "owner"))
	require.Equal(t, "100uusd", eventAttribute(event, "amount"))
}

func TestRejectedStakeEmitsNoModuleEvent(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err := NewMsgServerImpl(keeper).Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin("uluna", math.NewInt(100)), LockTierId: 1,
	})

	require.Error(t, err)
	for _, event := range ctx.EventManager().Events() {
		require.NotEqual(t, "ustcstaking_stake", event.Type)
	}
}

func findModuleEvent(t *testing.T, ctx sdk.Context, eventType string) sdk.Event {
	t.Helper()
	for _, event := range ctx.EventManager().Events() {
		if event.Type == eventType {
			return event
		}
	}
	t.Fatalf("missing event %s", eventType)
	return sdk.Event{}
}

func eventAttribute(event sdk.Event, key string) string {
	for _, attribute := range event.Attributes {
		if attribute.Key == key {
			return attribute.Value
		}
	}
	return ""
}

func TestStakeRejectsPositionIDExhaustionBeforeTransfer(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	keeper.SetNextPositionID(ctx, ^uint64(0))
	server := NewMsgServerImpl(keeper)
	bank := keeper.bankKeeper.(*recordingBankKeeper)

	_, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(100)), LockTierId: 1,
	})

	require.ErrorIs(t, err, types.ErrPositionIDExhausted)
	require.Zero(t, bank.accountToModuleCalls)
}

func TestBeginUnbondingSettlesRewardsAndStopsFutureAccrual(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	duration := 24 * time.Hour
	state := newTestRewardState(ctx, math.LegacyNewDecWithPrec(5, 1), math.NewInt(100))
	state.AllocatedRewards = math.NewInt(50)
	keeper.SetRewardState(ctx, state)
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(50))
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(100)),
		Shares: math.NewInt(100), RewardDebt: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
		LockDuration: &duration,
	})
	server := NewMsgServerImpl(keeper)

	_, err := server.BeginUnbonding(sdk.WrapSDKContext(ctx), &types.MsgBeginUnbonding{Owner: owner, PositionId: 1})
	require.NoError(t, err)

	position, _ := keeper.GetPosition(ctx, 1)
	require.True(t, position.Shares.IsZero())
	require.Equal(t, math.NewInt(50), position.ClaimableRewards.Amount)
	require.Equal(t, types.PositionStatus_POSITION_STATUS_UNBONDING, position.Status)
	require.Equal(t, math.ZeroInt(), keeper.GetRewardState(ctx).TotalShares)

	state = keeper.GetRewardState(ctx)
	state.RewardIndex = math.LegacyNewDec(10)
	require.Equal(t, math.NewInt(50), keeper.AccruedRewards(position, state))
}

func TestWithdrawBeforeCompletionFails(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	end := ctx.BlockTime().Add(time.Hour)
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(100)),
		Status: types.PositionStatus_POSITION_STATUS_UNBONDING, UnbondingEndTime: &end,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
	})
	server := NewMsgServerImpl(keeper)

	_, err := server.Withdraw(sdk.WrapSDKContext(ctx), &types.MsgWithdraw{Owner: owner, PositionId: 1})
	require.ErrorIs(t, err, types.ErrNotMatured)
}

func TestClaimFailsWhenRewardPoolCannotCoverLiability(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyOneDec(), math.NewInt(10)))
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(),
		Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(5))
	server := NewMsgServerImpl(keeper)

	_, err := server.ClaimRewards(sdk.WrapSDKContext(ctx), &types.MsgClaimRewards{Owner: owner, PositionId: 1})
	require.ErrorIs(t, err, types.ErrRewardPoolInsolvent)
	require.Zero(t, keeper.bankKeeper.(*recordingBankKeeper).moduleToAccountCalls)
}

func TestClaimPaysStoredAndNewlyAccruedRewardsForActivePosition(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	state := newTestRewardState(ctx, math.LegacyOneDec(), math.NewInt(10))
	state.AllocatedRewards = math.NewInt(12)
	keeper.SetRewardState(ctx, state)
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.NewInt(5).ToLegacyDec(),
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.NewInt(7)),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
	})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(12))

	response, err := NewMsgServerImpl(keeper).ClaimRewards(sdk.WrapSDKContext(ctx), &types.MsgClaimRewards{Owner: owner, PositionId: 1})

	require.NoError(t, err)
	require.Equal(t, math.NewInt(12), response.Amount.Amount)
	position, found := keeper.GetPosition(ctx, 1)
	require.True(t, found)
	require.True(t, position.ClaimableRewards.Amount.IsZero())
	require.True(t, position.RewardDebt.Equal(math.LegacyNewDec(10)))
	require.Equal(t, 1, keeper.bankKeeper.(*recordingBankKeeper).moduleToAccountCalls)
}

func TestFundRewardsRequiresGovernanceAuthority(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	authority := testAddress()
	configureKeeper(t, ctx, keeper, authority)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100)))

	_, err := NewMsgServerImpl(keeper).FundRewards(sdk.WrapSDKContext(ctx), &types.MsgFundRewards{
		Authority: testAddress(), Amount: sdk.NewCoin(types.BondDenom, math.NewInt(40)),
	})

	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Zero(t, keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper).calls)
}

func TestFundRewardsEventIdentifiesCommunityPool(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	authority := keeper.GetAuthority()
	configureKeeper(t, ctx, keeper, authority)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(100)))
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err := NewMsgServerImpl(keeper).FundRewards(sdk.WrapSDKContext(ctx), &types.MsgFundRewards{
		Authority: authority, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(40)),
	})

	require.NoError(t, err)
	event := findModuleEvent(t, ctx, types.EventTypeFundRewards)
	require.Equal(t, authority, eventAttribute(event, "authority"))
	require.Equal(t, "community_pool", eventAttribute(event, "source"))
	require.Equal(t, "40uusd", eventAttribute(event, "amount"))
	require.Equal(t, "0.000000000000000000", eventAttribute(event, "reward_index_before"))
	require.Equal(t, "0.000000000000000000", eventAttribute(event, "reward_index_after"))
	require.Equal(t, "40", eventAttribute(event, "unallocated_rewards"))
	require.Equal(t, "0uusd", eventAttribute(event, "reward_pool_balance_before"))
	require.Equal(t, "40uusd", eventAttribute(event, "reward_pool_balance_after"))
}

func configureKeeper(t *testing.T, ctx sdk.Context, keeper Keeper, authority string) {
	t.Helper()
	duration := 24 * time.Hour
	params := types.DefaultParams()
	params.LockTiers = []types.LockTier{{Id: 1, Duration: &duration, Multiplier: math.LegacyNewDecWithPrec(15, 1)}}
	params.Authority = authority
	keeper.SetParams(ctx, params)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt()))
}

func testAddress() string {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
}
