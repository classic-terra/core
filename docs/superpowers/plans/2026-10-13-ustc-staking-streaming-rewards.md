# USTC Streaming Rewards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. The author of this handoff is not implementing it. Follow the user's later execution instructions; do not deploy, open another PR, or discard existing changes automatically.

**Goal:** Replace immediate funding allocation with a governance-funded reserve that releases USTC over time, while preserving immediate staking eligibility, claims, configurable unbonding tiers, and separate principal custody.

**Architecture:** Keep the cumulative reward index and existing position ledger. Track unallocated reserve separately from allocated/protected rewards, accrue a global daily rate using consensus elapsed time, and settle before any change in shares, funding, or parameters. Use BeginBlock and constant work per settlement; no per-position activation queue.

**Tech Stack:** Go, Cosmos SDK v0.53.6, cosmossdk.io/math, protobuf/gogoproto/Buf, existing native bank/community-pool adapter, Go tests, SDK simulations.

---

## Read first and working rules

The authoritative economic behavior, field tags, formulas, and acceptance vectors are in [the design specification](../specs/2026-10-13-ustc-staking-streaming-rewards-design.md). Read it fully. Do not infer economics from old tests: several deliberately assert the behavior being replaced.

Read AGENTS.md and its nine baseline documents in order. Historical Phase 3 instructions are canceled. Then read `docs/superpowers/plans/2026-10-08-ustc-staking-review-remediation.md`; its age-queue and minimum-stake tasks are superseded by this plan, while its independent security/CI fixes remain required.

Inspected baseline: HEAD `4b0b6a1dcd44649e894167197ec58fc0f0dce781` plus a substantial dirty remediation patch. In particular, current `position.go` has the derived claim reserve and pruning, `msg_server.go` has ReturnRewardDust, `module.go` has simulation operations, and the upgrade handler has account normalization. **Do not start from clean HEAD and accidentally omit those changes.** Record the dirty diff, preserve unrelated edits, and create a checkpoint only under the user's normal Git workflow. Do not `git reset`, `git clean`, or blindly stage all files.

Before code implementation, compare `graphify-out/coding/manifest.json` with HEAD; refresh via `scripts/graphify-refresh coding` if required. Use graph/wiki for navigation and source for evidence. Keep `graphify-out/` local and ignored. This documentation handoff does not regenerate the graph.

Implementation progress is not production readiness. Tests and validation commands below are work for the implementing model; none were run as part of writing this handoff.

## Defaults and explicit non-goals

```text
fresh default tier: id=1, duration=1814400s, multiplier=1.000000000000000000
fresh release rate: 0 uusd/day (governance configures production amount)
minimum new stake: no extra floor; positive uusd and positive resulting shares
minimum reward age: none
eligibility queue: none
paused: blocks new stakes and emission; permits top-ups and exits/claims
principal return: explicit Withdraw after maturity
rate unit: uusd per 86400 seconds for all stakers combined
```

Preserve configurable tiers, current positive-value validation, and historical snapshots. Do not add fixed APR, automatic compounding, automatic principal release, treasury contracts, an arbitrary funding wallet, or validator features. Do not implement previously proposed hard tier limits without a separate decision. Guard SDK arithmetic regardless of those limits.

## File map

