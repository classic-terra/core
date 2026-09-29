## Abstract

The `x/cron` module provides a governance-controlled scheduler for recurring CosmWasm contract execution. It stores named schedules, evaluates them at block boundaries, and dispatches due contract calls from the cron module account through Terra's existing Wasm message server.

This implementation is intentionally narrow:

- only module-authority messages may create, remove, or update cron state
- scheduled payloads are Wasm `ExecuteContract` messages with no funds
- execution can occur at either `BeginBlock` or `EndBlock`
- execution throughput per stage is bounded by a module parameter

## Contents

1. **[Concepts](01_concepts.md)**
   - [Scheduled Execution](01_concepts.md#scheduled-execution)
   - [Authority Model](01_concepts.md#authority-model)
   - [Execution Stages](01_concepts.md#execution-stages)
   - [Failure Semantics](01_concepts.md#failure-semantics)
2. **[State](02_state.md)**
   - [Schedule](02_state.md#schedule)
   - [ScheduleCount](02_state.md#schedulecount)
   - [Params](02_state.md#params)
3. **[Block Lifecycle](03_block_lifecycle.md)**
   - [BeginBlock](03_block_lifecycle.md#beginblock)
   - [EndBlock](03_block_lifecycle.md#endblock)
4. **[Messages](04_messages.md)**
   - [MsgAddSchedule](04_messages.md#msgaddschedule)
   - [MsgRemoveSchedule](04_messages.md#msgremoveschedule)
   - [MsgUpdateParams](04_messages.md#msgupdateparams)
5. **[Events](05_events.md)**
   - [Cron Module Events](05_events.md#cron-module-events)
   - [Nested Wasm Events](05_events.md#nested-wasm-events)
6. **[Parameters](06_params.md)**
