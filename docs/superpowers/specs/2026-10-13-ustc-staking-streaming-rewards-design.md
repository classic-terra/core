# USTC staking: funded streaming rewards specification

Status: implementation handoff for the user-approved recommendation. This document specifies future changes; it does not claim they are implemented, tested, or deployed.

Date: 2026-10-13. Repository: `/home/sofoli/core`. Inspected HEAD: `4b0b6a1dcd44649e894167197ec58fc0f0dce781`, **plus the existing uncommitted review-remediation changes**. A clean checkout of HEAD alone is not this baseline. Preserve those changes.

Companion: [implementation plan](../plans/2026-10-13-ustc-staking-streaming-rewards.md).

## 1. Scope and precedence

Implement a governance-funded reserve that releases existing USTC over time. Reuse native `x/ustcstaking` positions, shares, reward index, principal custody, claims, and unbonding. Users become eligible immediately, claim whenever desired, and stop earning when they begin unbonding.

This replaces the instantaneous allocation in `keeper/rewards.go:FundRewards`. It supersedes the seven-day eligibility policy, maturity queue, 1,000-entry processing cap, and one-USTC stake minimum proposed in the 2026-10-08 remediation plan. Do not implement those superseded tasks. Retain independent remediation: immutable governance authority, pool-account normalization, surplus-tolerant custody, error codes, owner indexes, pruning, simulations, and upgrade verification. Adapt dust return to protect the new reserve.

There is no Phase 3. Do not modify LUNC staking, validator power, commissions, slashing, minting, oracle allocation, or distribution rewards. Continue using only the constrained community-pool adapter to fund native USTC rewards. No TreasuryManager, POL, Wasm contract, autonomous community-pool withdrawals, or direct-deposit funding API.

This document fixes implementation choices needed to make the recommendation executable. The actual production release rate and chain upgrade identity remain operator/governance inputs, not invented deployment values. Default release rate is zero, which safely permits implementation and testing without choosing an economic budget.

## 2. User-visible contract

| Topic | Required behavior |
|---|---|
| Stake | Any positive `uusd` amount producing positive shares; at the default 1× tier, minimum is one micro-USTC (`1 uusd`) |
| Eligibility | Immediate for subsequent streamed releases; no age requirement and no historical rewards |
| Launch tier | Fresh defaults contain tier ID 1, duration 21 days, multiplier 1× |
| Other tiers | Retain governance-configurable tiers and immutable per-position snapshots |
| Meaning of tier | Duration is the wait **after starting unbonding**, not a minimum active staking term |
| Claims | Available while active, unbonding, withdrawn, paused, or reserve-empty, for rewards actually earned |
| Unbonding | Allowed anytime; settle earnings through the request time, then remove shares immediately |
| Principal withdrawal | Existing explicit `Withdraw` transaction after maturity; no automatic principal-return queue |
| Funding | Governance may fund before the first stake, during operation, while paused, or after exhaustion |
| Top-up | Adds runway at the existing rate; does not immediately allocate the deposit |
| Exhaustion | Stop new accrual at available reserve; preserve earned claims and exits |
| Restart | Top-up resumes future accrual; never backpay the exhausted interval |
| No stakers | Do not consume reserve or accumulate a payout for the next staker |
| Yield | Variable with total weighted stake; no guaranteed APR and no automatic compounding |

Default 21 days means `1814400s`. Preserve the existing wire field `lock_duration`; label it “unbonding duration” in explanatory text. Fresh defaults may stake before funding, although rate zero produces no new rewards. Imported positions retain their original duration and multiplier even if governance changes or removes their tier.

## 3. Governance controls and defaults

Add `Params.reward_rate_per_day`: a non-null SDK `math.Int` serialized as a string, measured in **uusd per 86,400 seconds for the entire program**. It is not a per-user rate. Default: `0`. For example, `1000000000` means 1,000 USTC/day. This example is not the production default.

Governance updates the rate through the existing full `MsgUpdateParams`. Settle old settings through the transaction's block timestamp before storing new settings. A zero rate stops future accrual while allowing staking. `Paused=true` stops new stakes and future accrual; claims, unbonding, withdrawal, governance top-ups, and parameter updates remain available. On pause/resume, never pay for paused time.

Continue requiring `GetAuthority()` to equal the gov module address; `Params.authority` remains a compatibility field validated against that address. Neither funding nor parameter control may be delegated to an arbitrary wallet.