| Files | Responsibility |
|---|---|
| `proto/terra/ustcstaking/v1/ustcstaking.proto` | Rate and streaming state fields |
| `proto/terra/ustcstaking/v1/query.proto` | Pool information and pending rewards |
| `x/ustcstaking/types/{params,genesis,errors}.go` | Defaults, validation, added error if needed |
| `x/ustcstaking/types/*.pb.go`, `*.pb.gw.go` | Generated artifacts only |
| NEW `x/ustcstaking/keeper/streaming.go` | Pure state transition, checked math, cached settlement |
| `x/ustcstaking/keeper/rewards.go` | Funding semantics; pending reward helper |
| `x/ustcstaking/keeper/msg_server.go` | Transaction settlement order, claims, reserve protection |
| `x/ustcstaking/keeper/position.go` | Preserve reserve/index/pruning behavior |
| `x/ustcstaking/module.go` | BeginBlock interface, consensus version, migration registration |
| `x/ustcstaking/keeper/{invariants,query_server}.go` | Reserve-aware checks and read APIs |
| `x/ustcstaking/client/cli/query.go` | Pinned aggregate validation and error exit |
| `x/ustcstaking/genesis.go` | Initialize/export all streaming state |
| NEW `x/ustcstaking/keeper/migrations.go` | Version 1 to 2 migration |
| `app/upgrades/ustc_staking/{upgrades,constants}.go`, `app/app.go` | Existing handler/loader adaptation where required |
| `x/ustcstaking/simulation/{genesis,operations,proposals,decoder}.go` | Time-based scenarios and new params |
| `app/ustc_staking_integration_test.go`, `app/ustc_staking_upgrade_test.go`, `app/sim_test.go` | Real-keeper and activation coverage |
| `client/docs/swagger-ui/swagger.yaml`, `client/docs/statik/statik.go` | Generated API docs |

No funding-adapter rewrite should be needed. Its existing source is `app/keepers/community_pool_adapter.go`; test atomic community-pool bookkeeping and preserve its authority boundaries.

## Task 1: Check the baseline and establish meaningful fixtures

- [ ] Record `git status --short`, `git rev-parse HEAD`, and existing diffs for the files in the map. Inspect the exact installed math module, app SDK interfaces, and generated field conventions before copying snippets.
- [ ] Read `keeper/test_utils_test.go`. Its current fake bank records transfers but does not debit reward balances on claims. Correct the fake to track principal inflows/outflows and reward outflows, reject insufficient funds, and offer injected bank failures. Never use that fake alone to prove rollback: mutable Go fake fields do not roll back with SDK CacheContext. Atomicity tests use real bank/distribution keepers or KV-backed fakes.
- [ ] Add deterministic test time: `time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)`. Replace time.Now in the new streaming fixture; do not change global time helpers unnecessarily.
- [ ] Introduce one reusable fixture that initializes Params, RewardState, bank balance, and next-position ID. New fields must be explicit zero values, not nil `math.Int`s. The fixture is a test utility, not a runtime normalization path.

Future fixture initialization, after Task 2 generates fields:

```go
params := types.DefaultParams()
params.RewardRatePerDay = math.NewInt(100)
k.SetParams(ctx, params)
now := ctx.BlockTime()
k.SetRewardState(ctx, types.RewardState{
    RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt(),
    UnallocatedRewards: math.ZeroInt(), AllocatedRewards: math.ZeroInt(),
    LastRewardTime: &now, EmissionRemainder: math.ZeroInt(),
    IndexRemainder: math.ZeroInt(),
})
k.SetNextPositionID(ctx, 1)
```

**Checkpoint:** baseline provenance recorded; no claim that legacy instant-funding tests establish streaming correctness.

## Task 2: Add schema, defaults, and pure validation

**Files:** protobuf source, types/params.go, types/genesis.go, corresponding type tests.

- [ ] Add spec section 4 fields with the exact unused tags. Example Int field syntax:

```protobuf
string reward_rate_per_day = 6 [
  (cosmos_proto.scalar) = "cosmos.Int",
  (gogoproto.customtype) = "cosmossdk.io/math.Int",
  (gogoproto.nullable) = false,
  (gogoproto.moretags) = "yaml:\"reward_rate_per_day\""
];
```

Use the same annotations for RewardState Int fields. Time field:

```protobuf
google.protobuf.Timestamp last_reward_time = 5 [
  (gogoproto.stdtime) = true,
  (gogoproto.moretags) = "yaml:\"last_reward_time\""
];
```

