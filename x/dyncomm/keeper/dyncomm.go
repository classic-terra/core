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
	minComm := k.StakingKeeper.MinCommissionRate(ctx).MulInt64(100)

	y := factorA.Mul(factorB)
	if y.GT(D) {
		y = D
	}
	if minComm.GT(y) {
		y = minComm
	}
	return y.QuoInt64(100)

}

func (k Keeper) UpdateValidatorRates(ctx sdk.Context, validator stakingtypes.Validator) {

	currRate := validator.Commission.Rate
	newRate := k.CalculateDynCommission(ctx, validator)
	newMaxRate := validator.Commission.MaxRate

	// we have no pain if current rate of validator
	// is GTE than dyn commission
	if currRate.GTE(newRate) {
		return
	}

	if newMaxRate.LT(newRate) {
		newMaxRate = newRate
	}

	newValidator := validator
	newValidator.Commission = stakingtypes.NewCommission(
		newRate,
		newMaxRate,
		validator.Commission.MaxChangeRate,
	)

	k.StakingKeeper.SetValidator(ctx, newValidator)

}

func (k Keeper) UpdateAllBondedValidatorRates(ctx sdk.Context) (err error) {

	var lastErr error = nil
	k.StakingKeeper.IterateValidators(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {

		val := validator.(stakingtypes.Validator)

		if !val.IsBonded() {
			return false
		}

		k.UpdateValidatorRates(ctx, val)

		return false

	})

	return lastErr

}
