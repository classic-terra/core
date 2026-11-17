package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/cron/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryParamsResponse{Params: k.GetParams(sdk.UnwrapSDKContext(c))}, nil
}

func (k Keeper) Schedule(c context.Context, req *types.QueryScheduleRequest) (*types.QueryScheduleResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	schedule, found := k.GetSchedule(sdk.UnwrapSDKContext(c), req.Name)
	if !found {
		return nil, status.Error(codes.NotFound, "schedule not found")
	}

	return &types.QueryScheduleResponse{Schedule: *schedule}, nil
}

func (k Keeper) Schedules(c context.Context, req *types.QuerySchedulesRequest) (*types.QuerySchedulesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	var schedules []types.Schedule

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var schedule types.Schedule
		k.cdc.MustUnmarshal(value, &schedule)
		schedules = append(schedules, schedule)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QuerySchedulesResponse{Schedules: schedules, Pagination: pageRes}, nil
}
