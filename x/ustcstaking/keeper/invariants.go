package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) ValidateState(ctx sdk.Context) error {
	for _, invariant := range []sdk.Invariant{
		principalInvariant(k),
		activeSharesInvariant(k),
		rewardSolvencyInvariant(k),
	} {
		message, broken := invariant(ctx)
		if broken {
			return types.ErrInvalidRewardState.Wrap(message)
		}
	}
	return nil
}

func RegisterInvariants(registry sdk.InvariantRegistry, keeper Keeper) {
	registry.RegisterRoute(types.ModuleName, "principal-custody", principalInvariant(keeper))
	registry.RegisterRoute(types.ModuleName, "active-shares", activeSharesInvariant(keeper))
	registry.RegisterRoute(types.ModuleName, "reward-solvency", rewardSolvencyInvariant(keeper))
}

func principalInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		expected := math.ZeroInt()
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			if position.Status == types.PositionStatus_POSITION_STATUS_WITHDRAWN {
				if !position.Principal.Amount.IsNil() && position.Principal.Amount.IsPositive() {
					offending = position.Id
					return true
				}
				return false
			}
			if !position.Principal.Amount.IsNil() {
				expected = expected.Add(position.Principal.Amount)
			}
			return false
		})
		actual, valid := poolBalance(ctx, keeper, types.PrincipalPoolName)
		if offending != 0 || !valid || actual.LT(expected) {
			return fmt.Sprintf("principal custody: expected=%s actual=%s offending_position=%d", expected, actual, offending), true
		}
		return "", false
	}
}

func activeSharesInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		expected := math.ZeroInt()
		var offending uint64
		state := keeper.GetRewardState(ctx)
		if state.RewardIndex.IsNil() || state.RewardIndex.IsNegative() || state.TotalShares.IsNil() || state.TotalShares.IsNegative() {
			return fmt.Sprintf("active shares: uninitialized or negative reward state index=%s total_shares=%s", state.RewardIndex, state.TotalShares), true
		}
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			shares := position.Shares
			if shares.IsNil() {
				shares = math.ZeroInt()
			}
			if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
				expected = expected.Add(shares)
			} else if shares.IsPositive() {
				offending = position.Id
				return true
			}
			return false
		})
		actual := state.TotalShares
		if offending != 0 || actual.IsNegative() || !actual.Equal(expected) {
			return fmt.Sprintf("active shares: expected=%s actual=%s offending_position=%d", expected, actual, offending), true
		}
		return "", false
	}
}

func rewardSolvencyInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		state := keeper.GetRewardState(ctx)
		liability := math.ZeroInt()
		reserved := math.ZeroInt()
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			if position.ClaimableRewards.Denom != types.BondDenom || position.ClaimableRewards.Amount.IsNil() ||
				position.ClaimableRewards.Amount.IsNegative() {
				offending = position.Id
				return true
			}
			amount := keeper.AccruedRewards(position, state)
			reserved = reserved.Add(position.ClaimableRewards.Amount)
			if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
				if position.Shares.IsNil() || position.RewardDebt.IsNil() || state.RewardIndex.IsNil() {
					offending = position.Id
					return true
				}
				if position.Shares.ToLegacyDec().Mul(state.RewardIndex).Sub(position.RewardDebt).IsNegative() {
					offending = position.Id
					return true
				}
				amount = amount.Add(position.ClaimableRewards.Amount)
			}
			if amount.IsNil() || amount.IsNegative() {
				offending = position.Id
				return true
			}
			liability = liability.Add(amount)
			return false
		})
		actual, valid := poolBalance(ctx, keeper, types.RewardPoolName)
		if offending != 0 || !valid || actual.LT(liability) || !reserved.Equal(keeper.GetReservedRewards(ctx)) {
			return fmt.Sprintf("reward solvency: expected=%s actual=%s offending_position=%d", liability, actual, offending), true
		}
		return "", false
	}
}

func poolBalance(ctx sdk.Context, keeper Keeper, moduleName string) (math.Int, bool) {
	coin := keeper.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(moduleName), types.BondDenom)
	if coin.Amount.IsNil() || coin.Amount.IsNegative() {
		return math.ZeroInt(), false
	}
	return coin.Amount, true
}
