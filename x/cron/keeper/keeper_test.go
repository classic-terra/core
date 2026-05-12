package keeper

import (
	"testing"

	"github.com/classic-terra/core/v4/x/cron/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestKeeperScheduleCRUD(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(10)

	err := input.Keeper.AddSchedule(ctx, "job", 3, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}}, 1, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	require.NoError(t, err)

	schedule, found := input.Keeper.GetSchedule(ctx, "job")
	require.True(t, found)
	require.Equal(t, "job", schedule.Name)
	require.Equal(t, uint64(3), schedule.Period)
	require.Equal(t, int32(1), input.Keeper.GetScheduleCount(ctx))

	input.Keeper.RemoveSchedule(ctx, "job")
	require.Equal(t, int32(0), input.Keeper.GetScheduleCount(ctx))
	_, found = input.Keeper.GetSchedule(ctx, "job")
	require.False(t, found)
}

func TestKeeperExecuteReadySchedules(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(5)

	require.NoError(t, input.Keeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-a", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-b", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"pong":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-c", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"skip":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_BEGIN_BLOCKER))

	input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)

	require.Len(t, input.MsgServer.calls, 1)
	schedule, found := input.Keeper.GetSchedule(ctx, "job-a")
	require.True(t, found)
	require.Equal(t, uint64(5), schedule.LastExecuteHeight)
}

func TestKeeperSchedulesPagination(t *testing.T) {
	input := createTestInput(t)

	ctx := input.Ctx.WithBlockHeight(5)
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-a", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-b", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"pong":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-c", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"skip":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	res, err := input.Keeper.Schedules(sdk.WrapSDKContext(ctx), &types.QuerySchedulesRequest{
		Pagination: &query.PageRequest{Limit: 2},
	})
	require.NoError(t, err)
	require.Len(t, res.Schedules, 2)
	require.NotNil(t, res.Pagination)
	require.NotEmpty(t, res.Pagination.NextKey)
}

func TestKeeperScheduleQuery(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(5)

	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-a", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	res, err := input.Keeper.Schedule(sdk.WrapSDKContext(ctx), &types.QueryScheduleRequest{Name: "job-a"})
	require.NoError(t, err)
	require.Equal(t, "job-a", res.Schedule.Name)
}
