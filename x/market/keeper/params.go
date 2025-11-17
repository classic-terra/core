package keeper

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v3/x/market/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BasePool is liquidity pool(usdr unit) which will be made available per PoolRecoveryPeriod
func (k Keeper) BasePool(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyBasePool, &res)
	return
}

// MinStabilitySpread is the minimum spread applied to swaps to / from Luna.
// Intended to prevent swing trades exploiting oracle period delays
func (k Keeper) MinStabilitySpread(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyMinStabilitySpread, &res)
	return
}

// PoolRecoveryPeriod is the period required to recover Terra&Luna Pools to the MintBasePool & BurnBasePool
func (k Keeper) PoolRecoveryPeriod(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyPoolRecoveryPeriod, &res)
	return
}

// EpochLengthBlocks is the number of blocks per market epoch (burn/refill cadence)
func (k Keeper) EpochLengthBlocks(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyEpochLengthBlocks, &res)
	return
}

// SwapFeeBurnRate returns the fraction [0,1] of the swap fee that should be burned
func (k Keeper) SwapFeeBurnRate(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeySwapFeeBurnRate, &res)
	return
}

// SwapFeeCommunityRate returns the fraction [0,1] of the swap fee that should be sent to the Community Pool
func (k Keeper) SwapFeeCommunityRate(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeySwapFeeCommunityRate, &res)
	return
}

// MaxOracleAgeSeconds returns the maximum age in seconds for oracle prices
func (k Keeper) MaxOracleAgeSeconds(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyMaxOracleAgeSeconds, &res)
	return
}

// TwapLookbackWindow returns the number of blocks for TWAP calculation
func (k Keeper) TwapLookbackWindow(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyTWAPLookbackWindow, &res)
	return
}

// MaxTwapDeviation returns the maximum deviation from TWAP
func (k Keeper) MaxTwapDeviation(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyMaxTWAPDeviation, &res)
	return
}

// DailyCapFactor returns the daily cap factor
func (k Keeper) DailyCapFactor(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyDailyCapFactor, &res)
	return
}

// GetParams returns the total set of market parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSetIfExists(ctx, &params)
	return params
}

// SetParams sets the total set of market parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
