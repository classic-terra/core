<!--
order: 6
-->

# Parameters

The cron module contains the following parameters:

| Key   | Type   | Example |
|-------|--------|---------|
| limit | uint64 | `5`     |

## limit

`limit` is the maximum number of schedules the module will execute in a single stage pass.

The limit is applied independently to:

- one `BeginBlock` execution pass
- one `EndBlock` execution pass

This parameter bounds block-time work even if more schedules are due.