- [ ] Extend QueryRewardStateResponse and QueryPositionResponse exactly as spec section 8, plus QueryValidateStateResponse fields 10/11/12. Import `cosmos_proto/cosmos.proto` into query.proto if adding custom Int annotations there. Existing response tags are unchanged.
- [ ] Generate code with `make proto-gen`. Confirm expected fields appear with `rg -n 'RewardRatePerDay|UnallocatedRewards|IndexRemainder' x/ustcstaking/types/*.pb.go`. Inspect `scripts/protocgen.sh` if the command generates no USTC output; do not hand-edit generated code. Use the repository's pinned container/tool versions. Inspect generated diff for unrelated churn.
- [ ] Change DefaultParams to add rate zero and one tier:

```go
duration := 21 * 24 * time.Hour
// In the returned Params literal:
LockTiers: []LockTier{{Id: 1, Duration: &duration, Multiplier: math.LegacyOneDec()}},
RewardRatePerDay: math.ZeroInt(),
```

- [ ] Add rate validation: nil or negative is invalid; zero and positive representable values are valid. Preserve existing authority and tier checks. SDK decode and checked arithmetic must reject out-of-range values without a process panic.
- [ ] DefaultGenesis initializes U/A/remainders to zero; LastRewardTime may be nil only for truly empty default state. Genesis validation rejects partial/malformed streaming state and requires both whole and exact decimal liabilities <=A. Runtime validator takes consensus time separately to reject a future timestamp.
- [ ] Add tests: default 21-day/1×/zero-rate, nil/negative rate, positive-rate units, empty default timestamp, missing timestamp with funds/positions, negative U/A/remainders, invalid remainder ranges, unchanged historical snapshots.

```bash
go test ./x/ustcstaking/types -run 'Test.*(Params|Genesis|Streaming)' -count=1
```

Expected: selected tests run and pass. Keep a temporary task-local compile failure only while the schema and dependent fixtures are being updated; do not skip nil-state checks to make old fixtures pass.

## Task 3: Implement the pure streaming calculation

**Create:** `keeper/streaming.go`, `keeper/streaming_test.go`, `keeper/arithmetic_bounds_test.go`.

- [ ] First write deterministic tests for E1/E2/E6/E7/E8/E11/E12 and the large-share carry case from the spec. Test `advanceRewardState` directly before wiring bank/message behavior.
- [ ] Implement the exact signatures in spec section 5. Keep math in the pure helper, storage and backing checks in AccrueRewards. Do not call bank sends or iterate positions here.
- [ ] Implement big-integer floor division without aliasing inputs. This is the arithmetic core to use, with validated nonnegative operands:

```go
// stdmath aliases the standard math/big package to distinguish SDK math.
numerator := new(stdmath.Int).Mul(rate.BigInt(), elapsedNS)
numerator.Add(numerator, emissionCarry.BigInt())
due, nextCarry := new(stdmath.Int), new(stdmath.Int)
due.QuoRem(numerator, stdmath.NewInt(86_400_000_000_000), nextCarry)
// Clamp due against U as big.Int BEFORE converting into SDK math.Int.
releasedBig := new(stdmath.Int).Set(due)
if releasedBig.Cmp(unallocated.BigInt()) >= 0 {
    releasedBig.Set(unallocated.BigInt())
    nextCarry.SetInt64(0)
}
indexNumerator := new(stdmath.Int).Mul(releasedBig, stdmath.NewInt(1_000_000_000_000_000_000))
indexNumerator.Add(indexNumerator, indexCarry.BigInt())
deltaScaled, nextIndexCarry := new(stdmath.Int), new(stdmath.Int)
deltaScaled.QuoRem(indexNumerator, totalShares.BigInt(), nextIndexCarry)
// Convert deltaScaled using LegacyNewDecFromBigIntWithPrec(deltaScaled, 18)
// only after checking the pinned SDK decimal bounds and the resulting index.
```

- [ ] Use exact seconds/nanoseconds difference in big integers rather than `now.Sub(last).Nanoseconds()` for unbounded intervals. Check `now.Before(last)` first. No UTC wall-clock calls.
- [ ] Handle all idle branches by advancing LastRewardTime and clearing EmissionRemainder. Keep IndexRemainder except where S=0, which requires it be zero. Same-time calls are identity operations.
- [ ] Check initialized scalar state, emission/index remainder bounds, SDK Int conversions, decimal index additions, share/debt products, and worst-case future reserve allocation before unsafe SDK operations. Use the pinned SDK source to implement checked conversion helpers; add exact boundary and one-past-boundary tests. Do not rely on `recover` or floating point.
- [ ] AccrueRewards: read params/state, validate B>=U+A, calculate, store, and emit `ustcstaking_rewards_accrued` only for positive release. Return errors without partial writes.

