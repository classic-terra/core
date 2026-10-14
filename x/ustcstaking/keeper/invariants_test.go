package keeper

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPrincipalInvariantMatchesPool(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(10)), Shares: math.NewInt(1), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}
	require.NoError(t, keeper.SetPosition(ctx, position))
	bank := keeper.bankKeeper.(*recordingBankKeeper)
	bank.principalPoolBalance = position.Principal

	message, broken := principalInvariant(keeper)(ctx)

	require.False(t, broken, message)
}

func TestPrincipalInvariantAllowsSurplus(t *testing.T) {
	ctx, k := newKeeperTest(t)
	k.bankKeeper.(*recordingBankKeeper).principalPoolBalance = sdk.NewInt64Coin(types.BondDenom, 1)
	msg, broken := principalInvariant(k)(ctx)
	require.False(t, broken, msg)
}

func TestPrincipalInvariantReportsMismatch(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(10)), Shares: math.NewInt(1), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}))
	keeper.bankKeeper.(*recordingBankKeeper).principalPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(9))

	message, broken := principalInvariant(keeper)(ctx)

	require.True(t, broken)
	require.Contains(t, message, "expected=10")
}

func TestActiveSharesInvariantMatchesRewardState(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner, Shares: math.NewInt(7), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}))
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.NewInt(7)})

	message, broken := activeSharesInvariant(keeper)(ctx)

	require.False(t, broken, message)
}

func TestActiveSharesInvariantRejectsUninitializedRewardState(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec().Neg(), TotalShares: math.ZeroInt()})

	message, broken := activeSharesInvariant(keeper)(ctx)

	require.True(t, broken)
	require.Contains(t, message, "uninitialized")
}

func TestRewardSolvencyInvariantIncludesActiveAndClaimable(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(), ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}))
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(10)})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(13))

	message, broken := rewardSolvencyInvariant(keeper)(ctx)

	require.False(t, broken, message)
}

func TestRewardSolvencyInvariantReportsShortfall(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(), ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}))
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(10)})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(9))

	message, broken := rewardSolvencyInvariant(keeper)(ctx)

	require.True(t, broken)
	require.True(t, strings.Contains(message, "expected=10"))
}

func TestRewardSolvencyInvariantIncludesActiveClaimableRewards(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(),
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.NewInt(5)),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, keeper.SetPosition(ctx, position))
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(10)})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(14))

	message, broken := rewardSolvencyInvariant(keeper)(ctx)

	require.True(t, broken)
	require.Contains(t, message, "expected=15")
}

func TestRewardSolvencyInvariantRejectsActiveRewardDebtAboveEntitlement(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	position := types.Position{
		Id: 1, Owner: testAddress(), Shares: math.OneInt(), RewardDebt: math.LegacyOneDec(),
		Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, keeper.SetPosition(ctx, position))
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.OneInt()})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.ZeroInt())

	message, broken := rewardSolvencyInvariant(keeper)(ctx)

	require.True(t, broken)
	require.Contains(t, message, "offending_position=1")
}

type invariantRegistry struct {
	routes map[string]sdk.Invariant
}

type noBalanceScanBank struct{ *recordingBankKeeper }

func (noBalanceScanBank) GetAllBalances(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	panic("unexpected denomination scan")
}

func TestValidateStateDoesNotScanForeignDenominations(t *testing.T) {
	ctx, k := newKeeperTest(t)
	k.bankKeeper = noBalanceScanBank{k.bankKeeper.(*recordingBankKeeper)}
	require.NotPanics(t, func() {
		require.NoError(t, k.ValidateState(ctx))
		_, err := NewQueryServerImpl(k).ValidateState(ctx, &types.QueryValidateStateRequest{})
		require.NoError(t, err)
	})
}

func (r *invariantRegistry) RegisterRoute(_ string, route string, invariant sdk.Invariant) {
	r.routes[route] = invariant
}

func TestRegisterInvariantsAddsAllRoutes(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	registry := &invariantRegistry{routes: map[string]sdk.Invariant{}}

	RegisterInvariants(registry, keeper)

	require.Len(t, registry.routes, 3)
	for _, route := range []string{"principal-custody", "active-shares", "reward-solvency"} {
		_, ok := registry.routes[route]
		require.True(t, ok, route)
	}
	for _, invariant := range registry.routes {
		_, _ = invariant(ctx)
	}
}

func TestValidateStateRejectsPoolMismatch(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(10)), Shares: math.NewInt(1), Status: types.PositionStatus_POSITION_STATUS_ACTIVE}))

	err := keeper.ValidateState(ctx)

	require.Error(t, err)
}
