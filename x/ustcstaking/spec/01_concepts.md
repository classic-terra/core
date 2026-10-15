<!--
order: 1
-->

# Concepts

## Position lifecycle

Each deposit creates an independent position:

```text
stake -> active -> begin unbonding -> matured -> withdraw
                         |
                         +-> claim rewards (available according to position state)
```

`Stake` transfers `uusd` from the user to the principal module account. `BeginUnbonding` settles rewards earned to that point and removes the position's shares from active reward accounting. `Withdraw` becomes valid only after the recorded lock duration has elapsed and returns the recorded principal. A withdrawn position cannot be reused.

The lock duration is copied into the position at stake time. Later governance changes to a lock tier do not change existing positions.

## Shares and lock tiers

Governance configures lock tiers. Each tier has:

* a unique numeric ID;
* a positive lock duration; and
* a positive share multiplier.

For a deposit amount `A` and tier multiplier `M`, the position receives:

```text
shares = truncate(A * M)
```

The multiplier is also stored on the position as an immutable accounting snapshot. The implementation uses fixed-point decimal arithmetic and truncates fractional shares; the resulting rounding dust is not paid as rewards.

## Reward accounting

The module maintains a global reward index and total active shares. Funding is explicit: governance debits the distribution community pool and transfers the same USTC into the reward-pool module account through the application adapter.

For a position with shares `S`, reward index `I`, and position reward-debt snapshot `D`:

```text
accrued = truncate(S * (I - D))
```

Governance funding moves existing `uusd` into an unallocated reserve `U`; it never changes the reward index immediately. A governance-configured rate `R` releases at most `R` `uusd` per 86,400 seconds across all active shares. BeginBlock uses consensus block time to move released value from `U` to allocated/protected rewards `A` and advances the index. Rewards are settled before a stake joins or shares leave, so new positions receive no rewards from before they joined and unbonding positions stop earning immediately.

The module tracks accounting in fixed-point values but pays whole `uusd` coins. A claim pays its whole-unit entitlement and retains any active fractional entitlement in reward debt, so frequent claims do not erase fractions. On unbonding, the fractional part is terminal and is conservatively retained as rounding dust. `A` protects released rewards and legacy funds; it is not a user-facing exact claim total.

## No-minting model

This module differs from classic Terra monetary modules:

* staking deposits are transferred into a principal pool;
* reward funding is transferred into a separate reward pool;
* no module account has minting or burning permissions;
* withdrawal returns principal only; and
* reward claims cannot exceed the reward-pool balance.

This makes the reward liability externally funded and bounded by available USTC. If the reward pool is insolvent for a claim, the claim fails without changing the position's accounting.

## Authority and pause controls

The governance authority can replace the complete parameter set, including lock tiers, the daily release rate, and the pause flag, and is the only authority that can fund rewards from the community pool. When paused, new staking and reward emission stop; governance can still fund the unallocated reserve, and existing positions can unbond, withdraw after maturity, or claim released rewards. Governance parameter updates remain available so the module can be resumed. The default tier is a 21-day lock with a 1× multiplier, and the default release rate is zero.
