package keeper

import (
	"math/big"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// Migrator upgrades the legacy reward pool while preserving existing claims.
type Migrator struct{ keeper Keeper }

func NewMigrator(k Keeper) Migrator { return Migrator{keeper: k} }

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	ctx, write := ctx.CacheContext()
	k := m.keeper
	params := k.GetParams(ctx)
	params.Authority = k.GetAuthority()
	if params.RewardRatePerDay.IsNil() {
		params.RewardRatePerDay = math.ZeroInt()
	}
	if err := params.Validate(); err != nil {
		return types.ErrInvalidGenesis.Wrapf("legacy params: %s", err)
	}
	state := k.GetRewardState(ctx)
	if state.RewardIndex.IsNil() || !state.RewardIndex.IsInValidRange() || state.RewardIndex.IsNegative() ||
		state.TotalShares.IsNil() || state.TotalShares.IsNegative() {
		return types.ErrInvalidRewardState.Wrap("legacy reward index or total shares is invalid")
	}
	positions := make([]types.Position, 0)
	activeShares, principalLiability, exactLiability := new(big.Int), new(big.Int), new(big.Int)
	var maxID uint64
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	iter := store.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		if len(iter.Key()) != 8 {
			iter.Close()
			return types.ErrInvalidPosition.Wrap("legacy position key must be 8 bytes")
		}
		var p types.Position
		if err := k.cdc.Unmarshal(iter.Value(), &p); err != nil {
			iter.Close()
			return types.ErrInvalidPosition.Wrapf("decode legacy position: %s", err)
		}
		if p.Id > maxID {
			maxID = p.Id
		}
		if p.ClaimableRewards.Amount.IsNil() || p.ClaimableRewards.Amount.IsNegative() ||
			p.Shares.IsNil() || p.Shares.IsNegative() || p.RewardDebt.IsNil() || p.RewardDebt.IsNegative() {
			iter.Close()
			return types.ErrInvalidPosition.Wrapf("position %d has invalid legacy accounting", p.Id)
		}
		if p.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
			if !shareIndexProductInRange(p.Shares, state.RewardIndex) {
				iter.Close()
				return types.ErrInvalidPosition.Wrapf("position %d entitlement exceeds decimal range", p.Id)
			}
			pending := new(big.Int).Mul(p.Shares.BigInt(), state.RewardIndex.BigInt())
			pending.Sub(pending, p.RewardDebt.BigInt())
			if pending.Sign() < 0 {
				iter.Close()
				return types.ErrInvalidPosition.Wrapf("position %d reward debt exceeds entitlement", p.Id)
			}
			activeShares.Add(activeShares, p.Shares.BigInt())
			exactLiability.Add(exactLiability, pending)
		} else if p.Status != types.PositionStatus_POSITION_STATUS_UNBONDING && p.Status != types.PositionStatus_POSITION_STATUS_WITHDRAWN {
			iter.Close()
			return types.ErrInvalidPosition.Wrapf("position %d has unknown status", p.Id)
		}
		if p.Status != types.PositionStatus_POSITION_STATUS_WITHDRAWN {
			if p.Principal.Denom != types.BondDenom || p.Principal.Amount.IsNil() || p.Principal.Amount.IsNegative() {
				iter.Close()
				return types.ErrInvalidPosition.Wrapf("position %d has invalid principal", p.Id)
			}
			principalLiability.Add(principalLiability, p.Principal.Amount.BigInt())
		}
		exactLiability.Add(exactLiability, new(big.Int).Mul(p.ClaimableRewards.Amount.BigInt(), rewardIndexScaleBig))
		positions = append(positions, p)
	}
	iter.Close()
	if activeShares.Cmp(state.TotalShares.BigInt()) != 0 {
		return types.ErrInvalidRewardState.Wrap("legacy total shares do not match active positions")
	}
	if principalLiability.BitLen() > math.MaxBitLen {
		return types.ErrInvalidPosition.Wrap("legacy principal liability exceeds SDK integer range")
	}
	principalPool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.PrincipalPoolName), types.BondDenom)
	if principalPool.Amount.IsNil() || principalPool.Amount.BigInt().Cmp(principalLiability) < 0 {
		return types.ErrInvalidGenesis.Wrap("legacy principal pool cannot cover position principal")
	}
	rewardPool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if rewardPool.Amount.IsNil() || rewardPool.Amount.IsNegative() {
		return types.ErrRewardPoolInsolvent
	}
	minimumBacking := new(big.Int).Quo(new(big.Int).Set(exactLiability), rewardIndexScaleBig)
	if new(big.Int).Mod(exactLiability, rewardIndexScaleBig).Sign() != 0 {
		minimumBacking.Add(minimumBacking, big.NewInt(1))
	}
	if minimumBacking.Cmp(rewardPool.Amount.BigInt()) > 0 {
		return types.ErrRewardPoolInsolvent.Wrap("legacy pool cannot cover existing position liabilities")
	}
	// Existing bank funds were not designated for the new streaming schedule.
	// Protect the entire legacy balance; only a later governance funding call
	// enters U and becomes subject to the configured daily release rate.
	state.AllocatedRewards = rewardPool.Amount
	state.UnallocatedRewards = math.ZeroInt()
	state.EmissionRemainder, state.IndexRemainder = math.ZeroInt(), math.ZeroInt()
	lastRewardTime := ctx.BlockTime().UTC()
	state.LastRewardTime = &lastRewardTime
	nextPositionID := k.GetNextPositionID(ctx)
	if nextPositionID <= maxID {
		if maxID == ^uint64(0) {
			return types.ErrPositionIDExhausted
		}
		nextPositionID = maxID + 1
	}
	genesis := types.NewGenesisState(params, positions, state, nextPositionID)
	if err := types.ValidateGenesis(genesis); err != nil {
		return types.ErrInvalidGenesis.Wrapf("streaming migration state: %s", err)
	}
	k.SetParams(ctx, params)
	k.SetRewardState(ctx, state)
	k.SetNextPositionID(ctx, nextPositionID)
	if err := k.RebuildDerivedState(ctx); err != nil {
		return types.ErrInvalidGenesis.Wrapf("rebuild derived position state: %s", err)
	}
	write()
	return nil
}
