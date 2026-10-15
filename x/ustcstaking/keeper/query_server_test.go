package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestQueryPositionReturnsNotFoundForUnknownID(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	server := NewQueryServerImpl(keeper)

	_, err := server.Position(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{PositionId: 99})

	require.Error(t, err)
}

func TestQueryPositionsByOwnerIsDeterministicAndPaginated(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	other := testAddress()
	for _, position := range []types.Position{
		{Id: 3, Owner: owner, Shares: math.NewInt(30)},
		{Id: 1, Owner: owner, Shares: math.NewInt(10)},
		{Id: 2, Owner: other, Shares: math.NewInt(20)},
	} {
		keeper.SetPosition(ctx, position)
	}
	server := NewQueryServerImpl(keeper)

	response, err := server.PositionsByOwner(sdk.WrapSDKContext(ctx), &types.QueryPositionsByOwnerRequest{
		Owner: owner, Pagination: &query.PageRequest{Limit: 1},
	})

	require.NoError(t, err)
	require.Len(t, response.Positions, 1)
	require.Equal(t, uint64(1), response.Positions[0].Id)
	require.NotEmpty(t, response.Pagination.NextKey)
}

func TestQueryPositionsByOwnerSupportsOffsetPagination(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	for id := uint64(1); id <= 5; id++ {
		require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: id, Owner: owner}))
	}
	server := NewQueryServerImpl(keeper)

	response, err := server.PositionsByOwner(sdk.WrapSDKContext(ctx), &types.QueryPositionsByOwnerRequest{
		Owner: owner, Pagination: &query.PageRequest{Offset: 2, Limit: 2},
	})

	require.NoError(t, err)
	require.Len(t, response.Positions, 2)
	require.Equal(t, uint64(3), response.Positions[0].Id)
	require.Equal(t, uint64(4), response.Positions[1].Id)
	require.NotEmpty(t, response.Pagination.NextKey)
}

func TestQueryPositionsByOwnerRejectsIndexToMissingPosition(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	index := prefix.NewStore(ctx.KVStore(keeper.storeKey), types.OwnerPositionKeyPrefix)
	ownerAddress, err := sdk.AccAddressFromBech32(owner)
	require.NoError(t, err)
	prefix.NewStore(index, types.OwnerPositionPrefix(ownerAddress)).Set(sdk.Uint64ToBigEndian(99), []byte{})
	server := NewQueryServerImpl(keeper)

	_, err = server.PositionsByOwner(sdk.WrapSDKContext(ctx), &types.QueryPositionsByOwnerRequest{Owner: owner})

	require.ErrorIs(t, err, types.ErrInvalidPosition)
}

func TestQueriesDoNotMutateRewardDebt(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id: ownerPositionID, Owner: owner, Shares: math.NewInt(10),
		RewardDebt: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	keeper.SetPosition(ctx, position)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyOneDec(), math.NewInt(10)))
	server := NewQueryServerImpl(keeper)

	_, err := server.Position(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{PositionId: ownerPositionID})
	require.NoError(t, err)
	got, _ := keeper.GetPosition(ctx, ownerPositionID)
	require.True(t, got.RewardDebt.IsZero())
}

func TestQueryRewardStateAndParams(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	server := NewQueryServerImpl(keeper)
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyOneDec(), math.NewInt(7)))

	rewardState, err := server.RewardState(sdk.WrapSDKContext(ctx), &types.QueryRewardStateRequest{})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(7), rewardState.RewardState.TotalShares)

	params, err := server.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), params.Params)
}

func TestRewardStateQueryReturnsReserveBalanceAndPotentialRunway(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.NewInt(100)
	params.Paused = true
	keeper.SetParams(ctx, params)
	state := newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt())
	state.UnallocatedRewards = math.NewInt(200)
	keeper.SetRewardState(ctx, state)
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 250)
	server := NewQueryServerImpl(keeper)

	response, err := server.RewardState(sdk.WrapSDKContext(ctx), &types.QueryRewardStateRequest{})

	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), response.RewardRatePerDay)
	require.Equal(t, sdk.NewInt64Coin(types.BondDenom, 250), response.RewardPoolBalance)
	require.True(t, response.Paused)
	require.True(t, response.RunwayDefined)
	require.Equal(t, "172800", response.RunwaySeconds)
	require.Equal(t, math.NewInt(200), keeper.GetRewardState(ctx).UnallocatedRewards)
}

