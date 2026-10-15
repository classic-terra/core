package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestMigrate1to2PreservesLegacyPositionEntitlementsAndProtectsLegacyPool(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.Int{}
	k.SetParams(ctx, params)
	paramsBytes := k.cdc.MustMarshal(&params)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, keepLegacyFields(paramsBytes, 5))
	duration := 21 * 24 * time.Hour
	position := types.Position{
		Id: 1, Owner: testAddress(), Principal: sdk.NewInt64Coin(types.BondDenom, 9),
		LockTierId: 1, LockDuration: &duration, ShareMultiplier: math.LegacyOneDec(),
		Shares: math.NewInt(3), RewardDebt: math.LegacyZeroDec(),
		ClaimableRewards: sdk.NewInt64Coin(types.BondDenom, 2),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, k.SetPosition(ctx, position))
	state := types.RewardState{RewardIndex: math.LegacyNewDecWithPrec(5, 1), TotalShares: math.NewInt(3)}
	k.SetRewardState(ctx, state)
	stateBytes := k.cdc.MustMarshal(&state)
	legacyStateBytes := keepLegacyFields(stateBytes, 2)
	ctx.KVStore(k.storeKey).Set(types.RewardStateKey, legacyStateBytes)
	require.Equal(t, legacyStateBytes, keepLegacyFields(legacyStateBytes, 2))
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.principalPoolBalance = sdk.NewInt64Coin(types.BondDenom, 9)
	bank.rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 20)
	k.SetNextPositionID(ctx, 2)

	err := NewMigrator(k).Migrate1to2(ctx)

	require.NoError(t, err)
	got := k.GetRewardState(ctx)
	require.Equal(t, math.NewInt(20), got.AllocatedRewards)
	require.True(t, got.UnallocatedRewards.IsZero())
	require.Equal(t, math.LegacyNewDecWithPrec(5, 1), got.RewardIndex)
	require.Equal(t, math.NewInt(3), got.TotalShares)
	require.Equal(t, ctx.BlockTime(), *got.LastRewardTime)
	require.True(t, got.EmissionRemainder.IsZero())
	require.True(t, got.IndexRemainder.IsZero())
	require.True(t, k.GetParams(ctx).RewardRatePerDay.IsZero())
	require.Equal(t, math.NewInt(2), k.GetReservedRewards(ctx))
	require.NoError(t, k.ValidateState(ctx))
}

func TestMigrate1to2RejectsUnderfundedLegacyLiabilitiesWithoutWriting(t *testing.T) {
	ctx, k := newKeeperTest(t)
	params := types.DefaultParams()
	params.RewardRatePerDay = math.Int{}
	k.SetParams(ctx, params)
	paramsBytes := k.cdc.MustMarshal(&params)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, keepLegacyFields(paramsBytes, 5))
	duration := 21 * 24 * time.Hour
	position := types.Position{
		Id: 1, Owner: testAddress(), Principal: sdk.NewInt64Coin(types.BondDenom, 9),
		LockTierId: 1, LockDuration: &duration, ShareMultiplier: math.LegacyOneDec(),
		Shares: math.OneInt(), RewardDebt: math.LegacyZeroDec(),
		Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	require.NoError(t, k.SetPosition(ctx, position))
	legacyState := types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.OneInt()}
	k.SetRewardState(ctx, legacyState)
	stateBytes := k.cdc.MustMarshal(&legacyState)
	ctx.KVStore(k.storeKey).Set(types.RewardStateKey, keepLegacyFields(stateBytes, 2))
	bank := k.bankKeeper.(*recordingBankKeeper)
	bank.principalPoolBalance = sdk.NewInt64Coin(types.BondDenom, 9)
	bank.rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 0)
	paramsBefore := append([]byte(nil), ctx.KVStore(k.storeKey).Get(types.ParamsKey)...)
	stateBefore := append([]byte(nil), ctx.KVStore(k.storeKey).Get(types.RewardStateKey)...)
	require.Equal(t, stateBefore, keepLegacyFields(stateBefore, 2)) // v1 state had only fields 1 and 2

	err := NewMigrator(k).Migrate1to2(ctx)

	require.ErrorIs(t, err, types.ErrRewardPoolInsolvent)
	require.Equal(t, paramsBefore, ctx.KVStore(k.storeKey).Get(types.ParamsKey))
	require.Equal(t, stateBefore, ctx.KVStore(k.storeKey).Get(types.RewardStateKey))
}

// keepLegacyFields converts the encoded current message to its exact v1 wire
// shape by removing fields introduced after the original schema.
func keepLegacyFields(encoded []byte, maximum protowire.Number) []byte {
	var legacy []byte
	for len(encoded) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(encoded)
		if tagLength < 0 {
			panic(protowire.ParseError(tagLength))
		}
		fieldLength := protowire.ConsumeFieldValue(number, wireType, encoded[tagLength:])
		if fieldLength < 0 {
			panic(protowire.ParseError(fieldLength))
		}
		totalLength := tagLength + fieldLength
		if number <= maximum {
			legacy = append(legacy, encoded[:totalLength]...)
		}
		encoded = encoded[totalLength:]
	}
	return legacy
}
