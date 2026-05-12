<!--
order: 1
-->

# Concepts

## Scheduled Execution

A cron schedule is a named record that says:

- which Wasm contract messages to execute
- how often to execute them, measured in block intervals
- whether execution happens in `BeginBlock` or `EndBlock`
- the last block height at which the schedule executed

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

Cron executes each due schedule in a cached context.

- Before execution starts, the module updates `last_execute_height` on the stored schedule.
- It then executes each nested contract message using the cron module account as the sender.
- If any nested message fails, the cached writes are discarded, so no partial Wasm effects are committed for that schedule.

The failure is logged, but the schedule remains advanced to the current height because the schedule record itself is updated before the cached execution block. This means failed schedules are not retried again in the same block.
