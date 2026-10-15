<!--
order: 2
-->

# State

## Positions

Positions are keyed by numeric ID. The position stores:

| Field | Meaning |
|---|---|
| `id` | Unique position ID. |
| `owner` | Bech32 account address. |
| `principal` | Original positive `uusd` deposit, returned on withdrawal. |
| `shares` | Active reward shares; zero after unbonding begins. |
| `lock_tier_id` | Tier selected when the position was created. |
| `share_multiplier` | Multiplier snapshot used to derive shares. |
| `reward_debt` | Global reward-index snapshot for settled accounting. |
| `unbonding_end_time` | Maturity time after unbonding begins. |
| `status` | `ACTIVE`, `UNBONDING`, or `WITHDRAWN`. |
| `claimable_rewards` | Rewards settled for later claim. |
| `lock_duration` | Immutable duration snapshot used for maturity. |

Storage prefixes:

```text
0x10<position_id>           -> Position
0x11<len(owner)><owner><position_id> -> owner lookup namespace
0x12<unbonding...>           -> unbonding lookup namespace
0x13                        -> next position ID
```

The owner index is keyed by a one-byte address-length prefix, owner bytes, and an eight-byte big-endian position ID. Queries use this index; primary position records remain authoritative. Index entries and the reserved-claimable aggregate are rebuilt from primary positions during genesis import.

## Reward state

```text
0x20 -> RewardState
0x21 -> derived reserved integer claimable rewards
```

`RewardState` contains:

* `reward_index`: cumulative fixed-point rewards per active share;
* `total_shares`: sum of shares for active positions;
* `unallocated_rewards` (`U`): governance-funded `uusd` not yet released;
* `allocated_rewards` (`A`): released or legacy-protected rewards, including conservative rounding reserve;
* `last_reward_time`: last consensus timestamp included in settlement;
* `emission_remainder`: sub-`uusd` elapsed-rate carry; and
* `index_remainder`: division carry for the current share cohort.

Funding increases `U`, never the index. BeginBlock releases `min(U, floor((R × elapsed_nanoseconds + emission_remainder) / 86400000000000))` when unpaused and shares are active. It moves the released amount from `U` to `A` and advances the index. Empty-share and paused intervals do not accumulate emission backlog. Index remainder resets when the active share cohort changes; emission remainder survives positive-share changes and clears when shares reach zero, paused, rate zero, or exhausted.

## Parameters

```text
0x30 -> Params
```

`Params` contains the fixed bond denomination, lock tiers, immutable governance authority, pause flag, and `reward_rate_per_day` (`R`) in `uusd` per 86,400 seconds. The production default rate is zero; the default tier is 21 days and 1×. Protobuf field 4 and the removed `funding_authority` name are reserved. Genesis validation requires non-negative accounting values, valid owners, valid coin denominations, unique position IDs, and equality between active position shares and `RewardState.total_shares`.

Custody validation requires principal balance `>=` principal liability, exact reward liability `<= A`, and reward-pool balance `B >= U + A`; unsolicited USTC surplus and unrelated denominations never create rewards. Active-share ledger equality remains exact. The derived reserve `C` equals all integer claimables, including frozen claims on withdrawn positions. Governance may return only `B - U - C` when total active shares are zero, and dust return reconciles `A` to `C`.

The paginated validate-state query returns USTC collateral balances only. It does not enumerate inert foreign-denomination coins; use the bank module's paginated balance queries to inspect those separately.