For this streaming change, retain current positive-duration/positive-multiplier/unique-positive-ID tier validation. Do not silently implement the previously unapproved 32-tier/3,650-day/100× economic limits as part of this handoff. Arithmetic must still be checked against SDK representable limits. A separate tier-limit decision can be added later. At a governance-configured multiplier below 1×, positive-share rounding can require more than `1 uusd`; document this rather than adding a hidden stake minimum.

## 4. Accounting model and protobuf additions

Principal and rewards remain in their existing separate module accounts. Physical reward-pool balance is not synonymous with available funding.

Use these names consistently in Go and protobuf:

| New field | Tag | Type | Meaning |
|---|---:|---|---|
| `Params.reward_rate_per_day` | 6 | `math.Int` | Global emission budget in uusd/day |
| `RewardState.unallocated_rewards` | 3 | `math.Int` | `U`: funded uusd not yet released |
| `RewardState.allocated_rewards` | 4 | `math.Int` | `A`: released or legacy-reserved uusd not yet paid; conservatively includes rounding dust |
| `RewardState.last_reward_time` | 5 | `*time.Time` | Last settled consensus timestamp |
| `RewardState.emission_remainder` | 6 | `math.Int` | Numerator remainder for sub-uusd time-based emission |
| `RewardState.index_remainder` | 7 | `math.Int` | Remainder from division by current total shares |

Retain `reward_index` tag 1 and `total_shares` tag 2. Do not reuse reserved `Params` tag 4. Follow existing `cosmos.Int`, gogoproto customtype/non-null, and yaml annotations. No new position field is required: preserve fractional active rewards using reward debt (section 7).

Define constants for arithmetic:

```text
Q = 10^18                         # LegacyDec reward-index scale
D = 86_400 * 1_000_000_000         # nanoseconds per day
R = reward_rate_per_day           # integer uusd/day
S = total_shares                  # positive integer while earning
I = reward_index                  # existing LegacyDec, conceptually scaled by Q
```

`ReservedRewardsKey` already stores `C`, the sum of integer `Position.claimable_rewards`. Keep it derived through `SetPosition`, `DeletePosition`, and genesis rebuild. **C is not A:** active index-based earnings are not in C. Never use C alone to protect the pool while shares exist.

Fresh state: I=0, S=0, U=0, A=0, both remainders=0. Initialize `last_reward_time` from genesis/upgrade consensus time, never local wall-clock time.

## 5. Streaming algorithm and block timing

Implement a pure calculation plus a keeper wrapper:

```go
func advanceRewardState(state types.RewardState, params types.Params, now time.Time) (types.RewardState, math.Int, error)
func (k Keeper) AccrueRewards(ctx sdk.Context) error
```

The returned integer is the number of uusd released on this call. Neither function visits positions. The pure function must not mutate the caller's SDK integer/decimal backing pointers: use non-mutating operations and copies of `big.Int` values.

Algorithm (all divisions floor nonnegative integers):

```text
validate state and params; missing runtime fields are corruption, not free defaults
if now < last_reward_time: return error, write nothing
if now == last_reward_time: return unchanged state and zero released
elapsed_ns = exact nanosecond difference of consensus timestamps
set returned last_reward_time = now

if paused OR R == 0 OR S == 0 OR U == 0:
    emission_remainder = 0
    return state and zero released

numerator = R * elapsed_ns + emission_remainder
due, new_emission_remainder = divmod(numerator, D)
released = min(due, U)
if released == U:
    new_emission_remainder = 0    # no time debt survives reserve exhaustion

index_numerator = released * Q + index_remainder
index_increment_scaled, new_index_remainder = divmod(index_numerator, S)
I += index_increment_scaled / Q  # construct LegacyDec exactly, never float
U -= released
A += released
store both remainders
return updated state and released
```

If due=0, the index carry can remain unchanged; divmod is still well-defined. Index carry remains valid only while the shareholder cohort is unchanged. After settling and before adding/removing shares, reset `index_remainder=0`. The discarded division remainder was already conservatively protected inside A; it is rounding dust, not spendable reserve. When S becomes zero, also clear `emission_remainder` so later entrants receive no idle-period budget. Preserve emission remainder across positive-to-positive share changes: it describes less than one uusd of unallocated global emission, not an individual entitlement; this avoids an attacker repeatedly resetting the clock using tiny stakes.

Use BeginBlock, before the block's transactions, to call `AccrueRewards`. The app already lists `ustcstaking` in `orderBeginBlockers`; verify membership, do not duplicate it. Retain no-op EndBlock. This pays elapsed time to the shares held before this block's transactions. New stakes begin earning in later time intervals. At the same timestamp, repeated settlement releases zero additional rewards.