Sample pure test with explicit expected state:

```go
func TestStreamingOneDay(t *testing.T) {
    start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
    p := types.DefaultParams()
    p.RewardRatePerDay = math.NewInt(100)
    s := types.RewardState{
        RewardIndex: math.LegacyZeroDec(), TotalShares: math.NewInt(100),
        UnallocatedRewards: math.NewInt(1000), AllocatedRewards: math.ZeroInt(),
        LastRewardTime: &start, EmissionRemainder: math.ZeroInt(), IndexRemainder: math.ZeroInt(),
    }
    got, released, err := advanceRewardState(s, p, start.Add(24*time.Hour))
    require.NoError(t, err)
    require.Equal(t, math.NewInt(100), released)
    require.Equal(t, math.NewInt(900), got.UnallocatedRewards)
    require.Equal(t, math.NewInt(100), got.AllocatedRewards)
    require.True(t, got.RewardIndex.Equal(math.LegacyOneDec()))
    require.True(t, s.RewardIndex.IsZero(), "input must not be mutated")
}
```

```bash
go test ./x/ustcstaking/keeper -run 'TestStreaming|TestArithmetic' -count=1
```

**Checkpoint:** time-splitting gives identical release/index with unchanged cohort; zero shares/pause/empty reserve never create backlog; no positions scanned.

## Task 4: Replace funding and wire governance updates

**Modify:** keeper/rewards.go, keeper/msg_server.go; add keeper/streaming_funding_test.go.

- [ ] Replace the old “fund then accrue instantly” tests with streaming assertions; retain and strengthen transfer-failure/authority/denomination tests. Rename `TestFundingWithNoActiveSharesFailsWithoutTransfer`: it must now prove success into U with I unchanged.
- [ ] In Keeper.FundRewards, validate amount/addition bounds, use `ctx.CacheContext()`, accrue the old reserve, call the native funding adapter, add to U, save, and `write()`. Remove ErrNoActiveShares. Avoid accidental index increase from the deposit itself.
- [ ] Remove only the FundRewards message's pause rejection; preserve Stake's pause rejection. Authority must still be checked before settlement or transfer.
- [ ] UpdateParams must accrue with old params inside a cache before SetParams. Clear emission remainder when new pause=true or new rate=0. Do not update any existing position's duration/multiplier.
- [ ] Keep full atomicity in nested caches: an outer failed operation cannot commit inner settlement or bank events. SDK cached context event managers are used consistently.

Operation shape (apply to each mutating method, preserving its concrete response/event fields):

```go
parent := sdk.UnwrapSDKContext(goCtx)
ctx, write := parent.CacheContext()
// Validate request/authorization and calculate checked bounds.
if err := k.Keeper.AccrueRewards(ctx); err != nil { return nil, err }
// Perform this operation in ctx only; all getters/setters/bank calls use ctx.
// Do not call write until all validation, transfer, and persistence succeeds.
write()
```

- [ ] Test top-up before any stakes, funding while paused, top-up at exhaustion, top-up after a long empty interval, zero-rate prefunding, old-rate/new-rate split, no unauthorized clock writes, and real-adapter rollback.

```bash
go test ./x/ustcstaking/keeper -run 'Test.*(Funding|FundRewards|UpdateParams|Authority|Streaming)' -count=1
```

## Task 5: Settle lifecycle changes and preserve active claim fractions

**Modify:** keeper/msg_server.go, rewards.go, position.go only as necessary; add keeper/streaming_lifecycle_test.go.

- [ ] Stake: accrue before new shares; reset IndexRemainder before cohort change; set debt to current index times new shares; use existing snapshot and principal transfer paths. Check all share additions/products before bank transfer.
- [ ] Claim: compute active whole pending separately from old explicit claimable. Preserve fractional pending by advancing debt by **only active whole pending**. The critical replacement is:

