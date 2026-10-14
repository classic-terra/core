package keeper

import (
	"context"
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
	shares := msg.Amount.Amount.ToLegacyDec().Mul(tier.Multiplier).TruncateInt()
	if !shares.IsPositive() {
		return nil, types.ErrInvalidLockTier.Wrap("amount and multiplier produce zero shares")
	}
	positionID := k.GetNextPositionID(ctx)
	if positionID == ^uint64(0) {
		return nil, types.ErrPositionIDExhausted
	}
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, types.PrincipalPoolName, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}

	state := k.GetRewardState(ctx)
	index := state.RewardIndex
	if index.IsNil() {
		index = math.LegacyZeroDec()
	}
	position := types.Position{
		Id:               positionID,
		Owner:            msg.Owner,
		Principal:        msg.Amount,
		Shares:           shares,
		LockTierId:       msg.LockTierId,
		ShareMultiplier:  tier.Multiplier,
		RewardDebt:       shares.ToLegacyDec().Mul(index),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
		LockDuration:     tier.Duration,
	}
	if err := k.SetPosition(ctx, position); err != nil {
		return nil, err
	}
	k.SetNextPositionID(ctx, positionID+1)
	if state.TotalShares.IsNil() {
		state.TotalShares = math.ZeroInt()
	}
	state.TotalShares = state.TotalShares.Add(shares)
	k.SetRewardState(ctx, state)
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeStake,
		sdk.NewAttribute("position_id", strconv.FormatUint(positionID, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("amount", position.Principal.String()),
		sdk.NewAttribute("lock_tier_id", strconv.FormatUint(uint64(position.LockTierId), 10)),
		sdk.NewAttribute("shares", position.Shares.String()),
	))

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

	state := k.GetRewardState(ctx)
	claimable := position.ClaimableRewards.Amount
	if claimable.IsNil() {
		claimable = math.ZeroInt()
	}
	claimable = claimable.Add(k.AccruedRewards(position, state))
	if state.TotalShares.IsNil() || state.TotalShares.LT(position.Shares) {
		return nil, types.ErrInvalidRewardState
	}
	state.TotalShares = state.TotalShares.Sub(position.Shares)
	position.Shares = math.ZeroInt()
	position.RewardDebt = math.LegacyZeroDec()
	position.ClaimableRewards = sdk.NewCoin(types.BondDenom, claimable)
	position.Status = types.PositionStatus_POSITION_STATUS_UNBONDING
	end := ctx.BlockTime().Add(*position.LockDuration)
	position.UnbondingEndTime = &end
	if err := k.SetPosition(ctx, position); err != nil {
		return nil, err
	}
	k.SetRewardState(ctx, state)
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeBeginUnbonding,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("unbonding_end_time", position.UnbondingEndTime.UTC().Format(time.RFC3339Nano)),
		sdk.NewAttribute("claimable_rewards", position.ClaimableRewards.String()),
	))
	return &types.MsgBeginUnbondingResponse{}, nil
}

func (k msgServer) Withdraw(goCtx context.Context, msg *types.MsgWithdraw) (*types.MsgWithdrawResponse, error) {
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
	if position.Status != types.PositionStatus_POSITION_STATUS_UNBONDING || position.UnbondingEndTime == nil {
		return nil, types.ErrInvalidPositionState
	}
	if ctx.BlockTime().Before(*position.UnbondingEndTime) {
		return nil, types.ErrNotMatured
	}
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.PrincipalPoolName, owner, sdk.NewCoins(position.Principal)); err != nil {
		return nil, err
	}
	principal := position.Principal
	position.Principal = sdk.NewCoin(types.BondDenom, math.ZeroInt())
	position.Status = types.PositionStatus_POSITION_STATUS_WITHDRAWN
	if err := k.saveOrPruneSettledPosition(ctx, position); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeWithdraw,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("principal", principal.String()),
	))
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

	state := k.GetRewardState(ctx)
	amount := k.AccruedRewards(position, state)
	if amount.IsNil() {
		amount = math.ZeroInt()
	}
	if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE && !position.ClaimableRewards.Amount.IsNil() {
		amount = amount.Add(position.ClaimableRewards.Amount)
	}
	if amount.IsZero() {
		ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeClaimRewards,
			sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
			sdk.NewAttribute("owner", position.Owner),
			sdk.NewAttribute("amount", sdk.NewCoin(types.BondDenom, amount).String()),
		))
		return &types.MsgClaimRewardsResponse{Amount: sdk.NewCoin(types.BondDenom, amount)}, nil
	}
	pool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if pool.Amount.IsNil() || pool.Amount.LT(amount) {
		return nil, types.ErrRewardPoolInsolvent
	}
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RewardPoolName, owner, sdk.NewCoins(sdk.NewCoin(types.BondDenom, amount))); err != nil {
		return nil, err
	}
	if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
		index := state.RewardIndex
		if index.IsNil() {
			index = math.LegacyZeroDec()
		}
		position.RewardDebt = position.Shares.ToLegacyDec().Mul(index)
	}
	position.ClaimableRewards = sdk.NewCoin(types.BondDenom, math.ZeroInt())
	if err := k.saveOrPruneSettledPosition(ctx, position); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeClaimRewards,
		sdk.NewAttribute("position_id", strconv.FormatUint(position.Id, 10)),
		sdk.NewAttribute("owner", position.Owner),
		sdk.NewAttribute("amount", sdk.NewCoin(types.BondDenom, amount).String()),
	))
	return &types.MsgClaimRewardsResponse{Amount: sdk.NewCoin(types.BondDenom, amount)}, nil
}

func (k msgServer) FundRewards(goCtx context.Context, msg *types.MsgFundRewards) (*types.MsgFundRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.Paused {
		return nil, types.ErrModulePaused
	}
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrUnauthorized
	}
	before := k.GetRewardState(ctx).RewardIndex
	beforePool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if err := k.Keeper.FundRewards(ctx, msg.Amount); err != nil {
		return nil, err
	}
	after := k.GetRewardState(ctx).RewardIndex
	pool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeFundRewards,
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("source", "community_pool"),
		sdk.NewAttribute("amount", msg.Amount.String()),
		sdk.NewAttribute("reward_index_before", before.String()),
		sdk.NewAttribute("reward_index_after", after.String()),
		sdk.NewAttribute("reward_pool_balance_before", beforePool.String()),
		sdk.NewAttribute("reward_pool_balance_after", pool.String()),
	))
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
	k.SetParams(ctx, msg.Params)
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeUpdateParams,
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("paused", strconv.FormatBool(msg.Params.Paused)),
		sdk.NewAttribute("lock_tier_count", strconv.Itoa(len(msg.Params.LockTiers))),
	))
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
	state := k.GetRewardState(ctx)
	if state.TotalShares.IsNil() || !state.TotalShares.IsZero() {
		return nil, types.ErrInvalidRewardState.Wrap("dust cannot return while positions are active")
	}
	pool := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)
	if pool.Amount.IsNil() || pool.Amount.IsNegative() {
		return nil, types.ErrRewardPoolInsolvent
	}
	available := pool.Amount.Sub(k.GetReservedRewards(ctx))
	if available.IsNegative() || available.LT(msg.Amount.Amount) {
		return nil, types.ErrRewardPoolInsolvent.Wrap("amount exceeds unreserved reward dust")
	}
	if err := k.communityPoolKeeper.ReturnRewardDustToCommunityPool(ctx, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("ustcstaking_return_reward_dust", sdk.NewAttribute("authority", msg.Authority), sdk.NewAttribute("amount", msg.Amount.String()), sdk.NewAttribute("destination", "community_pool")))
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