Every successful state-changing message listed below also invokes settlement, ensuring correctness in direct keeper tests and governance execution. Settle before modifying shares, rate, pause, or funding. Use a cached context for each operation so bank failures or later validation failures discard settlement and events too; do not assume direct keeper callers provide BaseApp rollback. Flush only on complete success.

`AccrueRewards` checks scalar validity and actual uusd backing `B >= U + A`, then computes in memory and writes once. Unexpected foreign denominations and surplus are inert. No pool transfer occurs during accrual; moving U into A is internal accounting. Every normal block is O(1) in position count. No eligibility queue, timer service, keeper-wide scan, or per-staker bank payment.

## 6. Integer safety and time

No float arithmetic. Use `math/big.Int` for products and divmod intermediates (`R*elapsed_ns`, `released*Q`, `S*Iscaled`, share calculations). Copy operands before mutating them. Validate signs and representable bounds before conversion into SDK Int/LegacyDec; use the pinned `cosmossdk.io/math` limits, not an assumed machine-word size.

Avoid `time.Duration` overflow when calculating elapsed time. Derive nanoseconds from timestamp seconds/nanoseconds using big integers; reject reversed timestamps. Timestamp fields must be valid protobuf timestamps. Missing timestamp is allowed only in a fresh empty default genesis and is normalized by InitGenesis; it is forbidden in initialized runtime state.

Rate is nonnegative and bounded by SDK Int's representable positive range. Admission checks must prevent a stake, funding, or parameter update from creating a future index/debt overflow. Check projected worst-case use of the entire existing reserve plus A with a one-share denominator, using SDK decimal bounds, before committing those transactions. Use checked integer addition for U/A/S and checked share-multiplier conversion. Accrual is capped by U; never construct an SDK Int from an uncapped elapsed-time product. Return an ordinary typed error before mutation for invalid requests. Do not use recover to hide overflow.

Index division carry prevents repeated short blocks from permanently losing emissions when S is large. Intermediate calculations must pass a regression where `S=10^19`, R=10 uusd/day, and ten one-uusd releases produce a positive index increment despite each isolated division rounding to zero.

## 7. Position and message behavior

All amounts below are integer uusd except I, reward debt, and raw pending rewards.

### Stake

Validate owner/coin/tier/positive resulting shares and position-ID availability. In a cache, accrue existing shares first, send principal through the existing bank path, reset index carry, create position with `reward_debt=shares*I`, increase S, and persist. Snapshot the tier duration and multiplier exactly as today. A new position cannot inherit earlier index increases. Do not reset the global emission clock backwards.

### ClaimRewards

In a cache, accrue first. For active positions compute:

```text
pending_exact = shares * I - reward_debt
pending_whole = floor(pending_exact)
pay = pending_whole + explicit_claimable
new_reward_debt = old_reward_debt + pending_whole
new_explicit_claimable = 0
```

For inactive positions, pay only explicit claimable; debt stays zero. Before transfer require `pay <= A`, `B >= U+A`, and valid nonnegative state. Transfer pay from reward pool, subtract pay from A, save/prune, and commit. Do not reset active debt to `shares*I`: that would discard fractional earnings on every claim. Zero-value claims succeed without a bank transfer or loss of fractions. Fractions become payable only at whole-uusd precision.

### BeginUnbonding

In a cache, accrue first, add `floor(shares*I-debt)` to explicit claimable, reset global index carry, remove shares, zero position shares/debt, set status unbonding, set end time to `now + position.LockDuration`, and save. A stays unchanged because earned funds stay owed. No new rewards during unbonding. Clear emission remainder if S becomes zero.

One terminal fraction below one uusd is discarded when an active position ends; it remains conservatively inside A until dust reconciliation. State this explicitly in documentation. Preserving fractions across repeated active claims is mandatory; adding fractional claims to closed positions is outside this minimal design.

### Withdraw

Keep the explicit maturity check and principal-only transfer. Earned rewards remain claimable afterward. Use cached settlement first for consistency. Preserve pruning: remove a withdrawn position only after principal, shares, debt, and integer claimable are zero. Do not remove unpaid positions.

### FundRewards

Require gov authority and a positive uusd coin. Remove both the “no active shares” rejection and funding's pause rejection. In a cache, accrue the OLD reserve first, call the existing community-pool adapter, add the transferred amount to U, preserve A and I as computed by settlement, then commit. No elapsed empty period may use the new deposit. Check additions before transferring.

