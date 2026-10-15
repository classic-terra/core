# USTC staking: technical advisor briefing

Updated October 15, 2026. Reviewed implementation: `dfb176f8`; packaged production binary source: `34d0703`. The intervening commit changed tests and documentation only.

## Economic behavior

Governance authorizes `MsgFundRewards`, which moves existing uusd from distribution's community pool into unallocated reward reserve U. Funding itself does not allocate the deposit. BeginBlock accrues a global `reward_rate_per_day` using elapsed consensus time and weighted active shares. Accrual is constant work in position count and uses integer arithmetic with remainder carries.

Released funds move from U into allocated/protected rewards A. Bank balance B must cover U+A, and A must cover exact earned liabilities. Physical reward-pool balance is not interchangeable with remaining release budget. Principal custody uses a distinct account; the USTC keeper exposes no mint or LUNC-staking dependency.

Messages settle the old interval before changing shares, funding, rate, or pause state. Cached operations commit settlement, transfers, and events together on success. New entrants receive no historical earnings. Active claims preserve fractional entitlement; unbonding freezes whole rewards and discards a terminal fraction below one uusd into protected rounding dust.

## User experience and governance

Fresh defaults are one 21-day unbonding tier, 1× weight, and zero emission. Users can start unbonding immediately; no separate pre-unbond lock applies. Earning stops at unbonding, principal withdrawal is explicit at maturity, and earned rewards remain claimable afterward. Tier edits affect future positions; existing snapshots persist.

Governance may prefund without stakers, fund while paused, change the rate, or top up after exhaustion. Pause blocks new stakes and emission while allowing funding, claims, and exits. Idle intervals do not create backpay. A chain-time gap with active shares can accrue on restart, capped by reserve. Dust return cannot recover U or earned claims.

The proposed 30M USTC budget corresponds to `41095890410 uusd/day` over 730 active days; see the [analysis](../30m-ustc-two-year-rewards-analysis.md) for 731-day calendar treatment and participation scenarios. This is a configurable emission budget, not a fixed APR or scheduled on-chain end date.

## Migration, verification, and operations

The v2 migration protects legacy bank funds as allocated rewards, initializes U=0 and the new rate to zero, and preserves earned entitlements. Separate upgrade registrations handle absent and existing USTC stores. Operators must verify the target chain before selecting either path.

The [final review](../final-streaming-review-2026-10-15.md) records source checks, fresh affected test results, 35 import/export PASS logs, 33 full after-import PASS logs, two separate terminal cases, and completed localnet lifecycle/restart snapshots. After-import simulation normalizes zero-height LUNC unbonding entries for the SDK simulator; ordinary import/export is separately tested. This is substantial testnet evidence, not exhaustive security proof.

Swagger/statik generation is deferred for testnet. Disable Swagger UI; the generated Go/gateway APIs remain available and CLI/REST/gRPC were smoke-tested. Public governance, validator coordination, plan/height selection, and post-activation verification remain operational work. No public deployment is claimed.

The approved scope has no Phase 3. No validator incentive program, TreasuryManager, POL, Wasm funding contract, minting, or LUNC consensus changes are included.

## References

- [Streaming design](../../superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md)
- [Implementation plan](../../superpowers/plans/2026-10-13-ustc-staking-streaming-rewards.md)
- [Validation evidence](../phase-2-streaming-rewards-validation.md)
- [Operator handoff](../streaming-testnet-deployment.md)
- [C-suite briefing](c-suite-briefing.md)
