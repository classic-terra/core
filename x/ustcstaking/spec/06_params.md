<!--
order: 6
-->

# Parameters

| Key | Type | Current default/constraint |
|---|---|---|
| `bond_denom` | string | Must be `uusd`. |
| `lock_tiers` | `[]LockTier` | Empty by default; governance must configure a tier before staking is enabled. |
| `authority` | address | Fixed governance module address; read-only compatibility field, not an authority-rotation control. |
| `paused` | bool | `false` by default. |

Each `LockTier` has:

| Key | Type | Constraint |
|---|---|---|
| `id` | uint32 | Unique within the parameter set. |
| `duration` | duration | Positive. |
| `multiplier` | decimal | Positive fixed-point value. |

There is intentionally no funding-authority parameter, inflation rate, mint allowance, reward emission schedule, or automatic funding parameter. Governance may fund rewards only from the distribution community pool through `MsgFundRewards`; the application adapter debits both FeePool accounting and the distribution module balance before crediting the reward pool. This transfer does not mint USTC.

Privileged messages authenticate against the keeper's immutable governance module address. Parameter validation rejects any alternate authority, including a wallet address.
