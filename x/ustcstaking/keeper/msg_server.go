package keeper

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type msgServer struct{ Keeper }

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Stake(goCtx context.Context, msg *types.MsgStake) (*types.MsgStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.Paused {
		return nil, types.ErrModulePaused
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	tier, found := findLockTier(params, msg.LockTierId)
	if !found {
		return nil, types.ErrInvalidLockTier.Wrapf("tier %d does not exist", msg.LockTierId)
	}
	shares, err := calculateShares(msg.Amount.Amount, tier.Multiplier)
	if err != nil {
		return nil, err
	}
	if !shares.IsPositive() {
		return nil, types.ErrInvalidLockTier.Wrap("amount and multiplier produce zero shares")
	}
	positionID := k.GetNextPositionID(ctx)
	if positionID == ^uint64(0) {
		return nil, types.ErrPositionIDExhausted
	}
	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	state := k.GetRewardState(cacheCtx)
	newTotalShares, err := checkedRewardAdd(state.TotalShares, shares)
	if err != nil {
		return nil, err
	}
	// Admission must protect future accrual for the post-stake cohort, not
	// merely the existing shares or the current index.
	projected := state
	projected.TotalShares = newTotalShares
	if err := validateProjectedRewardIndex(projected, math.ZeroInt()); err != nil {
		return nil, err
	}
	if !shareIndexProductInRange(newTotalShares, state.RewardIndex) {
		return nil, types.ErrInvalidRewardState.Wrap("stake would make reward debt exceed SDK decimal range")
	}
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(cacheCtx, owner, types.PrincipalPoolName, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}

	debt, err := sharesRewardDebt(shares, state.RewardIndex)
	if err != nil {
		return nil, err
	}
	position := types.Position{
		Id:               positionID,
		Owner:            msg.Owner,
		Principal:        msg.Amount,
		Shares:           shares,
		LockTierId:       msg.LockTierId,
		ShareMultiplier:  tier.Multiplier,
		RewardDebt:       debt,
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
		LockDuration:     tier.Duration,
	}
	if err := k.SetPosition(cacheCtx, position); err != nil {
		return nil, err
	}
	k.SetNextPositionID(cacheCtx, positionID+1)
	state.TotalShares = newTotalShares
	state.IndexRemainder = math.ZeroInt()
	k.SetRewardState(cacheCtx, state)
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeStake,
		sdk.NewAttribute("position_id", strconv.FormatUint(positionID, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("amount", position.Principal.String()),
		sdk.NewAttribute("lock_tier_id", strconv.FormatUint(uint64(position.LockTierId), 10)),
		sdk.NewAttribute("shares", position.Shares.String()),
	))
	write()

	return &types.MsgStakeResponse{PositionId: positionID}, nil
}

func (k msgServer) BeginUnbonding(goCtx context.Context, msg *types.MsgBeginUnbonding) (*types.MsgBeginUnbondingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	position, found := k.GetPosition(ctx, msg.PositionId)
	if !found {
		return nil, types.ErrPositionNotFound
	}
	if position.Owner != msg.Owner {
		return nil, types.ErrPositionOwner
	}
	if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE || position.LockDuration == nil {
		return nil, types.ErrInvalidPositionState
	}

	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	position, found = k.GetPosition(cacheCtx, msg.PositionId)
	if !found || position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE || position.LockDuration == nil {
		return nil, types.ErrInvalidPositionState
	}
	state := k.GetRewardState(cacheCtx)
	claimable, err := checkedRewardAdd(position.ClaimableRewards.Amount, k.AccruedRewards(position, state))
	if err != nil {
		return nil, err
	}
	if state.TotalShares.IsNil() || state.TotalShares.LT(position.Shares) {
		return nil, types.ErrInvalidRewardState
	}
	state.TotalShares = state.TotalShares.Sub(position.Shares)
	state.IndexRemainder = math.ZeroInt()
	if state.TotalShares.IsZero() {
		state.EmissionRemainder = math.ZeroInt()
	}
	position.Shares = math.ZeroInt()
	position.RewardDebt = math.LegacyZeroDec()
	position.ClaimableRewards = sdk.NewCoin(types.BondDenom, claimable)
	position.Status = types.PositionStatus_POSITION_STATUS_UNBONDING
	end := cacheCtx.BlockTime().Add(*position.LockDuration)
	position.UnbondingEndTime = &end
	if err := k.SetPosition(cacheCtx, position); err != nil {
		return nil, err
	}
	k.SetRewardState(cacheCtx, state)
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeBeginUnbonding,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("unbonding_end_time", position.UnbondingEndTime.UTC().Format(time.RFC3339Nano)),
		sdk.NewAttribute("claimable_rewards", position.ClaimableRewards.String()),
	))
	write()
	return &types.MsgBeginUnbondingResponse{}, nil
}