```go
// pendingWhole excludes any explicit ClaimableRewards.
pendingWhole := k.AccruedRewards(position, state)
pay := pendingWhole.Add(position.ClaimableRewards.Amount)
// After checks and successful bank transfer:
position.RewardDebt = position.RewardDebt.Add(pendingWhole.ToLegacyDec())
position.ClaimableRewards = sdk.NewCoin(types.BondDenom, math.ZeroInt())
state.AllocatedRewards = state.AllocatedRewards.Sub(pay)
```

The snippet applies to ACTIVE positions only. For inactive positions AccruedRewards already returns explicit claimable; do not add it twice. Derive `pendingWhole` and `pay` in separate active/inactive branches. Zero payment must not discard fractional debt.

- [ ] Before any reward bank transfer, enforce pay<=A and B>=U+A. Keep SetPosition/DeletePosition updates to derived C; persist updated RewardState in the same cache.
- [ ] Unbond: accrue, freeze integer pending into explicit claimable, zero debt/shares, reset IndexRemainder, reduce S, clear EmissionRemainder if S becomes zero. Preserve A. Set maturity to request time plus the **stored** duration. Document terminal sub-uusd truncation.
- [ ] Withdraw: keep principal-only transfer at `now >= end_time`, cache settlement, and preserve unpaid claims. Keep existing pruning logic.
- [ ] Test the full E2–E5 sequence; default minimum 1 uusd; arbitrary new entrants get no earlier rewards; 1×/2× weighting; old snapshots after tier edits; immediate unbond; maturity one nanosecond early/equal/late; paused/empty-reserve exits and claims.
- [ ] Test E10 with 2 equal shares at 3 uusd/day. Each owner must receive 3 after two days whether claiming every day or only once. Add active explicit claimable to prove no double addition, then unbond/withdraw/claim twice.

```bash
go test ./x/ustcstaking/keeper -run 'Test.*(Stake|Claim|Unbond|Withdraw|Prun|Streaming)' -count=1
```

## Task 6: Protect reserve, dust, and invariants

**Modify:** keeper/invariants.go, msg_server.go:ReturnRewardDust; tests in invariants_test.go and dust_test.go.

- [ ] Replace reward backing check with both L_exact<=A (including fractions) and B>=U+A, while retaining whole-liability diagnostics. Preserve derived C equality, shares equality, and principal>=liability. Validate all new scalar fields and remainder bounds.
- [ ] ReturnRewardDust: reject active shares; settle in cache; verify A>=C and backing; compute `available=B-U-C`; reject excessive requested amount; set A=C; return through the adapter; commit only on success.
- [ ] Do not use `poolBalance - GetReservedRewards()` alone: that old expression can sweep unallocated funding. Do not enable reserve sweeping while paused.
- [ ] Add E9 as an exact test, plus exact-maximum success, one-over failure, donation-only surplus, no active shares with substantial U, allocated rounding dust, adapter failure rollback, foreign coins, and malformed A<C rejection.
- [ ] Test conservation after every lifecycle action: total supply unchanged, principal untouched by reward operations, funded amount equals paid+U+A absent legacy surplus/dust-return adjustments.

```bash
go test ./x/ustcstaking/keeper -run 'Test.*(Invariant|Solvency|Dust|Surplus|Streaming)' -count=1
```

## Task 7: Wire BeginBlock and expose accurate queries

**Modify:** module.go, keeper/query_server.go, client/cli/query.go; module/query/CLI tests.

- [ ] Implement the SDK v0.53.6 BeginBlock interface with this receiver:

```go
func (am AppModule) BeginBlock(ctx context.Context) error {
    return am.keeper.AccrueRewards(sdk.UnwrapSDKContext(ctx))
}
```