func TestPositionQueryIncludesWholePendingWithoutMutatingPosition(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id: 1, Owner: owner, Shares: math.OneInt(), RewardDebt: math.LegacyZeroDec(),
		ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 2), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, keeper.SetPosition(ctx, position))
	state := newTestRewardState(ctx, math.LegacyNewDecWithPrec(15, 1), math.OneInt())
	state.AllocatedRewards = math.NewInt(3)
	keeper.SetRewardState(ctx, state)
	server := NewQueryServerImpl(keeper)

	response, err := server.Position(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{PositionId: 1})

	require.NoError(t, err)
	require.Equal(t, math.NewInt(3), response.PendingRewards.Amount)
	require.True(t, response.Position.RewardDebt.IsZero())
	stored, _ := keeper.GetPosition(ctx, 1)
	require.True(t, stored.RewardDebt.IsZero())
}

func TestQueryValidateStateReportsValidAndCorruptState(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	server := NewQueryServerImpl(keeper)

	response, err := server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{Limit: 1}})
	require.NoError(t, err)
	require.True(t, response.Valid)
	require.Equal(t, "0", response.PrincipalLiability)
	require.Equal(t, "0", response.ActiveShares)
	require.Equal(t, "0", response.RewardLiability)
	require.Empty(t, response.Pagination.NextKey)

	keeper.bankKeeper.(*recordingBankKeeper).principalPoolBalance = sdk.NewCoin(types.BondDenom, math.OneInt())
	require.NoError(t, keeper.ValidateState(ctx))
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: testAddress(), Principal: sdk.NewInt64Coin(types.BondDenom, 2)}))
	require.Error(t, keeper.ValidateState(ctx))
}

func TestQueryValidateStateBoundsEachPageAndReturnsPartialLiabilities(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	for id := uint64(1); id <= 2; id++ {
		require.NoError(t, keeper.SetPosition(ctx, types.Position{
			Id: id, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(10)),
			Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(),
			ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
			Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
		}))
	}
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyZeroDec(), math.NewInt(20)))
	server := NewQueryServerImpl(keeper)

	first, err := server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{Limit: 1}})
	require.NoError(t, err)
	require.True(t, first.Valid)
	require.Equal(t, "10", first.PrincipalLiability)
	require.Equal(t, "10", first.ActiveShares)
	require.NotEmpty(t, first.Pagination.NextKey)

	second, err := server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 1}})
	require.NoError(t, err)
	require.Equal(t, "10", second.PrincipalLiability)
	require.Equal(t, "10", second.ActiveShares)
	require.Empty(t, second.Pagination.NextKey)

	_, err = server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{Limit: maxValidateStatePageSize + 1}})
	require.Error(t, err)
	_, err = server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{Offset: 1, Limit: 1}})
	require.Error(t, err)
	_, err = server.ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{Pagination: &query.PageRequest{CountTotal: true, Limit: 1}})
	require.Error(t, err)
}

func TestValidateStatePageReturnsWholeAndExactRewardLiabilities(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id: 1, Owner: owner, Principal: sdk.NewInt64Coin(types.BondDenom, 10),
		Shares: math.NewInt(3), RewardDebt: math.LegacyZeroDec(),
		ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 2), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, keeper.SetPosition(ctx, position))
	state := newTestRewardState(ctx, math.LegacyNewDecWithPrec(5, 1), math.NewInt(3))
	state.AllocatedRewards = math.NewInt(4)
	keeper.SetRewardState(ctx, state)
	response, err := NewQueryServerImpl(keeper).ValidateState(sdk.WrapSDKContext(ctx), &types.QueryValidateStateRequest{})

	require.NoError(t, err)
	require.Equal(t, "2", response.ClaimableLiability)
	require.Equal(t, "3", response.RewardLiability)
	require.Equal(t, "3.500000000000000000", response.RewardLiabilityExact)
	require.Equal(t, math.NewInt(2), response.ReservedRewards)
}

func BenchmarkPositionsByOwner100000(b *testing.B) {
	ctx, keeper := newKeeperTest(b)
	target := testAddress()
	owners := make([]string, 1000)
	owners[0] = target
	for i := 1; i < len(owners); i++ {
		owners[i] = testAddress()
	}
	for id := uint64(1); id <= 100_000; id++ {
		owner := owners[int(id%uint64(len(owners)))]
		if id%uint64(len(owners)) == 0 {
			owner = target
		}
		if err := keeper.SetPosition(ctx, types.Position{Id: id, Owner: owner}); err != nil {
			b.Fatal(err)
		}
	}
	server := NewQueryServerImpl(keeper)
	request := &types.QueryPositionsByOwnerRequest{Owner: target, Pagination: &query.PageRequest{Limit: 100}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response, err := server.PositionsByOwner(sdk.WrapSDKContext(ctx), request)
		if err != nil {
			b.Fatal(err)
		}
		if len(response.Positions) != 100 {
			b.Fatalf("expected 100 positions, got %d", len(response.Positions))
		}
	}
}

const ownerPositionID uint64 = 42
