package keeper

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPruneOnlyAfterWithdrawalAndFinalClaim(t *testing.T) {
	for _, reward := range []int64{0, 5} {
		t.Run(math.NewInt(reward).String(), func(t *testing.T) {
			ctx, k := newKeeperTest(t)
			owner := testAddress()
			end := ctx.BlockTime().Add(-time.Second)
			p := types.Position{Id: 1, Owner: owner, Principal: sdk.NewInt64Coin(types.BondDenom, 10), Shares: math.ZeroInt(), RewardDebt: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_UNBONDING, UnbondingEndTime: &end, ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, reward)}
			require.NoError(t, k.SetPosition(ctx, p))
			bank := k.bankKeeper.(*recordingBankKeeper)
			bank.principalPoolBalance = sdk.NewInt64Coin(types.BondDenom, 10)
			bank.rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 5)
			if reward > 0 {
				state := k.GetRewardState(ctx)
				state.AllocatedRewards = math.NewInt(reward)
				k.SetRewardState(ctx, state)
			}
			server := NewMsgServerImpl(k)
			_, err := server.Withdraw(ctx, &types.MsgWithdraw{Owner: owner, PositionId: 1})
			require.NoError(t, err)
			_, found := k.GetPosition(ctx, 1)
			require.Equal(t, reward > 0, found)
			if reward > 0 {
				_, err = server.ClaimRewards(ctx, &types.MsgClaimRewards{Owner: owner, PositionId: 1})
				require.NoError(t, err)
			}
			_, found = k.GetPosition(ctx, 1)
			require.False(t, found)
			res, err := NewQueryServerImpl(k).PositionsByOwner(ctx, &types.QueryPositionsByOwnerRequest{Owner: owner})
			require.NoError(t, err)
			require.Empty(t, res.Positions)
		})
	}
}
