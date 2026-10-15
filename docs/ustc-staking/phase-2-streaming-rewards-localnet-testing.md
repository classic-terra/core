# USTC Streaming-Rewards Seven-Validator Localnet Campaign

This campaign validates governance-funded USTC streaming on a disposable
seven-validator Terra Classic localnet. It complements the implementation
plan's unit, real-keeper, migration, and simulation tests; it does not approve
activation or replace the operator-led upgrade rehearsal. It does not change
LUNC staking, validator power, or distribution rewards.

The previous [Phase 2 campaign](phase-2-community-pool-localnet-testing.md) is
retained as historical evidence. Its instant-allocation, no-active-share
funding rejection, paused-funding rejection, and empty-default-tier
expectations are obsolete and must not be used.

## Candidate and baseline

Record the exact source commit and binary hash, chain ID, seven validator
heights, block interval, test start/end time, and evidence directory. Start
from fresh genesis and wait for all seven nodes to produce blocks. Snapshot
the USTC total supply, community pool, distribution account, both USTC staking
module accounts, params, reward state, positions, owner queries, and validator
state. The default tier is 21 days at 1×; the default daily reward rate is
zero. Both staking module accounts have no mint or burn permission.

## Configure the disposable test economy

Submit a governance `MsgUpdateParams` proposal using the complete current
Params object. Preserve the live authority and every configured tier. For a
fresh localnet, an illustrative short tier and test rate are:

```json
{
  "bond_denom": "uusd",
  "lock_tiers": [
    {"id": 1, "duration": "30s", "multiplier": "1.000000000000000000"},
    {"id": 2, "duration": "60s", "multiplier": "2.000000000000000000"}
  ],
  "authority": "<gov module address queried from this chain>",
  "paused": false,
  "reward_rate_per_day": "86400000"
}
```

Here `86400000` means 86,400,000 `uusd` per 86,400 seconds for the whole
program, or 1,000 `uusd` each second. It is a test value, not a recommended
production budget. A governance proposal replaces the full Params object, so
operators must query and preserve any tier or setting not being changed.

Seed the distribution community pool through its normal localnet procedure.
Then, before any account stakes, execute a governance `MsgFundRewards` proposal
for a bounded amount, for example `300000000uusd`. Verify the community pool
decreased by exactly the funded integer amount, the reward-pool bank balance
increased by that amount, total supply and principal stayed unchanged, `U`
increased by the amount, `A` and the reward index did not increase from the
deposit, and the funding event reports the resulting `U`.

## Observe streamed eligibility and accounting

Record block height/time, `U`, `A`, index, total shares, pool balance, and both
users' pending rewards before each lifecycle operation. Stake validator 0's
test account, let at least one block advance, then stake validator 1's account.
The first staker receives no rewards for time before its stake; the second
staker receives no rewards for time before its stake. After later blocks,
confirm `U` falls and `A` and the index rise only according to elapsed
consensus time and the active-share weights. Do not infer accrual from wall
clock sleeps; use block timestamps and query the chain.

Claim one position after a release and claim it repeatedly across later
releases. Compare cumulative whole-unit rewards with the same position claimed
once later; active fractional rewards must survive earlier claims. Confirm
queries report pending whole rewards without changing stored reward debt or
other state. Confirm all nodes return the same query result at a pinned height.

Change the daily rate through governance after observing a partial interval.
The old rate must apply through the proposal's execution block; the new rate
applies only afterward. Repeat with pause and resume: no rewards accrue while
paused, the paused interval is not back-paid, and the resumed rate starts from
the resumed consensus time. While paused, execute a valid funding proposal and
confirm `U` rises without release. With no active shares, funding must also
succeed and remain unallocated until a position becomes active.

Test reserve exhaustion with a small `U` and an elapsed interval long enough to
consume it. Confirm the release is capped at `U`, the emission carry clears,
and later time alone releases nothing. Fund again after exhaustion and verify
the top-up receives no historical backpay. Test unauthorized, wrong-denom,
zero, and community-pool-underfunded proposals; each failure must leave
balances, state, events, and FeePool accounting unchanged.

## Exit, dust, restart, and final reconciliation

Have one user begin unbonding immediately after staking and another unbond
after earning. Confirm the first earns zero, while the second freezes its
earned amount. Neither continues earning after its shares leave. Verify
withdrawal fails before maturity and succeeds at the recorded maturity time;
the position uses its snapshotted duration even if governance later changes
the tier. Claim frozen rewards before and after withdrawal. When all shares are
zero, test governance dust return at the exact permitted maximum and one unit
above it. The returned amount must exclude `U` and `C`; existing claims remain
payable.

Stop and restart one validator with its data intact, then stop/restart the
whole network without deleting homes. Compare heights and state after recovery.
Before/after every critical action, reconcile USTC supply, principal custody,
reward bank balance `B`, unallocated `U`, allocated/protected `A`, exact
liability, claim reserve `C`, community-pool decimal accounting, and
distribution-module balance. Require `B >= U + A`, exact position liability
`<= A`, principal balance `>=` principal liability, and `C` equal to summed
whole-unit claims. Any donation may increase `B` but must not change `U`, `A`,
the index, or user pending rewards.

Record every proposal ID and terminal result, transaction hash and full result,
block height/time, events, raw query responses, node logs, restart outcome, and
the exact candidate hash. Report failed or unrun scenarios separately. The
old-state-to-v2-binary upgrade and migration rehearsal remains a separate
release gate.