func (k msgServer) Withdraw(goCtx context.Context, msg *types.MsgWithdraw) (*types.MsgWithdrawResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	position, found := k.GetPosition(cacheCtx, msg.PositionId)
	if !found {
		return nil, types.ErrPositionNotFound
	}
	if position.Owner != msg.Owner {
		return nil, types.ErrPositionOwner
	}
	if position.Status != types.PositionStatus_POSITION_STATUS_UNBONDING || position.UnbondingEndTime == nil {
		return nil, types.ErrInvalidPositionState
	}
	if cacheCtx.BlockTime().Before(*position.UnbondingEndTime) {
		return nil, types.ErrNotMatured
	}
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.PrincipalPoolName, owner, sdk.NewCoins(position.Principal)); err != nil {
		return nil, err
	}
	principal := position.Principal
	position.Principal = sdk.NewCoin(types.BondDenom, math.ZeroInt())
	position.Status = types.PositionStatus_POSITION_STATUS_WITHDRAWN
	if err := k.saveOrPruneSettledPosition(cacheCtx, position); err != nil {
		return nil, err
	}
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeWithdraw,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("principal", principal.String()),
	))
	write()
	return &types.MsgWithdrawResponse{}, nil
}

func (k msgServer) ClaimRewards(goCtx context.Context, msg *types.MsgClaimRewards) (*types.MsgClaimRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	position, found := k.GetPosition(ctx, msg.PositionId)
	if !found {
		return nil, types.ErrPositionNotFound
	}
	if position.Owner != msg.Owner {
		return nil, types.ErrPositionOwner
	}
	if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE &&
		position.Status != types.PositionStatus_POSITION_STATUS_UNBONDING &&
		position.Status != types.PositionStatus_POSITION_STATUS_WITHDRAWN {
		return nil, types.ErrInvalidPositionState
	}

	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	position, found = k.GetPosition(cacheCtx, msg.PositionId)
	if !found {
		return nil, types.ErrPositionNotFound
	}
	state := k.GetRewardState(cacheCtx)
	pendingWhole := math.ZeroInt()
	if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
		pending, err := activePendingWhole(position, state)
		if err != nil {
			return nil, err
		}
		pendingWhole = pending
	}
	claimable := position.ClaimableRewards.Amount
	if claimable.IsNil() || claimable.IsNegative() {
		return nil, types.ErrInvalidPosition.Wrap("claimable reward amount is invalid")
	}
	amount, err := checkedRewardAdd(pendingWhole, claimable)
	if err != nil {
		return nil, err
	}
	pool := k.bankKeeper.GetBalance(cacheCtx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	backing, err := checkedRewardAdd(state.UnallocatedRewards, state.AllocatedRewards)
	if err != nil {
		return nil, err
	}
	if pool.Amount.IsNil() || pool.Amount.IsNegative() || pool.Amount.LT(backing) || state.AllocatedRewards.LT(amount) {
		return nil, types.ErrRewardPoolInsolvent
	}
	if amount.IsPositive() {
		owner, _ := sdk.AccAddressFromBech32(msg.Owner)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.RewardPoolName, owner, sdk.NewCoins(sdk.NewCoin(types.BondDenom, amount))); err != nil {
			return nil, err
		}
	}
	if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE && pendingWhole.IsPositive() {
		debt, err := sharesRewardDebt(position.Shares, state.RewardIndex)
		if err != nil {
			return nil, err
		}
		currentDebt := position.RewardDebt
		if currentDebt.IsNil() {
			return nil, types.ErrInvalidPosition.Wrap("active position reward debt is missing")
		}
		position.RewardDebt = currentDebt.Add(pendingWhole.ToLegacyDec())
		if position.RewardDebt.GT(debt) {
			return nil, types.ErrInvalidPosition.Wrap("claim would move reward debt above the current entitlement")
		}
	}
	position.ClaimableRewards = sdk.NewCoin(types.BondDenom, math.ZeroInt())
	state.AllocatedRewards = state.AllocatedRewards.Sub(amount)
	if err := k.saveOrPruneSettledPosition(cacheCtx, position); err != nil {
		return nil, err
	}
	k.SetRewardState(cacheCtx, state)
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeClaimRewards,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("amount", sdk.NewCoin(types.BondDenom, amount).String()),
	))
	write()
	return &types.MsgClaimRewardsResponse{Amount: sdk.NewCoin(types.BondDenom, amount)}, nil
}

