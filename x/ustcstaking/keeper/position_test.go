package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSetPositionCreatesOwnerIndex(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()

	err := keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner})

	require.NoError(t, err)
	index := prefix.NewStore(ctx.KVStore(keeper.storeKey), types.OwnerPositionKeyPrefix)
	ownerStore := prefix.NewStore(index, types.OwnerPositionPrefix(mustAccAddress(t, owner)))
	require.NotNil(t, ownerStore.Get(sdk.Uint64ToBigEndian(1)))
}

func TestSetPositionUpdateDoesNotDuplicateOwnerIndex(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{Id: 1, Owner: owner}
	require.NoError(t, keeper.SetPosition(ctx, position))

	position.Shares = math.NewInt(7)
	require.NoError(t, keeper.SetPosition(ctx, position))

	index := prefix.NewStore(ctx.KVStore(keeper.storeKey), types.OwnerPositionKeyPrefix)
	ownerStore := prefix.NewStore(index, types.OwnerPositionPrefix(mustAccAddress(t, owner)))
	iterator := ownerStore.Iterator(nil, nil)
	defer iterator.Close()
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	require.Equal(t, 1, count)
}

func TestSetPositionRejectsOwnerChange(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	other := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner}))

	err := keeper.SetPosition(ctx, types.Position{Id: 1, Owner: other})

	require.Error(t, err)
	position, found := keeper.GetPosition(ctx, 1)
	require.True(t, found)
	require.Equal(t, owner, position.Owner)
}

func TestDeletePositionRemovesOwnerIndex(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	require.NoError(t, keeper.SetPosition(ctx, types.Position{Id: 1, Owner: owner}))

	require.NoError(t, keeper.DeletePosition(ctx, 1))
	index := prefix.NewStore(ctx.KVStore(keeper.storeKey), types.OwnerPositionKeyPrefix)
	ownerStore := prefix.NewStore(index, types.OwnerPositionPrefix(mustAccAddress(t, owner)))
	require.Nil(t, ownerStore.Get(sdk.Uint64ToBigEndian(1)))
}

func TestOwnerPositionKeyRoundTrip(t *testing.T) {
	owner := mustAccAddress(t, testAddress())
	key := types.OwnerPositionKey(owner, 42)

	decodedOwner, decodedID, err := types.OwnerFromPositionKey(key)

	require.NoError(t, err)
	require.Equal(t, owner, decodedOwner)
	require.Equal(t, uint64(42), decodedID)
	_, _, err = types.OwnerFromPositionKey([]byte{1, 2})
	require.Error(t, err)
}

func mustAccAddress(t *testing.T, value string) sdk.AccAddress {
	t.Helper()
	address, err := sdk.AccAddressFromBech32(value)
	require.NoError(t, err)
	return address
}

func TestPositionRoundTripPreservesTierSnapshot(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id:               7,
		Owner:            owner,
		Principal:        sdk.NewCoin(types.BondDenom, math.ZeroInt()),
		Shares:           math.NewInt(150),
		ShareMultiplier:  math.LegacyNewDecWithPrec(15, 1),
		RewardDebt:       math.LegacyZeroDec(),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
	}

	require.NoError(t, keeper.SetPosition(ctx, position))
	got, found := keeper.GetPosition(ctx, position.Id)

	require.True(t, found)
	require.Equal(t, position, got)
}

func TestAccruedRewardsNeverReturnsNegative(t *testing.T) {
	position := types.Position{
		Shares:     math.NewInt(10),
		RewardDebt: math.LegacyNewDec(20),
		Status:     types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	state := types.RewardState{
		RewardIndex: math.LegacyOneDec(),
		TotalShares: math.NewInt(10),
	}

	require.True(t, keeperAccruedRewards(position, state).IsZero())
}
