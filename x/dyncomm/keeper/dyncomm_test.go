package keeper

import (
	"testing"
	"time"

	core "github.com/classic-terra/core/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/teststaking"
	"github.com/stretchr/testify/require"
)

func TestCalculateVotingPower(t *testing.T) {

	input := CreateTestInput(t)
	helper := teststaking.NewHelper(
		t, input.Ctx, input.StakingKeeper,
	)
	helper.Denom = core.MicroLunaDenom
	helper.CreateValidatorWithValPower(ValAddrFrom(0), PubKeys[0], 9, true)
	helper.CreateValidatorWithValPower(ValAddrFrom(1), PubKeys[1], 1, true)
	helper.TurnBlock(time.Now())
	vals := input.StakingKeeper.GetBondedValidatorsByPower(input.Ctx)

	require.Equal(
		t,
		sdk.NewDecWithPrec(90, 0),
		input.DyncommKeeper.CalculateVotingPower(input.Ctx, vals[0]),
	)

	/*val0, err := CreateValidator(0, math.NewIntFromUint64(900))
	require.NoError(t, err, "error creating validator")
	input.StakingKeeper.SetValidator(input.Ctx, val0)
	val1, err := CreateValidator(1, math.NewIntFromUint64(100))
	require.NoError(t, err, "error creating validator")

	val0.Tokens = math.OneInt().MulRaw(9)
	val1.Tokens = math.OneInt().MulRaw(1)
	input.StakingKeeper.SetValidator(input.Ctx, val1)
	input.StakingKeeper.SetValidator(input.Ctx, val0)
	input.StakingKeeper.BlockValidatorUpdates(input.Ctx)

	//CallCreateValidatorHooks(input.Ctx, input.DistrKeeper, AddrFrom(0), ValAddrFrom(0))
	//CallCreateValidatorHooks(input.Ctx, input.DistrKeeper, AddrFrom(1), ValAddrFrom(1))

	// test input should have equal power Validators
	*/

}
