<!--
order: 1
-->

# Concepts

## Scheduled Execution

A cron schedule is a named record that says:

- which Wasm contract messages to execute
- how often to execute them, measured in block intervals
- whether execution happens in `BeginBlock` or `EndBlock`
- the last block height at which the schedule was attempted
- the last block height at which the schedule fully succeeded

At each supported block stage, the module scans stored schedules, selects the ones whose interval has elapsed, and executes up to the configured per-stage limit.

## Authority Model

Cron state is admin-controlled. The module exposes `MsgAddSchedule`, `MsgRemoveSchedule`, and `MsgUpdateParams`, but each message is gated by the module `authority` address configured when the keeper is constructed.

In Terra Classic app wiring, this authority is the standard governance authority address. A normal user account cannot directly create or modify schedules unless it is the configured authority.

## Execution Stages

The module supports two execution stages:

- `EXECUTION_STAGE_BEGIN_BLOCKER`
- `EXECUTION_STAGE_END_BLOCKER`

The stage is part of the stored schedule and is checked before execution. The app wires cron into both `BeginBlock` and `EndBlock`, so the same module can host schedules that run at either lifecycle point.

## Failure Semantics

Cron treats one schedule as one atomic job. A schedule may contain multiple Wasm contract messages, but those messages are expected to be related steps of the same job.

- Before execution starts, the module updates `last_run_height` on the stored schedule.
- It then executes each nested contract message using the cron module account as the sender.
- All nested messages run in one cached context with a schedule-level gas limit.
- If every nested message succeeds, the cached writes are committed, `last_execute_height` is updated, and `last_execution_error` is cleared.
- If any nested message fails or the schedule runs out of gas, the cached writes are discarded, `last_execute_height` is not updated, and `last_execution_error` is stored on the schedule.

Failed schedules do not prevent other ready schedules from running. Retry pacing is based on `last_run_height`, so a failed schedule waits for its period before being attempted again.
