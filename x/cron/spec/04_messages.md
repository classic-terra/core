<!--
order: 4
-->

# Messages

## MsgAddSchedule

Adds a new named schedule.

```go
type MsgAddSchedule struct {
	Authority      string
	Name           string
	Period         uint64
	Msgs           []MsgExecuteContract
	ExecutionStage ExecutionStage
}
```

Validation rules:

- `authority` must be a valid Bech32 account address
- `name` must be non-empty
- `period` must be greater than zero
- `msgs` must be non-empty
- `execution_stage` must be a defined enum value
- each nested contract message must have non-empty `contract` and `msg`
- each nested `msg` string must be valid JSON

The handler also enforces that `authority` equals the keeper authority configured by the app.

## MsgRemoveSchedule

Removes a schedule by name.

```go
type MsgRemoveSchedule struct {
	Authority string
	Name      string
}
```

Validation rules:

- `authority` must be a valid Bech32 account address
- `name` must be non-empty

The handler is authority-gated. Removing a missing schedule is a no-op at keeper level.

## MsgUpdateParams

Updates cron module parameters.

```go
type MsgUpdateParams struct {
	Authority string
	Params    Params
}
```

Validation rules:

- `authority` must be a valid Bech32 account address
- `params.limit` must be greater than zero

The handler is authority-gated.
