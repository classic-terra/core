package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SlashExceedingMissCounters slashes validators whose miss count has already exceeded the
// maximum allowed threshold, meaning they cannot possibly recover to meet MinValidPerWindow
// by the end of the current SlashWindow. This allows for early slashing/jailing rather than
// waiting until the end of the SlashWindow.
func (k Keeper) SlashExceedingMissCounters(ctx sdk.Context) {
	height := ctx.BlockHeight()
	distributionHeight := height - sdk.ValidatorUpdateDelay - 1

	// slash_window / vote_period
	votePeriodsPerWindow := uint64(
		sdk.NewDec(int64(k.SlashWindow(ctx))).
			QuoInt64(int64(k.VotePeriod(ctx))).
			TruncateInt64(),
	)

	minValidPerWindow := k.MinValidPerWindow(ctx)
	slashFraction := k.SlashFraction(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	k.IterateMissCounters(ctx, func(operator sdk.ValAddress, missCounter uint64) bool {
		// Calculate valid vote rate; (votePeriodsPerWindow - missCounter) / votePeriodsPerWindow
		// This is the BEST CASE scenario assuming perfect voting for the rest of the window
		if missCounter >= votePeriodsPerWindow {
			// Already exceeded total periods - definitely slash
			missCounter = votePeriodsPerWindow
		}

		validVoteRate := sdk.NewDecFromInt(
			sdk.NewInt(int64(votePeriodsPerWindow - missCounter))).
			QuoInt64(int64(votePeriodsPerWindow))

		// If even the best case valid vote rate is below threshold, validator cannot recover
		if validVoteRate.LT(minValidPerWindow) {
			validator := k.StakingKeeper.Validator(ctx, operator)
			if validator.IsBonded() && !validator.IsJailed() {
				consAddr, err := validator.GetConsAddr()
				if err != nil {
					panic(err)
				}

				k.StakingKeeper.Slash(
					ctx, consAddr,
					distributionHeight, validator.GetConsensusPower(powerReduction), slashFraction,
				)
				k.StakingKeeper.Jail(ctx, consAddr)
			}
		}

		return false
	})
}

// SlashAndResetMissCounters slashes any operator who is over the miss threshold
// and clears all operators' miss counters to zero at the end of the SlashWindow.
func (k Keeper) SlashAndResetMissCounters(ctx sdk.Context) {
	// Slash validators who have exceeded the threshold
	k.SlashExceedingMissCounters(ctx)

	// Reset all miss counters
	k.IterateMissCounters(ctx, func(operator sdk.ValAddress, _ uint64) bool {
		k.DeleteMissCounter(ctx, operator)
		return false
	})
}
