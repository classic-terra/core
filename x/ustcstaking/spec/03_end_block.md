<!--
order: 3
-->

# BeginBlock and block timing

## Constant-time reward release

At each BeginBlock, the module settles the global reward reserve using the block's consensus timestamp. It does not scan positions or pay users. If the module is unpaused, active shares exist, the daily rate is positive, and `U` is nonzero, it releases the accrued whole `uusd` amount up to the remaining reserve. The release advances the cumulative reward index; users claim separately at any time.

The rate is `uusd` per 86,400 seconds for all stakers combined. Emission and index division remainders are stored, making frequent block settlement equivalent to a single settlement over the same elapsed time. Pause, zero rate, no active shares, and an exhausted reserve do not accrue a future backlog. Funding and parameter changes settle the previous state first, so those changes are not retroactive.

No user transfer occurs during BeginBlock. `Withdraw` remains explicit and checks maturity against the consensus block timestamp; a matured position stays `UNBONDING` until its owner submits it. The module retains a no-op EndBlock for the SDK module interface.
