<!--
order: 6
-->

# Parameters

The cron module contains the following parameters:

| Key               | Type   | Example   |
|-------------------|--------|-----------|
| limit             | uint64 | `5`       |
| max_execution_gas | uint64 | `5000000` |

## limit

`limit` is the maximum number of schedules the module will execute in a single stage pass.

The limit is applied independently to:

- one `BeginBlock` execution pass
- one `EndBlock` execution pass

This parameter bounds block-time work even if more schedules are due.

## max_execution_gas

`max_execution_gas` is the maximum gas one atomic schedule execution may consume.

The gas limit applies to the full schedule, including all nested contract messages. If the schedule exceeds this limit, the cached writes are discarded, the schedule records a queryable execution error, and later ready schedules may still run.