Add a compile-time interface assertion using the installed SDK/core appmodule interface. Retain no-op EndBlock. Verify `app/modules.go:orderBeginBlockers` already includes the module; do not add a second entry.
- [ ] RewardState query returns committed state, R, B, paused, and potential runway. Compute U*86400/R via big.Int, with rate-zero handled using `runway_defined=false`. Use bounded checked conversion. Do not present A as exact currently claimable total.
- [ ] Position query computes total integer pending as in claims and returns a Coin alongside the unchanged stored Position. Query methods must not call AccrueRewards or mutate the store.
- [ ] ValidateState pages return page C, exact decimal liabilities, and stored derived C in new fields. CLI aggregates page L/L_exact/C/S at a pinned height and checks scalar ranges, C equality, L_exact<=A, B>=U+A, principal>=liability. Reuse a shared scalar validator to prevent drift between query and invariant logic.
- [ ] Make invalid CLI validation exit nonzero after printing diagnostic output. Test the exit/error path, not just output string contents.
- [ ] Test query results before/after a block, no query writes, frozen-time repeated calls, empty-rate runway, positive-rate potential runway while paused, reserve versus physical balance, multiple validation pages, corrupted reserve fields, donation tolerance.

```bash
go test ./x/ustcstaking/... -run 'Test.*(BeginBlock|Query|CLI|ValidateState|Streaming)' -count=1
```

## Task 8: Genesis, migration, and activation

**Modify/create:** types/genesis.go, genesis.go, keeper/migrations.go, module.go, app upgrade files and corresponding tests.

- [ ] Fresh InitGenesis fills missing timestamp only for the empty default and uses ctx.BlockTime. Current-schema exported state preserves timestamp/U/A/both carries. Rebuild existing derived owner/C indexes. Reject future timestamp and missing fields in nonempty state.
- [ ] Create keeper Migrator/NewMigrator/Migrate1to2. Use a cache; decode genuine legacy records; validate old positions and banking; set new rate zero, U zero, A equal to existing reward-pool uusd, timestamp upgrade time, carries zero. Preserve old positions/index/S/next-ID/tier settings. Rebuild derived indexes and run all applicable new invariants before commit.
- [ ] Bump consensus version to 2 and register migration 1→2:

```go
if err := cfg.RegisterMigration(types.ModuleName, 1, keeper.NewMigrator(am.keeper).Migrate1to2); err != nil {
    panic(fmt.Sprintf("register USTC staking v2 migration: %v", err))
}
```

Registration panic is construction failure, not a migration balance panic. Migration state errors must return errors through RunMigrations.
- [ ] Handle legacy JSON via the explicit legacy-binary import/upgrade/export route in the spec. Do not add permissive runtime fallback that treats arbitrary bank funds as streaming reserve.
- [ ] Update activation handler's existing-version guard for supported source versions; call registered migrations. Protect current pool normalization and permission checks. Make loader behavior conditional on verified absent/existing module store; never add an already existing store again. Inspect `app/app.go` loader registration before editing.
- [ ] Do not assert that candidate name `v14_5` is current mainnet policy. Release identity and predecessor version are external release inputs. Use isolated names/heights in tests; test both absence and existing v1 state without changing production chain selection by guesswork.
- [ ] Tests: genuine old wire bytes; existing earned/fractional rewards survive; A protects all bank funds and U starts zero; no stream until governance sets rate and funds; no-module activation; malicious pre-funded pool addresses and foreign coins; unsupported future version; bad backing rollback; no extra store addition; repeated export/import equality; restart with nonzero carries and later consensus timestamp.

```bash
go test ./x/ustcstaking/... ./app/upgrades/ustc_staking ./app -run 'Test.*(Genesis|Migrat|Upgrade|USTCStaking|Streaming)' -count=1
```

**Checkpoint:** before release, prove the actual target upgrade executes and blocks advance afterward. Do not replace this with a `terrad status` response. Preserve prior remediation CI assertions.

## Task 9: Real-keeper accounting and adversarial tests

**Modify:** app/ustc_staking_integration_test.go and app/keepers/community_pool_adapter_test.go. **Create:** keeper/streaming_property_test.go if a pure reference model is useful.

