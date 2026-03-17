package keeper

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VotePeriod returns the number of blocks during which voting takes place.
func (k Keeper) VotePeriod(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyVotePeriod, &res)
	return res
}

// VoteThreshold returns the minimum percentage of votes that must be received for a ballot to pass.
func (k Keeper) VoteThreshold(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyVoteThreshold, &res)
	return res
}

// RewardBand returns the ratio of allowable exchange rate error that a validator can be rewared
func (k Keeper) RewardBand(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyRewardBand, &res)
	return res
}

// RewardDistributionWindow returns the number of vote periods during which seigiornage reward comes in and then is distributed.
func (k Keeper) RewardDistributionWindow(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyRewardDistributionWindow, &res)
	return res
}

// Whitelist returns the denom list that can be activated
func (k Keeper) Whitelist(ctx sdk.Context) (res types.DenomList) {
	k.paramSpace.Get(ctx, types.KeyWhitelist, &res)
	return res
}

// SetWhitelist store new whitelist to param store
// this function is only for test purpose
func (k Keeper) SetWhitelist(ctx sdk.Context, whitelist types.DenomList) {
	k.paramSpace.Set(ctx, types.KeyWhitelist, whitelist)
}

// SlashFraction returns oracle voting penalty rate
func (k Keeper) SlashFraction(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeySlashFraction, &res)
	return res
}

// SlashWindow returns # of vote period for oracle slashing
func (k Keeper) SlashWindow(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeySlashWindow, &res)
	return res
}

// MinValidPerWindow returns oracle slashing threshold
func (k Keeper) MinValidPerWindow(ctx sdk.Context) (res math.LegacyDec) {
	k.paramSpace.Get(ctx, types.KeyMinValidPerWindow, &res)
	return res
}

// GetParams returns the total set of oracle parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSetIfExists(ctx, &params)
	return params
}

// SetParams sets the total set of oracle parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
