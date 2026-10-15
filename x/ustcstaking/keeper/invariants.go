package keeper

import (
	"fmt"
	"math/big"

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
		expected := new(big.Int)
		var offending uint64
		state := keeper.GetRewardState(ctx)
		if err := validateStreamingScalars(state, keeper.GetParams(ctx)); err != nil || state.LastRewardTime == nil {
			return fmt.Sprintf("active shares: uninitialized or invalid reward state index=%s total_shares=%s", state.RewardIndex, state.TotalShares), true
		}
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
				if position.Shares.IsNil() || !position.Shares.IsPositive() {
					offending = position.Id
					return true
				}
				expected.Add(expected, position.Shares.BigInt())
			} else if position.Shares.IsNil() || position.Shares.IsPositive() {
				offending = position.Id
				return true
			}
			return false
		})
		actual := state.TotalShares
		if expected.BitLen() > math.MaxBitLen || offending != 0 || actual.BigInt().Cmp(expected) != 0 {
			return fmt.Sprintf("active shares: expected=%s actual=%s offending_position=%d", expected, actual, offending), true
		}
		return "", false
	}
}

func rewardSolvencyInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		state := keeper.GetRewardState(ctx)
		if err := validateStreamingScalars(state, keeper.GetParams(ctx)); err != nil || state.LastRewardTime == nil {
			return fmt.Sprintf("reward solvency: invalid streaming state: %v", err), true
		}
		liability := new(big.Int)
		exactLiability := new(big.Int)
		claimableTotal := new(big.Int)
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			if position.ClaimableRewards.Denom != types.BondDenom || position.ClaimableRewards.Amount.IsNil() ||
				position.ClaimableRewards.Amount.IsNegative() {
				offending = position.Id
				return true
			}
			claimable := position.ClaimableRewards.Amount.BigInt()
			claimableTotal.Add(claimableTotal, claimable)
			liability.Add(liability, claimable)
			exactLiability.Add(exactLiability, new(big.Int).Mul(claimable, rewardIndexScaleBig))
			if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
				if position.Shares.IsNil() || position.RewardDebt.IsNil() || state.RewardIndex.IsNil() {
					offending = position.Id
					return true
				}
				if !shareIndexProductInRange(position.Shares, state.RewardIndex) {
					offending = position.Id
					return true
				}
				rawPending := new(big.Int).Mul(position.Shares.BigInt(), state.RewardIndex.BigInt())
				rawPending.Sub(rawPending, position.RewardDebt.BigInt())
				if rawPending.Sign() < 0 {
					offending = position.Id
					return true
				}
				exactLiability.Add(exactLiability, rawPending)
				wholePending := new(big.Int).Quo(new(big.Int).Set(rawPending), rewardIndexScaleBig)
				liability.Add(liability, wholePending)
			}
			if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE &&
				position.Status != types.PositionStatus_POSITION_STATUS_UNBONDING &&
				position.Status != types.PositionStatus_POSITION_STATUS_WITHDRAWN {
				offending = position.Id
				return true
			}
			return false
		})
		actual, valid := poolBalance(ctx, keeper, types.RewardPoolName)
		reserved, reserveErr := keeper.GetReservedRewardsChecked(ctx)
		if reserveErr != nil {
			return fmt.Sprintf("reward solvency: %v", reserveErr), true
		}
		protected := new(big.Int).Add(state.UnallocatedRewards.BigInt(), state.AllocatedRewards.BigInt())
		allocatedWhole := state.AllocatedRewards.BigInt()
		allocatedExact := new(big.Int).Mul(allocatedWhole, rewardIndexScaleBig)
		if offending != 0 || !valid || reserveErr != nil || reserved.IsNil() || liability.BitLen() > math.MaxBitLen || claimableTotal.BitLen() > math.MaxBitLen ||
			reserved.BigInt().Cmp(claimableTotal) != 0 || allocatedWhole.Cmp(liability) < 0 || allocatedExact.Cmp(exactLiability) < 0 ||
			actual.BigInt().Cmp(protected) < 0 {
			return fmt.Sprintf("reward solvency: liability=%s exact=%s claimable=%s allocated=%s protected=%s actual=%s offending_position=%d",
				liability, exactLiability, claimableTotal, state.AllocatedRewards, protected, actual, offending), true
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
