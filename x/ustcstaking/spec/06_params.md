<!--
order: 6
-->

# Parameters

| Key | Type | Current default/constraint |
|---|---|---|
| `bond_denom` | string | Must be `uusd`. |
| `lock_tiers` | `[]LockTier` | Tier 1 defaults to 21 days with a 1× multiplier. |
| `authority` | address | Fixed governance module address; read-only compatibility field, not an authority-rotation control. |
| `paused` | bool | `false` by default. |
| `reward_rate_per_day` | `math.Int` | `0 uusd` per 86,400 seconds by default; governance may set any valid non-negative rate. |

Each `LockTier` has:

| Key | Type | Constraint |
|---|---|---|
| `id` | uint32 | Unique within the parameter set. |
| `duration` | duration | Positive. |
| `multiplier` | decimal | Positive fixed-point value. |

There is intentionally no funding-authority parameter, mint allowance, or automatic funding mechanism. `reward_rate_per_day` controls how quickly the already-funded reserve is released across all active positions; it is not an APR for each user. Governance may fund rewards only from the distribution community pool through `MsgFundRewards`; the application adapter debits both FeePool accounting and the distribution module balance before crediting the reward pool. Funding does not mint USTC and may occur before staking, while paused, or after reserve exhaustion.

Privileged messages authenticate against the keeper's immutable governance module address. Parameter validation rejects any alternate authority, including a wallet address.