- [ ] Build E1–E15 using actual bank/distribution keepers where transfers/rollback matter. Seed existing funds through the established test fixture, never introduce minting into the production keeper interface.
- [ ] Replace old immediate-payout assertions with explicit time advances; each change should state why the old expected amount changes. Do not delete ownership, authority, community-pool insufficiency, supply-conservation, or pool-account tests.
- [ ] Build a small independent reference model using rational/big-integer arithmetic. Generate sequences of time advances, stake, claim, unbond, withdraw, top-up, rate changes, pause/resume, and donation. Track every user's principal and paid rewards separately from implementation storage. Compare integer payouts with the declared rounding policy, not with a duplicate call to production helper code.
- [ ] Include same-block stake/unbond, extremely small daily rates, very large shares, adversarial repeated claims, many tiny positions, funding front-running (new stake receives only its future fraction, not the deposit), and bank failure after prospective settlement. Prove earned+paid never exceeds funded/legacy-backed budget.
- [ ] Benchmark AccrueRewards with 1 and 100,000 stored positions while holding the same scalar state; it must not iterate positions or grow linearly with that population. No eligibility queue benchmark is required.

```bash
go test ./app ./app/keepers -run 'Test.*(USTC|CommunityPool|Streaming)' -count=1
go test ./x/ustcstaking/keeper -run 'TestStreamingReference|TestStreamingAdversarial' -count=1
go test ./x/ustcstaking/keeper -run '^$' -bench 'BenchmarkStreamingAccrual' -benchmem
```

## Task 10: Update simulation fixtures and operations

- [ ] In simulation/genesis.go, set an explicit nonzero **simulation-only** R, initialize new state fields, and preserve seeding through bank/community-pool genesis. Production DefaultParams remains zero-rate.
- [ ] In simulation/proposals.go, ensure generated MsgUpdateParams contains R and validates; generate top-ups independently of active-share count. Include rate changes and paused top-ups.
- [ ] In simulation/operations.go, ensure actions advance through real block timestamps and include claims both before and after releases. Keep BeginUnbonding/Withdraw maturity rules. Do not retain a special no-active-funding rejection.
- [ ] Update simulation/decoder.go and deterministic fixtures for new state fields. Export/import comparisons include timestamp and both carries; never add these economic fields to the list of ignored state differences.
- [ ] Update app/sim_test.go only for real new semantics. Keep the restored flag registration and actual simulation test names; do not return to app_test package/flag errors or nil WeightedOperations.

```bash
go test ./x/ustcstaking/simulation -count=1
make test-sim-nondeterminism
make test-sim-import-export
make test-sim-after-import
make test-sim-multi-seed-short
```

Expected: all four jobs execute their named tests and pass; collect per-seed stdout on failure. Passing with no matching test is not evidence.

## Task 11: Documentation, API artifacts, and operational examples

- [ ] Update `x/ustcstaking/spec/{01_concepts,02_state,03_end_block,04_messages,06_params}.md` (create a clearly named block-processing section if 03_end_block exists under another name). Explain BeginBlock, units, reserve/allocated distinction, fractional claims, terminal rounding, and explicit withdrawal.
- [ ] Update `docs/ustc-staking/executive-summary.md`, both audience briefings, phase-2 localnet testing and release-readiness docs. Mark the 2026-10-08 economic tasks superseded; retain independent hardening requirements. Do not implement historical Phase 3.
- [ ] Include this **illustrative** complete Params object in a governance MsgUpdateParams proposal; replace authority using the chain's actual gov module address, never a wallet:

```json
{
  "bond_denom": "uusd",
  "lock_tiers": [{"id": 1, "duration": "1814400s", "multiplier": "1.000000000000000000"}],
  "authority": "<chain gov module address>",
  "paused": false,
  "reward_rate_per_day": "1000000000"
}
```

