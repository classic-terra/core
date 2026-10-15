<!--
order: 4
-->

# Messages

## User messages

### `MsgStake`

```go
type MsgStake struct {
    Owner      string
    Amount     sdk.Coin // positive uusd
    LockTierId uint32
}
```

Validates the owner, denomination, amount, pause state, and configured tier. Transfers the amount to the principal pool and returns a new position ID.

### `MsgBeginUnbonding`

```go
type MsgBeginUnbonding struct {
    Owner      string
    PositionId uint64
}
```

Requires ownership and `ACTIVE` status. Settles accrued rewards, removes active shares, records the maturity timestamp using the position's duration snapshot, and changes status to `UNBONDING`.

### `MsgWithdraw`

```go
type MsgWithdraw struct {
    Owner      string
    PositionId uint64
}
```

Requires ownership, `UNBONDING` status, and a maturity timestamp no later than the current block time. Transfers the original principal from the principal pool and marks the position `WITHDRAWN`.

### `MsgClaimRewards`

```go
type MsgClaimRewards struct {
    Owner      string
    PositionId uint64
}
```

Settles rewards for the position and transfers them from the reward pool. Claims fail if the reward pool lacks the required balance. The position remains usable according to its lifecycle status.

### `MsgFundRewards`

```go
type MsgFundRewards struct {
    Authority string // governance module address
    Amount sdk.Coin // positive uusd
}
```

Requires the immutable governance module authority. Debits the distribution community pool and transfers the existing funds into the reward pool, increasing unallocated reserve `U`. It does not immediately increase the reward index and is allowed before staking, while paused, or after exhaustion. Users stake directly through `MsgStake`; no contract handles principal.

### `MsgReturnRewardDust`

Requires the immutable governance module authority and a positive `uusd` amount. The amount must not exceed `B - U - C`, where `B` is the reward-pool bank balance and `C` is the stored integer claim reserve; total active shares must be zero. Funds return to the distribution community pool and `A` is reconciled to `C` only if the transfer succeeds. This operation is allowed while paused, but never returns `U`.

## Authority messages

### `MsgUpdateParams`

Requires the immutable governance module authority and replaces the complete validated parameter set. This is the control used to configure tiers, set the global `reward_rate_per_day` in `uusd` per 86,400 seconds, or pause/resume operations. The complete params object must preserve any other configured tiers and values.

## Queries

The module exposes:

* `Query/Position(position_id)`;
* `Query/PositionsByOwner(owner, pagination)`;
* `Query/RewardState()`; and
* `Query/Params()`; and
* `Query/ValidateState(pagination)` for paged collateral and reward-liability diagnostics.

`Query/Position` includes total whole-unit pending rewards without mutating stored reward debt. `Query/RewardState` reports `U`, `A`, configured rate, actual reward-pool balance, pause state, and potential runway; runway is undefined when the rate is zero and is not a calendar promise while paused or without active shares.

REST paths are rooted at `/terra/ustcstaking/v1/` as defined in the protobuf query service.

Fully settled withdrawn positions are pruned from the primary store and owner index. Withdrawn positions with unpaid rewards remain claimable; after their final claim they are pruned. IDs are never reused, and queries or repeat actions on pruned IDs return not-found.
