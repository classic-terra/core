package keeper

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	core "github.com/classic-terra/core/v3/types"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
	"github.com/stretchr/testify/require"
)

func TestCalculateVotingPower(t *testing.T) {
	input := CreateTestInput(t)
	helper := testutil.NewHelper(
		t, input.Ctx, input.StakingKeeper,
	)
	helper.Denom = core.MicroLunaDenom
	helper.CreateValidatorWithValPower(ValAddrFrom(0), PubKeys[0], 9, true)
	helper.CreateValidatorWithValPower(ValAddrFrom(1), PubKeys[1], 1, true)
	helper.TurnBlock(time.Now())
	vals, err := input.StakingKeeper.GetBondedValidatorsByPower(input.Ctx)
	require.NoError(t, err)

	require.Equal(
		t,
		sdkmath.LegacyNewDecWithPrec(90, 0),
		input.DyncommKeeper.CalculateVotingPower(input.Ctx, vals[0]),
	)
}

func TestCalculateDynCommission(t *testing.T) {
	input := CreateTestInput(t)
	helper := testutil.NewHelper(
		t, input.Ctx, input.StakingKeeper,
	)
	helper.Denom = core.MicroLunaDenom
	helper.CreateValidatorWithValPower(ValAddrFrom(0), PubKeys[0], 950, true)
	helper.CreateValidatorWithValPower(ValAddrFrom(1), PubKeys[1], 46, true)
	helper.CreateValidatorWithValPower(ValAddrFrom(2), PubKeys[2], 4, true)
	helper.TurnBlock(time.Now())
	vals, err := input.StakingKeeper.GetBondedValidatorsByPower(input.Ctx)
	require.NoError(t, err)

	// capped commission
	require.Equal(
		t,
		sdkmath.LegacyNewDecWithPrec(20, 2),
		input.DyncommKeeper.CalculateDynCommission(input.Ctx, vals[0]),
	)

	// curve
	require.Equal(
		t,
		sdkmath.LegacyNewDecWithPrec(10086, 5),
		input.DyncommKeeper.CalculateDynCommission(input.Ctx, vals[1]),
	)

	// min. commission
	require.Equal(
		t,
		sdkmath.LegacyZeroDec(),
		input.DyncommKeeper.CalculateDynCommission(input.Ctx, vals[2]),
	)
}