Explain that the rate means 1,000 USTC/day for the whole program and is an example, not an approved budget. Operators query current full params and preserve any other configured tiers when changing rate. Existing MsgFundRewards then transfers only authority/amount; it does not change this rate.
- [ ] Localnet sequence: fund before stake; configure a short **test-only** tier and test rate through governance; stake two users at different block times; observe reserve decrease and pending increase; claim; change rate; pause/resume; exhaust/top-up; unbond/withdraw; export/restart; compare supply/community-pool/principal/reward balances. Use bounded polling of chain time/height, not assumptions about wall-clock seconds.
- [ ] Regenerate API artifacts with `make proto-gen` and `make apply-swagger`. Confirm reward rate/reserve fields appear in generated swagger and embedded statik. Never hand-edit generated output to hide a broken generator.
- [ ] Update CHANGELOG with the funding semantic change, immediate eligibility, default tier, zero default rate, pause/top-up behavior, and version-2 migration. No success claims until Task 12.

## Task 12: Final evidence and reviewer handoff

- [ ] Run focused tests after each task; now run the complete affected test suites and vet. Record commands, exit codes, seed IDs, and any environment blockers in a Markdown evidence file. No fabricated pass claims.

```bash
go test ./x/ustcstaking/... ./app/keepers ./app/upgrades/ustc_staking -count=1
go test ./app -run 'Test.*(USTC|Streaming)' -count=1
go vet ./x/ustcstaking/... ./app/keepers ./app/upgrades/ustc_staking
git diff --check
```

- [ ] Confirm all four simulation jobs from Task 10 and activation tests from Task 8 have current passing evidence; rerun when later changes invalidate it.
- [ ] Run the authorized seven-validator localnet campaign using `ustc-localnet-campaign` skill when its environment is available. Adapt expectations to streaming; do not carry forward old allocation/age assertions. If infrastructure is unavailable, record the exact unrun release gate rather than calling the feature release-ready.
- [ ] Review protected boundaries with `git diff` scoped to the new implementation checkpoint. Existing remediation changes in unrelated files are not evidence that streaming changed LUNC. Confirm USTC keeper interfaces still contain no mint/staking dependency and both module accounts retain no mint/burn permissions.
- [ ] Review generated artifacts, full-parameter proposal behavior, invariant/CLI agreement, and callback order. Confirm no source path assigns a new funding deposit directly to RewardIndex.
- [ ] Prepare a concise handoff: files changed, behavior, proof of supply/custody/rounding, defaults and governance controls, migration/upgrade prerequisites, limitations (manual withdrawal, terminal sub-uusd truncation), and remaining external release inputs.

## Reviewer rejection checklist

Reject implementation if any item is true:

- Funding cannot happen before staking, or a top-up immediately awards the whole deposit.
- New stakers receive elapsed-time rewards from before joining.
- Claims can spend U or dust return can sweep U/earned claims.
- Old A/C/bank balances are treated as future funding without explicit governance transfer.
- User claims reset debt to the complete current entitlement and discard active fractions.
- Rate/pause/top-up changes are applied retrospectively.
- Block processing scans positions, or an eligibility queue/age rule was reintroduced.
- Partial failed messages commit clock, reserve, bank, or event changes.
- Queries write state or estimate with local wall-clock time.
- Migration loses earned rewards, adds an existing store again, or silently rewrites snapshots.
- Tests pass only because streaming fields or reward differences are ignored during export/import.
- The author claims deployment/activation based only on an RPC status response.

## Implementer kickoff prompt

```text
Implement the approved USTC funded-streaming design in this repository.
Read docs/superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md
and docs/superpowers/plans/2026-10-13-ustc-staking-streaming-rewards.md fully.
Read AGENTS.md and its baseline docs; there is no Phase 3.
Preserve the existing dirty review-remediation patch; clean HEAD alone is not the baseline.
Use executing-plans, work through the numbered tasks, and report evidence at checkpoints.
Do not implement the superseded seven-day eligibility queue or one-USTC stake floor.
Do not modify LUNC staking, mint USTC, deploy, or publish without separate authorization.
Defaults: one 21-day/1x tier and zero release rate; governance chooses the production rate.
Protect unallocated reserve and allocated claims separately; follow the exact rounding rules.
If source differs materially from this handoff, report the concrete mismatch before changing policy.
```
