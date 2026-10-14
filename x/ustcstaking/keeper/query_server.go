package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const maxValidateStatePageSize uint64 = 500

type queryServer struct{ Keeper }

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Position(goCtx context.Context, req *types.QueryPositionRequest) (*types.QueryPositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	position, found := q.GetPosition(ctx, req.PositionId)
	if !found {
		return nil, sdkerrors.ErrNotFound.Wrapf("position %d", req.PositionId)
	}
	return &types.QueryPositionResponse{Position: position}, nil
}

func (q queryServer) PositionsByOwner(goCtx context.Context, req *types.QueryPositionsByOwnerRequest) (*types.QueryPositionsByOwnerResponse, error) {
	if _, err := sdk.AccAddressFromBech32(req.Owner); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("owner: %s", err)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	index := prefix.NewStore(ctx.KVStore(q.storeKey), types.OwnerPositionKeyPrefix)
	owner, _ := sdk.AccAddressFromBech32(req.Owner)
	store := prefix.NewStore(index, types.OwnerPositionPrefix(owner))
	positions := make([]types.Position, 0)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, _ []byte) error {
		if len(key) != 8 {
			return types.ErrInvalidPosition.Wrapf("invalid owner index key length %d", len(key))
		}
		positionID := sdk.BigEndianToUint64(key)
		position, found := q.GetPosition(ctx, positionID)
		if !found {
			return types.ErrInvalidPosition.Wrapf("owner index references missing position %d", positionID)
		}
		if position.Owner != req.Owner {
			return types.ErrInvalidPosition.Wrapf("owner index position %d belongs to %s", positionID, position.Owner)
		}
		positions = append(positions, position)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryPositionsByOwnerResponse{Positions: positions, Pagination: pageRes}, nil
}

func (q queryServer) RewardState(goCtx context.Context, _ *types.QueryRewardStateRequest) (*types.QueryRewardStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryRewardStateResponse{RewardState: q.GetRewardState(ctx)}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}

func (q queryServer) ValidateState(goCtx context.Context, req *types.QueryValidateStateRequest) (*types.QueryValidateStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pageReq := req.Pagination
	if pageReq == nil {
		pageReq = &query.PageRequest{Limit: maxValidateStatePageSize}
	} else {
		pageCopy := *pageReq
		pageReq = &pageCopy
		if pageReq.Limit == 0 {
			pageReq.Limit = maxValidateStatePageSize
		}
	}
	if pageReq.Limit > maxValidateStatePageSize {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("validation page limit %d exceeds maximum %d", pageReq.Limit, maxValidateStatePageSize)
	}
	if pageReq.Offset != 0 || pageReq.CountTotal {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("validation query only accepts bounded key pagination")
	}
	if len(pageReq.Key) != 0 && len(pageReq.Key) != 8 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("validation page key must be empty or 8 bytes")
	}

	response := &types.QueryValidateStateResponse{
		Valid: true, PrincipalLiability: "0", ActiveShares: "0", RewardLiability: "0",
		PrincipalPoolBalances: sdk.NewCoins(q.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.PrincipalPoolName), types.BondDenom)),
		RewardPoolBalances:    sdk.NewCoins(q.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.RewardPoolName), types.BondDenom)),
		RewardState:           q.GetRewardState(ctx),
	}
	principalLiability, activeShares, rewardLiability := math.ZeroInt(), math.ZeroInt(), math.ZeroInt()
	positions := prefix.NewStore(ctx.KVStore(q.storeKey), types.PositionKeyPrefix)
	pageRes, err := query.Paginate(positions, pageReq, func(key, value []byte) error {
		if len(key) != 8 {
			return types.ErrInvalidPosition.Wrapf("invalid primary position key length %d", len(key))
		}
		var position types.Position
		if err := q.cdc.Unmarshal(value, &position); err != nil {
			return types.ErrInvalidPosition.Wrapf("cannot decode position key %x: %s", key, err)
		}
		if position.Id != sdk.BigEndianToUint64(key) {
			return types.ErrInvalidPosition.Wrapf("primary position key does not match stored ID %d", position.Id)
		}
		if position.ClaimableRewards.Denom != types.BondDenom || position.ClaimableRewards.Amount.IsNil() || position.ClaimableRewards.Amount.IsNegative() {
			return types.ErrInvalidPosition.Wrapf("position %d has invalid claimable rewards", position.Id)
		}
		if position.Status == types.PositionStatus_POSITION_STATUS_WITHDRAWN {
			if position.Principal.Amount.IsNil() || !position.Principal.Amount.IsZero() {
				return types.ErrInvalidPosition.Wrapf("withdrawn position %d retains principal", position.Id)
			}
		} else {
			if position.Principal.Denom != types.BondDenom || position.Principal.Amount.IsNil() || !position.Principal.Amount.IsPositive() {
				return types.ErrInvalidPosition.Wrapf("position %d has invalid principal", position.Id)
			}
			principalLiability = principalLiability.Add(position.Principal.Amount)
		}

		positionRewards := position.ClaimableRewards.Amount
		switch position.Status {
		case types.PositionStatus_POSITION_STATUS_ACTIVE:
			state := response.RewardState
			if position.Shares.IsNil() || !position.Shares.IsPositive() || position.RewardDebt.IsNil() || state.RewardIndex.IsNil() || state.RewardIndex.IsNegative() {
				return types.ErrInvalidPosition.Wrapf("active position %d has invalid share accounting", position.Id)
			}
			activeShares = activeShares.Add(position.Shares)
			raw := position.Shares.ToLegacyDec().Mul(state.RewardIndex).Sub(position.RewardDebt)
			if raw.IsNegative() {
				return types.ErrInvalidPosition.Wrapf("active position %d reward debt exceeds entitlement", position.Id)
			}
			positionRewards = positionRewards.Add(raw.TruncateInt())
		case types.PositionStatus_POSITION_STATUS_UNBONDING, types.PositionStatus_POSITION_STATUS_WITHDRAWN:
			if position.Shares.IsNil() || !position.Shares.IsZero() || position.RewardDebt.IsNil() || !position.RewardDebt.IsZero() {
				return types.ErrInvalidPosition.Wrapf("inactive position %d retains shares or reward debt", position.Id)
			}
		default:
			return types.ErrInvalidPosition.Wrapf("position %d has unknown status %s", position.Id, position.Status)
		}
		rewardLiability = rewardLiability.Add(positionRewards)
		return nil
	})
	if err != nil {
		return nil, err
	}
	response.Pagination = pageRes
	response.PrincipalLiability = principalLiability.String()
	response.ActiveShares = activeShares.String()
	response.RewardLiability = rewardLiability.String()
	response.Detail = fmt.Sprintf("validated page of up to %d positions", pageReq.Limit)
	return response, nil
}
