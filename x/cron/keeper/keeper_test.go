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
	require.Equal(t, uint64(5), schedule.LastRunHeight)
	require.Equal(t, uint64(5), schedule.LastExecuteHeight)
}

func TestKeeperExecuteReadySchedules_ExecutionGasLimit(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(5)
	input.MsgServer.gasToConsume = 11

	require.NoError(t, input.Keeper.SetParams(ctx, types.Params{Limit: 1, MaxExecutionGas: 10}))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job", 1, []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	require.NotPanics(t, func() {
		input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	})

	require.Len(t, input.MsgServer.calls, 1)
	require.Equal(t, uint64(10), input.MsgServer.observedGasLimit)
}

func TestKeeperExecuteReadySchedules_FailedScheduleDoesNotBlockOthers(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(8)
	input.MsgServer.errByContract = map[string]error{
		"terra1fail": errFailedContract(),
	}

	require.NoError(t, input.Keeper.SetParams(ctx, types.NewParams(2)))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-a", 1, []types.MsgExecuteContract{
		{Contract: "terra1fail", Msg: `{"fail":{}}`},
		{Contract: "terra1ok", Msg: `{"ok":{}}`},
	}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job-b", 1, []types.MsgExecuteContract{
		{Contract: "terra1later", Msg: `{"later":{}}`},
	}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)

	require.Equal(t, []string{"terra1fail", "terra1later"}, calledContracts(input.MsgServer.calls))
	schedule, found := input.Keeper.GetSchedule(ctx, "job-a")
	require.True(t, found)
	require.Equal(t, uint64(8), schedule.LastRunHeight)
	require.Equal(t, uint64(0), schedule.LastExecuteHeight)
	require.Contains(t, schedule.LastExecutionError, "terra1fail")
	require.Contains(t, schedule.LastExecutionError, "contract execution failed")

	schedule, found = input.Keeper.GetSchedule(ctx, "job-b")
	require.True(t, found)
	require.Equal(t, uint64(8), schedule.LastRunHeight)
	require.Equal(t, uint64(8), schedule.LastExecuteHeight)
}

func TestKeeperExecuteReadySchedules_FailureOnlyUpdatesRunHeight(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(8)
	input.MsgServer.errByContract = map[string]error{
		"terra1fail": errFailedContract(),
	}

	require.NoError(t, input.Keeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job", 1, []types.MsgExecuteContract{
		{Contract: "terra1fail", Msg: `{"fail":{}}`},
	}, 3, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)

	schedule, found := input.Keeper.GetSchedule(ctx, "job")
	require.True(t, found)
	require.Equal(t, uint64(8), schedule.LastRunHeight)
	require.Equal(t, uint64(3), schedule.LastExecuteHeight)
	require.Contains(t, schedule.LastExecutionError, "terra1fail")
}

func TestKeeperExecuteReadySchedules_SuccessClearsLastExecutionError(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(8)
	input.MsgServer.errByContract = map[string]error{
		"terra1contract": errFailedContract(),
	}

	require.NoError(t, input.Keeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job", 1, []types.MsgExecuteContract{
		{Contract: "terra1contract", Msg: `{"ping":{}}`},
	}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	schedule, found := input.Keeper.GetSchedule(ctx, "job")
	require.True(t, found)
	require.NotEmpty(t, schedule.LastExecutionError)

	input.MsgServer.errByContract = nil
	input.Keeper.ExecuteReadySchedules(ctx.WithBlockHeight(9), types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)

	schedule, found = input.Keeper.GetSchedule(ctx, "job")
	require.True(t, found)
	require.Empty(t, schedule.LastExecutionError)
	require.Equal(t, uint64(9), schedule.LastExecuteHeight)
}

func TestKeeperExecuteReadySchedules_FailedAttemptWaitsForPeriodBeforeRetry(t *testing.T) {
	input := createTestInput(t)
	ctx := input.Ctx.WithBlockHeight(10)
	input.MsgServer.errByContract = map[string]error{
		"terra1fail": errFailedContract(),
	}

	require.NoError(t, input.Keeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, input.Keeper.AddSchedule(ctx, "job", 5, []types.MsgExecuteContract{
		{Contract: "terra1fail", Msg: `{"fail":{}}`},
	}, 0, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER))

	input.Keeper.ExecuteReadySchedules(ctx, types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	require.Len(t, input.MsgServer.calls, 1)

	input.Keeper.ExecuteReadySchedules(ctx.WithBlockHeight(11), types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	require.Len(t, input.MsgServer.calls, 1)

	input.Keeper.ExecuteReadySchedules(ctx.WithBlockHeight(15), types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER)
	require.Len(t, input.MsgServer.calls, 2)
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

func calledContracts(calls []*types.MsgExecuteContract) []string {
	contracts := make([]string, 0, len(calls))
	for _, call := range calls {
		contracts = append(contracts, call.Contract)
	}
	return contracts
}
