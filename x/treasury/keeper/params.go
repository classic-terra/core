package keeper

import (
	sdkmath "cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TaxPolicy defines constraints for TaxRate
func (k Keeper) TaxPolicy(ctx sdk.Context) (res types.PolicyConstraints) {
	k.paramSpace.Get(ctx, types.KeyTaxPolicy, &res)
	return res
}

// RewardPolicy defines constraints for RewardWeight
func (k Keeper) RewardPolicy(ctx sdk.Context) (res types.PolicyConstraints) {
	k.paramSpace.Get(ctx, types.KeyRewardPolicy, &res)
	return res
}

// SeigniorageBurdenTarget defines fixed target for the Seigniorage Burden. Between 0 and 1.
func (k Keeper) SeigniorageBurdenTarget(ctx sdk.Context) (res sdkmath.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeySeigniorageBurdenTarget, &res)
	return res
}

// MiningIncrement is a factor used to determine how fast MRL should grow over time
func (k Keeper) MiningIncrement(ctx sdk.Context) (res sdkmath.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyMiningIncrement, &res)
	return res
}

// WindowShort is a short period window for moving average
func (k Keeper) WindowShort(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyWindowShort, &res)
	return res
}

// WindowLong is a long period window for moving average
func (k Keeper) WindowLong(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyWindowLong, &res)
	return res
}

// WindowProbation is a period of time to prevent updates
func (k Keeper) WindowProbation(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyWindowProbation, &res)
	return res
}

func (k Keeper) GetBurnSplitRate(ctx sdk.Context) (res sdkmath.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyBurnTaxSplit, &res)
	return res
}

func (k Keeper) SetBurnSplitRate(ctx sdk.Context, burnTaxSplit sdkmath.LegacyDec) {
	k.paramSpace.Set(ctx, types.KeyBurnTaxSplit, burnTaxSplit)
}

func (k Keeper) GetMinInitialDepositRatio(ctx sdk.Context) (res sdkmath.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyMinInitialDepositRatio, &res)
	return res
}

func (k Keeper) SetMinInitialDepositRatio(ctx sdk.Context, minInitialDepositRatio sdkmath.LegacyDec) {
	k.paramSpace.Set(ctx, types.KeyMinInitialDepositRatio, minInitialDepositRatio)
}

func (k Keeper) GetOracleSplitRate(ctx sdk.Context) (res sdkmath.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyOracleSplit, &res)
	return res
}

func (k Keeper) SetOracleSplitRate(ctx sdk.Context, oracleSplit sdkmath.LegacyDec) {
	k.paramSpace.Set(ctx, types.KeyOracleSplit, oracleSplit)
}

// GetParams returns the total set of treasury parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSetIfExists(ctx, &params)
	return params
}

// SetParams sets the total set of treasury parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
