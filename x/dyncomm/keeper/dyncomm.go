package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GetVotingPower calculates the voting power of a validator in percent
func (k Keeper) CalculateVotingPower(ctx sdk.Context, validator stakingtypes.Validator) (ret sdk.Dec) {

	totalPower := k.StakingKeeper.GetLastTotalPower(ctx).Int64()
	validatorPower := sdk.TokensToConsensusPower(
		validator.Tokens,
		k.StakingKeeper.PowerReduction(ctx),
	)
	return sdk.NewDec(validatorPower).QuoInt64(totalPower).MulInt64(100)

}

// CalculateDynCommission calculates the min commission according
// to StrathColes formula
func (k Keeper) CalculateDynCommission(ctx sdk.Context, validator stakingtypes.Validator) (ret sdk.Dec) {

	// The original parameters as defined
	// by Strath
	A := k.GetMaxZero(ctx)
	B := k.GetSlopeBase(ctx)
	C := k.GetSlopeVpImpact(ctx)
	D := k.GetCap(ctx).MulInt64(100)
	x := k.CalculateVotingPower(ctx, validator)
	factorA := x.Sub(A)
	quotient := x.Quo(C)
	factorB := quotient.Add(B)
	minComm := k.StakingKeeper.MinCommissionRate(ctx)

	y := factorA.Mul(factorB)
	if y.GT(D) {
		y = D
	}
	if minComm.GT(y) {
		y = minComm
	}
	return y

}