func (k msgServer) FundRewards(goCtx context.Context, msg *types.MsgFundRewards) (*types.MsgFundRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrUnauthorized
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	cacheCtx, write := ctx.CacheContext()
	beforeState := k.GetRewardState(cacheCtx)
	beforePool := k.bankKeeper.GetBalance(cacheCtx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if err := k.Keeper.FundRewards(cacheCtx, msg.Amount); err != nil {
		return nil, err
	}
	afterState := k.GetRewardState(cacheCtx)
	pool := k.bankKeeper.GetBalance(cacheCtx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeFundRewards,
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("source", "community_pool"),
		sdk.NewAttribute("amount", msg.Amount.String()),
		sdk.NewAttribute("reward_index_before", beforeState.RewardIndex.String()),
		sdk.NewAttribute("reward_index_after", afterState.RewardIndex.String()),
		sdk.NewAttribute("unallocated_rewards", afterState.UnallocatedRewards.String()),
		sdk.NewAttribute("reward_pool_balance_before", beforePool.String()),
		sdk.NewAttribute("reward_pool_balance_after", pool.String()),
	))
	write()
	return &types.MsgFundRewardsResponse{Amount: msg.Amount}, nil
}

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrUnauthorized
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	state := k.GetRewardState(cacheCtx)
	if err := validateProjectedRewardIndex(state, math.ZeroInt()); err != nil {
		return nil, err
	}
	if msg.Params.Paused || msg.Params.RewardRatePerDay.IsZero() {
		state.EmissionRemainder = math.ZeroInt()
		k.SetRewardState(cacheCtx, state)
	}
	k.SetParams(cacheCtx, msg.Params)
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeUpdateParams,
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("paused", strconv.FormatBool(msg.Params.Paused)),
		sdk.NewAttribute("reward_rate_per_day", msg.Params.RewardRatePerDay.String()),
		sdk.NewAttribute("lock_tier_count", strconv.Itoa(len(msg.Params.LockTiers))),
	))
	write()
	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) ReturnRewardDust(goCtx context.Context, msg *types.MsgReturnRewardDust) (*types.MsgReturnRewardDustResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrUnauthorized
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	cacheCtx, write := ctx.CacheContext()
	if err := k.AccrueRewards(cacheCtx); err != nil {
		return nil, err
	}
	state := k.GetRewardState(cacheCtx)
	if state.TotalShares.IsNil() || !state.TotalShares.IsZero() {
		return nil, types.ErrInvalidRewardState.Wrap("dust cannot return while positions are active")
	}
	claimable, err := k.GetReservedRewardsChecked(cacheCtx)
	if err != nil {
		return nil, err
	}
	if claimable.IsNil() || claimable.IsNegative() || state.AllocatedRewards.LT(claimable) {
		return nil, types.ErrRewardPoolInsolvent.Wrap("allocated reserve is below stored claimable rewards")
	}
	pool := k.bankKeeper.GetBalance(cacheCtx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if pool.Amount.IsNil() || pool.Amount.IsNegative() {
		return nil, types.ErrRewardPoolInsolvent
	}
	backing, err := checkedRewardAdd(state.UnallocatedRewards, state.AllocatedRewards)
	if err != nil {
		return nil, err
	}
	if pool.Amount.LT(backing) {
		return nil, types.ErrRewardPoolInsolvent.Wrap("reward pool does not back allocated and unallocated reserves")
	}
	available := pool.Amount.Sub(state.UnallocatedRewards).Sub(claimable)
	if available.IsNegative() || available.LT(msg.Amount.Amount) {
		return nil, types.ErrRewardPoolInsolvent.Wrap("amount exceeds unreserved reward dust")
	}
	state.AllocatedRewards = claimable
	k.SetRewardState(cacheCtx, state)
	if err := k.communityPoolKeeper.ReturnRewardDustToCommunityPool(cacheCtx, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}
	cacheCtx.EventManager().EmitEvent(sdk.NewEvent("ustcstaking_return_reward_dust",
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("amount", msg.Amount.String()),
		sdk.NewAttribute("destination", "community_pool"),
		sdk.NewAttribute("unallocated_rewards", state.UnallocatedRewards.String()),
		sdk.NewAttribute("claimable_rewards", claimable.String()),
	))
	write()
	return &types.MsgReturnRewardDustResponse{}, nil
}

func findLockTier(params types.Params, id uint32) (types.LockTier, bool) {
	for _, tier := range params.LockTiers {
		if tier.Id == id {
			return tier, true
		}
	}
	return types.LockTier{}, false
}

func calculateShares(amount math.Int, multiplier math.LegacyDec) (math.Int, error) {
	if amount.IsNil() || !amount.IsPositive() || multiplier.IsNil() || multiplier.IsNegative() || !multiplier.IsInValidRange() {
		return math.Int{}, types.ErrInvalidLockTier.Wrap("amount or multiplier is invalid")
	}
	shares := new(big.Int).Mul(amount.BigInt(), multiplier.BigInt())
	shares.Quo(shares, big.NewInt(1_000_000_000_000_000_000))
	return checkedRewardInt(shares)
}

func sharesRewardDebt(shares math.Int, index math.LegacyDec) (math.LegacyDec, error) {
	if !shareIndexProductInRange(shares, index) {
		return math.LegacyDec{}, types.ErrInvalidRewardState.Wrap("position reward debt exceeds SDK decimal range")
	}
	raw := new(big.Int).Mul(shares.BigInt(), index.BigInt())
	debt := math.LegacyNewDecFromBigIntWithPrec(raw, math.LegacyPrecision)
	if !debt.IsInValidRange() {
		return math.LegacyDec{}, types.ErrInvalidRewardState.Wrap("position reward debt exceeds SDK decimal range")
	}
	return debt, nil
}

func activePendingWhole(position types.Position, state types.RewardState) (math.Int, error) {
	raw, err := activePendingRaw(position, state)
	if err != nil {
		return math.Int{}, err
	}
	whole := new(big.Int).Quo(raw, big.NewInt(1_000_000_000_000_000_000))
	return checkedRewardInt(whole)
}

func activePendingRaw(position types.Position, state types.RewardState) (*big.Int, error) {
	if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE || position.Shares.IsNil() || !position.Shares.IsPositive() ||
		position.RewardDebt.IsNil() || position.RewardDebt.IsNegative() || state.RewardIndex.IsNil() || state.RewardIndex.IsNegative() {
		return nil, types.ErrInvalidPosition.Wrap("active position reward accounting is invalid")
	}
	if !shareIndexProductInRange(position.Shares, state.RewardIndex) {
		return nil, types.ErrInvalidPosition.Wrap("active position reward entitlement exceeds SDK decimal range")
	}
	exactRaw := new(big.Int).Mul(position.Shares.BigInt(), state.RewardIndex.BigInt())
	exactRaw.Sub(exactRaw, position.RewardDebt.BigInt())
	if exactRaw.Sign() < 0 {
		return nil, types.ErrInvalidPosition.Wrap("active position reward debt exceeds accrued rewards")
	}
	return exactRaw, nil
}
