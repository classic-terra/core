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

## EndBlock

During `EndBlock`, the app calls:

```go
am.keeper.ExecuteReadySchedules(ctx, EXECUTION_STAGE_END_BLOCKER)
```

The same selection and execution flow is used, but only schedules tagged for `END_BLOCKER` are considered.

## Nested Contract Dispatch

Each scheduled item is dispatched as a Wasm `MsgExecuteContract` with:

- `sender = cron module account`
- `contract = schedule message contract`
- `msg = schedule message JSON payload`
- `funds = []`

Cron does not mint, move, or attach native coins itself. Any downstream token movement comes from the executed contract logic.
