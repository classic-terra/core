package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/classic-terra/core/v4/x/cron/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) AddSchedule(goCtx context.Context, req *types.MsgAddSchedule) (*types.MsgAddScheduleResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate MsgAddSchedule")
	}

	if authority := k.keeper.GetAuthority(); authority != req.Authority {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid authority; expected %s, got %s", authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.keeper.AddSchedule(ctx, req.Name, req.Period, req.Msgs, uint64(ctx.BlockHeight()), req.ExecutionStage); err != nil {
		return nil, errors.Wrap(err, "failed to add schedule")
	}

	return &types.MsgAddScheduleResponse{}, nil
}

func (k msgServer) RemoveSchedule(goCtx context.Context, req *types.MsgRemoveSchedule) (*types.MsgRemoveScheduleResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate MsgRemoveSchedule")
	}

	if authority := k.keeper.GetAuthority(); authority != req.Authority {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid authority; expected %s, got %s", authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.keeper.RemoveSchedule(ctx, req.Name)

	return &types.MsgRemoveScheduleResponse{}, nil
}

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate MsgUpdateParams")
	}

	if authority := k.keeper.GetAuthority(); authority != req.Authority {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid authority; expected %s, got %s", authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.keeper.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
