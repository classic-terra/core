package types

import (
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const emissionRemainderDenominator int64 = int64(24 * time.Hour)

var legacyDecimalScale = new(big.Int).Exp(big.NewInt(10), big.NewInt(math.LegacyPrecision), nil)

func maxLegacyDecimalRaw() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), math.MaxBitLen)
	max.Mul(max, legacyDecimalScale)
	return max.Sub(max, big.NewInt(1))
}

func NewGenesisState(params Params, positions []Position, rewardState RewardState, nextPositionID uint64) *GenesisState {
	return &GenesisState{
		Params:         params,
		Positions:      positions,
		RewardState:    rewardState,
		NextPositionId: nextPositionID,
	}
}

func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		DefaultParams(),
		nil,
		RewardState{
			RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt(),
			UnallocatedRewards: math.ZeroInt(), AllocatedRewards: math.ZeroInt(),
			EmissionRemainder: math.ZeroInt(), IndexRemainder: math.ZeroInt(),
		},
		1,
	)
}

func ValidateGenesis(genesis *GenesisState) error {
	if genesis == nil {
		return ErrInvalidGenesis.Wrap("genesis is nil")
	}
	if err := genesis.Params.Validate(); err != nil {
		return ErrInvalidGenesis.Wrapf("params: %s", err)
	}
	state := genesis.RewardState
	if state.RewardIndex.IsNil() || !state.RewardIndex.IsInValidRange() || state.RewardIndex.IsNegative() ||
		state.TotalShares.IsNil() || state.TotalShares.IsNegative() ||
		state.UnallocatedRewards.IsNil() || state.UnallocatedRewards.IsNegative() ||
		state.AllocatedRewards.IsNil() || state.AllocatedRewards.IsNegative() ||
		state.EmissionRemainder.IsNil() || state.EmissionRemainder.IsNegative() ||
		state.IndexRemainder.IsNil() || state.IndexRemainder.IsNegative() {
		return ErrInvalidRewardState.Wrap("reward state fields must be initialized and non-negative")
	}
	if state.EmissionRemainder.GTE(math.NewInt(emissionRemainderDenominator)) {
		return ErrInvalidRewardState.Wrap("emission remainder is outside its denominator")
	}
	if state.LastRewardTime != nil {
		if err := timestamppb.New(*state.LastRewardTime).CheckValid(); err != nil {
			return ErrInvalidRewardState.Wrapf("last reward time: %s", err)
		}
	}

	activeShares := new(big.Int)
	claimableLiability := new(big.Int)
	exactLiability := new(big.Int)
	seen := make(map[uint64]struct{}, len(genesis.Positions))
	maxID := uint64(0)
	for _, position := range genesis.Positions {
		if position.Id == ^uint64(0) {
			return ErrInvalidGenesis.Wrap("maximum position id cannot be incremented")
		}
		if _, exists := seen[position.Id]; exists || position.Id == 0 {
			return ErrInvalidPosition.Wrapf("duplicate or zero position id %d", position.Id)
		}
		seen[position.Id] = struct{}{}
		if position.Id > maxID {
			maxID = position.Id
		}
		if _, err := sdk.AccAddressFromBech32(position.Owner); err != nil {
			return ErrInvalidPosition.Wrapf("position %d owner: %s", position.Id, err)
		}
		if err := validatePrincipal(position.Principal, position.Status); err != nil {
			return ErrInvalidPosition.Wrapf("position %d principal: %s", position.Id, err)
		}
		if err := validateClaimable(position.ClaimableRewards); err != nil {
			return ErrInvalidPosition.Wrapf("position %d claimable rewards: %s", position.Id, err)
		}
		claimable := position.ClaimableRewards.Amount.BigInt()
		claimableLiability.Add(claimableLiability, claimable)
		exactLiability.Add(exactLiability, new(big.Int).Mul(claimable, legacyDecimalScale))
		if position.LockTierId == 0 || position.LockDuration == nil || *position.LockDuration <= 0 ||
			position.ShareMultiplier.IsNil() || !position.ShareMultiplier.IsPositive() || !position.ShareMultiplier.IsInValidRange() {
			return ErrInvalidLockSnapshot.Wrapf("position %d has invalid lock snapshot", position.Id)
		}
		if position.Shares.IsNil() || position.RewardDebt.IsNil() || position.Shares.IsNegative() || position.RewardDebt.IsNegative() {
			return ErrInvalidPosition.Wrapf("position %d has nil or negative accounting state", position.Id)
		}
		switch position.Status {
		case PositionStatus_POSITION_STATUS_ACTIVE:
			if position.UnbondingEndTime != nil {
				return ErrInvalidPosition.Wrapf("active position %d has unbonding completion time", position.Id)
			}
			if position.Shares.IsNil() || !position.Shares.IsPositive() {
				return ErrInvalidPosition.Wrapf("active position %d must have shares", position.Id)
			}
			pendingRaw := new(big.Int).Mul(position.Shares.BigInt(), state.RewardIndex.BigInt())
			if pendingRaw.Cmp(maxLegacyDecimalRaw()) > 0 {
				return ErrInvalidPosition.Wrapf("active position %d reward entitlement exceeds SDK decimal range", position.Id)
			}
			pendingRaw.Sub(pendingRaw, position.RewardDebt.BigInt())
			if pendingRaw.Sign() < 0 {
				return ErrInvalidPosition.Wrapf("active position %d reward debt exceeds accrued rewards", position.Id)
			}
			exactLiability.Add(exactLiability, pendingRaw)
			activeShares.Add(activeShares, position.Shares.BigInt())
		case PositionStatus_POSITION_STATUS_UNBONDING:
			if !position.Shares.IsZero() || position.UnbondingEndTime == nil || position.UnbondingEndTime.IsZero() ||
				!position.RewardDebt.IsZero() {
				return ErrInvalidPosition.Wrapf("unbonding position %d has invalid shares or maturity", position.Id)
			}
		case PositionStatus_POSITION_STATUS_WITHDRAWN:
			if !position.Shares.IsZero() || !position.Principal.Amount.IsZero() || !position.RewardDebt.IsZero() {
				return ErrInvalidPosition.Wrapf("withdrawn position %d retains principal or shares", position.Id)
			}
		default:
			return ErrInvalidPosition.Wrapf("position %d has unknown status", position.Id)
		}
	}
	if activeShares.BitLen() > math.MaxBitLen || activeShares.Cmp(state.TotalShares.BigInt()) != 0 {
		return ErrInvalidRewardState.Wrapf("total shares %s does not equal active shares %s", genesis.RewardState.TotalShares, activeShares)
	}
	if state.TotalShares.IsZero() && !state.IndexRemainder.IsZero() ||
		state.TotalShares.IsPositive() && state.IndexRemainder.GTE(state.TotalShares) {
		return ErrInvalidRewardState.Wrap("index remainder is outside its share denominator")
	}
	allocatedExact := new(big.Int).Mul(state.AllocatedRewards.BigInt(), legacyDecimalScale)
	if claimableLiability.BitLen() > math.MaxBitLen || state.AllocatedRewards.BigInt().Cmp(claimableLiability) < 0 || allocatedExact.Cmp(exactLiability) < 0 {
		return ErrInvalidRewardState.Wrap("allocated rewards do not cover position liabilities")
	}
	if state.LastRewardTime == nil {
		if len(genesis.Positions) != 0 || !state.RewardIndex.IsZero() || !state.TotalShares.IsZero() ||
			!state.UnallocatedRewards.IsZero() || !state.AllocatedRewards.IsZero() || claimableLiability.Sign() != 0 ||
			!state.EmissionRemainder.IsZero() || !state.IndexRemainder.IsZero() || genesis.NextPositionId != 1 ||
			!genesis.Params.Equal(DefaultParams()) {
			return ErrInvalidRewardState.Wrap("last reward time is required outside the empty default genesis")
		}
	}
	if genesis.NextPositionId <= maxID {
		return ErrInvalidGenesis.Wrapf("next position id %d must exceed max position id %d", genesis.NextPositionId, maxID)
	}
	return nil
}

func validatePrincipal(coin sdk.Coin, status PositionStatus) error {
	if coin.Denom != BondDenom || coin.Amount.IsNil() {
		return ErrInvalidDenom.Wrapf("expected %s principal coin", BondDenom)
	}
	if status == PositionStatus_POSITION_STATUS_WITHDRAWN {
		if !coin.Amount.IsZero() {
			return ErrInvalidDenom.Wrapf("expected zero %s principal for withdrawn position", BondDenom)
		}
		return nil
	}
	if !coin.Amount.IsPositive() {
		return ErrInvalidDenom.Wrapf("expected positive %s coin", BondDenom)
	}
	return nil
}

func validateClaimable(coin sdk.Coin) error {
	if coin.Denom != BondDenom || coin.Amount.IsNil() || coin.Amount.IsNegative() {
		return ErrInvalidDenom.Wrapf("expected non-negative %s claimable rewards", BondDenom)
	}
	return nil
}
