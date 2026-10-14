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
* `total_shares`: sum of shares for active positions.

The reward index is unchanged when there are no active shares; funding without active shares is rejected rather than creating unallocated rewards.

## Parameters

```text
0x30 -> Params
```

`Params` contains the fixed bond denomination, lock tiers, governance authority, and pause flag. Protobuf field 4 and the removed `funding_authority` name are reserved. Genesis validation requires non-negative accounting values, valid owners, valid coin denominations, unique position IDs, and equality between active position shares and `RewardState.total_shares`.

Custody validation requires USTC balances to cover liabilities; unsolicited USTC surplus and unrelated denominations never create rewards. Active-share ledger equality remains exact. The reserved-rewards aggregate equals all integer claimables, including frozen claims on withdrawn positions. Governance may return only unreserved reward-pool surplus when total active shares are zero.

The paginated validate-state query returns USTC collateral balances only. It does not enumerate inert foreign-denomination coins; use the bank module's paginated balance queries to inspect those separately.
