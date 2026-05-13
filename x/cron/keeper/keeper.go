package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v4/x/cron/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	accountKeeper types.AccountKeeper
	WasmMsgServer types.WasmMsgServer
	authority     string
}

// NewKeeper creates a new cron Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		accountKeeper: accountKeeper,
		authority:     authority,
	}
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ExecuteReadySchedules executes all schedules eligible for the given stage.
func (k Keeper) ExecuteReadySchedules(ctx sdk.Context, executionStage types.ExecutionStage) {
	schedules := k.getSchedulesReadyForExecution(ctx, executionStage)
	for _, schedule := range schedules {
		_ = k.executeSchedule(ctx, schedule)
	}
}

// AddSchedule validates and stores a new schedule.
func (k Keeper) AddSchedule(
	ctx sdk.Context,
	name string,
	period uint64,
	msgs []types.MsgExecuteContract,
	lastExecuteHeight uint64,
	executionStage types.ExecutionStage,
) error {
	schedule := types.Schedule{
		Name:              name,
		Period:            period,
		Msgs:              msgs,
		LastExecuteHeight: lastExecuteHeight,
		ExecutionStage:    executionStage,
	}
	if err := types.ValidateSchedule(schedule); err != nil {
		return err
	}
	if k.scheduleExists(ctx, name) {
		return fmt.Errorf("%w: %s", types.ErrDuplicateSchedule, name)
	}

	k.storeSchedule(ctx, schedule)
	k.changeTotalCount(ctx, 1)
	return nil
}

// RemoveSchedule deletes a schedule if it exists.
func (k Keeper) RemoveSchedule(ctx sdk.Context, name string) {
	if !k.scheduleExists(ctx, name) {
		return
	}

	k.changeTotalCount(ctx, -1)
	k.removeSchedule(ctx, name)
}

// GetSchedule returns the schedule with the given name, if it exists.
func (k Keeper) GetSchedule(ctx sdk.Context, name string) (*types.Schedule, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	bz := store.Get(types.GetScheduleKey(name))
	if bz == nil {
		return nil, false
	}

	var schedule types.Schedule
	k.cdc.MustUnmarshal(bz, &schedule)
	return &schedule, true
}

// GetAllSchedules returns all stored schedules.
func (k Keeper) GetAllSchedules(ctx sdk.Context) []types.Schedule {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	res := make([]types.Schedule, 0)

	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var schedule types.Schedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)
		res = append(res, schedule)
	}
	return res
}

// GetScheduleCount returns the total number of stored schedules.
func (k Keeper) GetScheduleCount(ctx sdk.Context) int32 {
	return k.getScheduleCount(ctx)
}

func (k Keeper) getSchedulesReadyForExecution(ctx sdk.Context, executionStage types.ExecutionStage) []types.Schedule {
	params := k.GetParams(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	res := make([]types.Schedule, 0)
	count := uint64(0)

	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var schedule types.Schedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)
		if k.intervalPassed(ctx, schedule) && schedule.ExecutionStage == executionStage {
			res = append(res, schedule)
			count++
			if count >= params.Limit {
				return res
			}
		}
	}

	return res
}

func (k Keeper) executeSchedule(ctx sdk.Context, schedule types.Schedule) (err error) {
	params := k.GetParams(ctx)
	schedule.LastRunHeight = uint64(ctx.BlockHeight())
	k.storeSchedule(ctx, schedule)

	cacheCtx, writeFn := ctx.CacheContext()
	limitedCtx := cacheCtx.WithGasMeter(storetypes.NewGasMeter(params.MaxExecutionGas))
	currentContract := ""
	lastExecutionErr := func(contract string, err error) string {
		if contract == "" {
			return err.Error()
		}
		return fmt.Sprintf("%s: %v", contract, err)
	}

	defer func() {
		if r := recover(); r != nil {
			if oog, ok := r.(storetypes.ErrorOutOfGas); ok {
				err = fmt.Errorf("cron execute out of gas: %s", oog.Descriptor)
				schedule.LastExecutionError = lastExecutionErr(currentContract, err)
				k.storeSchedule(ctx, schedule)
				ctx.Logger().Info("cron execute out of gas",
					"schedule_name", schedule.Name,
					"contract", currentContract,
					"max_execution_gas", params.MaxExecutionGas,
					"error", err,
				)
				return
			}
			err = fmt.Errorf("cron execute panic: %v", r)
			schedule.LastExecutionError = lastExecutionErr(currentContract, err)
			k.storeSchedule(ctx, schedule)
			ctx.Logger().Info("cron execute panic",
				"schedule_name", schedule.Name,
				"contract", currentContract,
				"error", err,
			)
		}
	}()

	for _, msg := range schedule.Msgs {
		currentContract = msg.Contract
		executeMsg := wasmtypes.MsgExecuteContract{
			Sender:   k.accountKeeper.GetModuleAddress(types.ModuleName).String(),
			Contract: msg.Contract,
			Msg:      []byte(msg.Msg),
			Funds:    sdk.NewCoins(),
		}

		if _, err := k.WasmMsgServer.ExecuteContract(sdk.WrapSDKContext(limitedCtx), &executeMsg); err != nil {
			schedule.LastExecutionError = lastExecutionErr(msg.Contract, err)
			k.storeSchedule(ctx, schedule)
			ctx.Logger().Info("cron execute failed",
				"schedule_name", schedule.Name,
				"contract", msg.Contract,
				"error", err,
			)
			return err
		}
	}

	writeFn()
	schedule.LastExecuteHeight = uint64(ctx.BlockHeight())
	schedule.LastExecutionError = ""
	k.storeSchedule(ctx, schedule)
	return nil
}

func (k Keeper) storeSchedule(ctx sdk.Context, schedule types.Schedule) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	store.Set(types.GetScheduleKey(schedule.Name), k.cdc.MustMarshal(&schedule))
}

func (k Keeper) removeSchedule(ctx sdk.Context, name string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	store.Delete(types.GetScheduleKey(name))
}

func (k Keeper) scheduleExists(ctx sdk.Context, name string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	return store.Has(types.GetScheduleKey(name))
}

func (k Keeper) intervalPassed(ctx sdk.Context, schedule types.Schedule) bool {
	lastRunHeight := schedule.LastRunHeight
	if lastRunHeight == 0 {
		lastRunHeight = schedule.LastExecuteHeight
	}
	return uint64(ctx.BlockHeight()) >= (lastRunHeight + schedule.Period)
}

func (k Keeper) changeTotalCount(ctx sdk.Context, incrementAmount int32) {
	store := ctx.KVStore(k.storeKey)
	count := k.getScheduleCount(ctx)
	newCount := types.ScheduleCount{Count: count + incrementAmount}
	store.Set(types.ScheduleCountKey, k.cdc.MustMarshal(&newCount))
}

func (k Keeper) getScheduleCount(ctx sdk.Context) int32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ScheduleCountKey)
	if bz == nil {
		return 0
	}

	var count types.ScheduleCount
	k.cdc.MustUnmarshal(bz, &count)
	return count.Count
}
