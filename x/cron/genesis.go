package cron

import (
	"github.com/classic-terra/core/v4/x/cron/keeper"
	"github.com/classic-terra/core/v4/x/cron/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	for _, schedule := range genState.ScheduleList {
		if err := k.AddSchedule(ctx, schedule.Name, schedule.Period, schedule.Msgs, schedule.LastExecuteHeight, schedule.ExecutionStage); err != nil {
			panic(err)
		}
	}
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesisState()
	genesis.Params = k.GetParams(ctx)
	genesis.ScheduleList = k.GetAllSchedules(ctx)
	return genesis
}