Existing `ErrNoActiveShares` code may remain for compatibility but this funding path no longer returns it. Funding messages continue carrying only authority and amount; funding never implicitly changes R.

### UpdateParams

Validate authority and complete new params. In a cache, accrue with OLD params, store new params, and commit. This preserves the old rate for the old interval. If entering pause or setting rate zero, clear emission remainder. Index carry may remain because shares have not changed. Tier edits only affect future positions.

### ReturnRewardDust

Keep governance-only and require S=0. After settlement, compute C from the existing derived reserve, require `A >= C` and `B >= U+A`. Reconcile `A=C` in the cache; no active index liability remains. Maximum return is `B-U-C`. Require requested amount <= maximum and transfer only through the existing return adapter. Commit A reconciliation only if the operation succeeds. U is never returnable through this dust API, even when the module is paused or the release rate is zero. No new reserve-cancellation API.

## 8. Invariants and queries

Required bank-backed checks:

```text
principal_pool_uusd >= sum(outstanding position principals)
S == sum(active position shares)
C == sum(explicit integer claimable rewards)
L == C + sum(floor(shares*I-debt) for active positions)
L_exact == C + sum(shares*I-debt for active positions)
0 <= L <= L_exact <= A
reward_pool_uusd >= U + A
0 <= emission_remainder < D
S > 0 => 0 <= index_remainder < S
S == 0 => index_remainder == 0
```

U, A, R, S, I, all remainders and liabilities must be initialized and nonnegative. Validate status/debt/snapshots as before. Bank surplus must not break invariants and must not become U automatically. Keep exact derived C validation in keeper invariants. Explicit whole-state validation may iterate positions; block processing and funding may not.

Extend existing RewardState response, not a new service:

| Response field | Tag | Type/meaning |
|---|---:|---|
| `reward_state` | 1 | Existing, now contains reserve/accounting fields |
| `reward_rate_per_day` | 2 | Same Int convention as params |
| `reward_pool_balance` | 3 | `Coin`, actual uusd bank balance |
| `paused` | 4 | bool |
| `runway_seconds` | 5 | Int string; floor(U*86400/R) if R>0, otherwise zero |
| `runway_defined` | 6 | bool, R>0; estimate assumes active, unpaused operation |

When S=0 or paused, runway is potential active runtime, not a calendar exhaustion promise. A is a conservative reserve for released rewards, not the exact sum of currently withdrawable integer claims. Call it “allocated/protected rewards” in user displays. Never label B as “available rewards.”

Add `QueryPositionResponse.pending_rewards` tag 2 (non-null Coin): total whole-uusd claim available at the queried state. Preserve stored Position unchanged. Owner-list clients can combine returned positions and RewardState at one pinned height; avoid an unbounded aggregate query.

Queries return committed state and never settle or write. BeginBlock keeps state current at block timestamps; do not project against machine wall time. `ValidateState` remains capped at 500 positions per page with height pinning. Add page `claimable_liability` string tag 10, `reserved_rewards` Int tag 11, and `reward_liability_exact` decimal string tag 12 to support full aggregation. CLI aggregates L/L_exact/C/S and compares with A/U/B and the stored derived reserve, not just B>=L. Check exact decimal liabilities as well as whole claimable coins, so several positions with fractional rewards cannot hide an underbacked import. Change the CLI's existing strict principal equality to `>=`, consistent with surplus-tolerant keeper invariants. Invalid CLI validation must return a nonzero exit status, not merely print `valid:false` and exit successfully.

## 9. Genesis, migration, and upgrade

Bump module consensus version from 1 to 2 and register a v1-to-v2 migration. Use a keeper migrator, tested independently. Keep old protobuf tags so old values decode, but normalize missing new fields only in migration or explicit legacy import, never silently in runtime getters.

Fresh chain/new-module activation: use default tier 1 (21 days, 1×), R=0, U=A=0, zero remainders, and consensus timestamp. Existing account-normalization protections run before new-module genesis. Do not assume prefunded account balances are approved funding.

Existing v1 state: preserve every position, index, total shares, claimable balance, next ID, tier snapshot, and existing params (add R=0). Set U=0. Conservatively set A to the existing reward-pool uusd bank balance, require it covers all existing exact decimal reward liabilities, initialize time to upgrade block time and remainders to zero, and rebuild derived state. This protects even already-earned fractional claims. Do not reinterpret old bank balances as new stream funding. If no shares remain, dust return can later reconcile A to C.

