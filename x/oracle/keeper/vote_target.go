package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsVoteTarget returns existence of a denom in the the voting target list
func (k Keeper) IsVoteTarget(ctx sdk.Context, denom string) bool {
	_, err := k.GetTobinTax(ctx, denom)
	return err == nil
}

// GetVoteTargets returns the voting target list on current vote period
func (k Keeper) GetVoteTargets(ctx sdk.Context) (voteTargets []string) {
	k.IterateTobinTaxes(ctx, func(denom string, _ math.LegacyDec) bool {
		voteTargets = append(voteTargets, denom)
		return false
	})

	return voteTargets
}
