<!--
order: 2
-->

# State

## Schedule

`Schedule` is the primary cron state object.

- `Schedule`: `0x01<name_bytes> -> proto(Schedule)`

```go
type Schedule struct {
	Name              string
	Period            uint64
	Msgs              []MsgExecuteContract
	LastExecuteHeight uint64
	ExecutionStage    ExecutionStage
}
```

Field meanings:

- `Name`: unique schedule identifier
- `Period`: execution interval in blocks
- `Msgs`: ordered list of Wasm execute payloads
- `LastExecuteHeight`: last successful scheduling checkpoint used for interval calculation
- `ExecutionStage`: whether the schedule runs in `BeginBlock` or `EndBlock`

`Msgs` contains only contract address plus raw JSON message payload:

```go
type MsgExecuteContract struct {
	Contract string
	Msg      string
}
```

## ScheduleCount

The module stores a denormalized count of schedules.

- `ScheduleCount`: `0x02 -> proto(ScheduleCount)`

```go
type ScheduleCount struct {
	Count int32
}
```

This count is incremented on add and decremented on remove.

## Params

Cron parameters are stored under a dedicated params key.

- `Params`: `0x03 -> proto(Params)`

```go
type Params struct {
	Limit uint64
}
```

`Limit` bounds how many schedules may execute during one pass of `BeginBlock` or `EndBlock`.