Validate legacy solvency before committing the conversion; pool balance must cover existing liabilities, principal pool must cover principal, and active shares must match. Missing funds are an actionable migration error; never mint or seize principal to fill a gap.

For current-schema export/import: serialize all RewardState fields and retain the prior settled timestamp. At the next block, elapsed consensus time accrues against imported shares and reserve, capped by U. Thus a restart/import gap follows the same elapsed-time rule as a delayed block; it does not fabricate funds. This is intentional. A default empty genesis with no timestamp initializes at genesis time. Reject future timestamps and incomplete nonempty current-schema state.

Direct v1 JSON import into the v2 binary is outside scope. Document the supported conversion route: import the old export into its compatible v1 binary on an isolated chain, execute the tested v1-to-v2 upgrade, then export using v2. Never use this procedure to start a second copy of a live validator. Do not accept ambiguous partially populated streaming fields as legacy. Migration tests must encode genuine v1 protobuf bytes, rather than construct fully initialized v2 fixtures and call them legacy.

The inspected activation candidate is named `v14_5`; that is not proof it is the correct live upgrade name. The existing handler accepts only existing module version 1 and the StoreUpgrades list unconditionally adds the store. Adapt registration/loader for two separate cases: absent module adds store; existing module migrates without adding it again. Reject unsupported future module versions. Maintainers must select an unapplied chain plan and verify predecessor/version-map state before release; implementation/tests use isolated plans and do not deploy.

## 10. Events and docs

Keep existing message events. Add `ustcstaking_rewards_accrued` only when released>0, with released amount, timestamp, index, U and A. Funding events include funded amount and resulting U; their index fields may reflect settlement of the old reserve but must not imply the deposit itself was allocated. Parameter events include R and pause. Dust events include actual return amount and preserved U/C.

Update protobuf-generated Go/gateway, Swagger and statik; keeper/module specs; operator/localnet guides; executive summary and audience briefings. Mark older lump-sum, age-queue, and “funding requires active shares” instructions superseded. Do not claim APR, automatic principal return, or a second waiting period before unbonding. No changes to historical archived documents are required beyond pointing current entry points to this spec.

## 11. Acceptance examples (integer amounts are uusd)

| ID | Setup/action | Expected result |
|---|---|---|
| E1 | At t0, fund 1000 with S=0; advance one day | U=1000, A=0, I=0 |
| E2 | A stakes 100 shares; R=100/day; advance one day | U decreases 100; A increases 100; A's pending=100 |
| E3 | At day 1 B stakes 100; advance another day | A pending=150, B pending=50, total A ledger=200 |
| E4 | Claim A's 150 at day 2 | A ledger=50; U unchanged; A cannot claim 150 again |
| E5 | A unbonds at day 2; next day releases 100 | B gets all 100; A earns zero afterward; A's principal withdrawable at day 23 |
| E6 | U=50,R=100/day; advance day, later top up 200 | Only 50 released before top-up; zero backpay; next day releases 100 |
| E7 | R=100/day for 12h then R=200/day for 12h | Total release=150 |
| E8 | R=100/day, pause after 12h, resume after 2 days | First 50 earned; pause adds zero; following 12h adds 50 |
| E9 | U=100,A=40,C=40,S=0,B=150; request dust return 11 | Reject; maximum return=10; U and claims protected |
| E10 | Two equal shares, R=3/day; claim every day vs once after 2 days | Each can receive total 3 after 2 days; frequent claiming does not destroy 0.5 fractions |
| E11 | One owner 1 share; R=1/day; 24 hourly accruals | One full uusd emitted, equal to one daily accrual |
| E12 | Same-time stake, claim, top-up, params changes | No extra elapsed-time rewards; no retroactive assignment |
| E13 | Failed community-pool transfer after settlement would advance state | All state, balances, and events roll back |
| E14 | Owner 100 shares unbonds immediately after staking | Valid; zero earned; withdrawal fails before 21 days, succeeds at equality |
| E15 | Donation increases B but not approved U | No change to stream rate/reserve; invariants remain valid |

Also require weighted tiers, tiny stake, no-active/paused funding, unauthorized calls, query immutability, genesis/restart equivalence, precise reserve exhaustion, index-carry large-share case, SDK overflow boundaries, genuine legacy migration, real-bank supply/community-pool reconciliation, and all four restored simulation jobs. The implementation plan specifies execution order and expected evidence.
