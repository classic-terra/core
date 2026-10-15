package keeper

import (
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestStakeRejectsFutureAggregateRewardDebtOverflow(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(4)
	k.SetParams(ctx, params)
	halfLimit := new(big.Int).Lsh(big.NewInt(1), math.MaxBitLen)
	halfLimit.Sub(halfLimit, big.NewInt(2)).Quo(halfLimit, big.NewInt(2))
	index := math.LegacyNewDecFromBigInt(halfLimit)
	state := newTestRewardState(ctx, index, math.OneInt())
	state.UnallocatedRewards = math.NewInt(4)
	k.SetRewardState(ctx, state)
	tier := params.LockTiers[0]
	require.NoError(t, k.SetPosition(ctx, types.Position{
		Id: 1, Owner: testAddress(), Principal: sdk.NewInt64Coin(types.BondDenom, 1),
		LockTierId: tier.Id, LockDuration: tier.Duration, ShareMultiplier: tier.Multiplier,
		Shares: math.OneInt(), RewardDebt: index, ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 0),
		Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}))
	k.SetNextPositionID(ctx, 2)
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.principalPoolBalance = sdk.NewInt64Coin(types.BondDenom, 1)
	bank.rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 4)
	require.NoError(t, k.ValidateState(ctx), "regression starts from valid backed state")
	_, err := NewMsgServerImpl(k).Stake(ctx, &types.MsgStake{Owner: testAddress(), Amount: sdk.NewInt64Coin(types.BondDenom, 1), LockTierId: 1})
	require.ErrorIs(t, err, types.ErrInvalidRewardState)
	require.Equal(t, state, k.GetRewardState(ctx))
	require.Equal(t, uint64(2), k.GetNextPositionID(ctx))
	require.Zero(t, bank.accountToModuleCalls, "reject before moving principal")
	require.Empty(t, ctx.EventManager().Events())
	require.NoError(t, k.AccrueRewards(ctx.WithBlockTime(ctx.BlockTime().Add(24*time.Hour))), "rejected stake cannot poison later BeginBlock")
}

func TestAccrueRewardsRejectsMissingRuntimeState(t *testing.T) {
	ctx, k := newKeeperTest(t)
	k.SetParams(ctx, types.DefaultParams())
	ctx.KVStore(k.storeKey).Delete(types.RewardStateKey)
	err := k.AccrueRewards(ctx)
	require.ErrorIs(t, err, types.ErrInvalidRewardState)
	require.Nil(t, ctx.KVStore(k.storeKey).Get(types.RewardStateKey), "corruption must not write fabricated state")
	require.Empty(t, ctx.EventManager().Events())
}
