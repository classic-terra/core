<!--
order: 5
-->

# Events

## Cron Module Events

The cron module does not define custom cron-specific event types in its keeper or message server code.

Successful authority messages still emit the standard SDK `message` event metadata produced by the message execution stack.

Execution failures are logged with:

- `schedule_name`
- `contract`
- `error`

but these log lines are not ABCI events.

Execution failures are also persisted on the schedule as `last_execution_error`. This field includes the failing contract and is cleared after the schedule later completes successfully.

## Nested Wasm Events

When an atomic schedule succeeds, any events emitted by the nested Wasm executions are surfaced through the normal Wasm execution pipeline.

Cron itself does not rewrite or add additional execution events around those nested calls.
