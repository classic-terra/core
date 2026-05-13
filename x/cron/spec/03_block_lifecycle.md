<!--
order: 3
-->

# Block Lifecycle

## BeginBlock

During `BeginBlock`, the app calls:

```go
am.keeper.ExecuteReadySchedules(ctx, EXECUTION_STAGE_BEGIN_BLOCKER)
```

The keeper:

1. loads current params
2. iterates all stored schedules
3. selects schedules whose interval has elapsed and whose `execution_stage` is `BEGIN_BLOCKER`
4. stops once `params.limit` schedules have been selected
5. executes each selected schedule in order

A schedule is due when the current block height is at least `schedule.last_run_height + schedule.period`. For older state where `last_run_height` is unset, the keeper falls back to `last_execute_height`.

## EndBlock

During `EndBlock`, the app calls:

```go
am.keeper.ExecuteReadySchedules(ctx, EXECUTION_STAGE_END_BLOCKER)
```

The same selection and execution flow is used, but only schedules tagged for `END_BLOCKER` are considered.

## Nested Contract Dispatch

Each nested schedule message is dispatched as a Wasm `MsgExecuteContract` with:

- `sender = cron module account`
- `contract = schedule message contract`
- `msg = schedule message JSON payload`
- `funds = []`

Cron does not mint, move, or attach native coins itself. Any downstream token movement comes from the executed contract logic.

## Atomic Schedule Execution

Each selected schedule is executed as one atomic job:

1. `last_run_height` is updated before the contract calls are attempted.
2. all nested contract calls run in a cached context with a schedule-level gas meter
3. if all contract calls succeed, cached writes are committed, `last_execute_height` is updated, and `last_execution_error` is cleared
4. if any contract call fails or the gas meter is exhausted, cached writes are discarded and `last_execution_error` is persisted

A failed schedule does not stop later ready schedules from executing in the same stage pass.
