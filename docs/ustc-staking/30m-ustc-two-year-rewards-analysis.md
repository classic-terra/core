# 30 million USTC: two-year streaming rewards budget

Prepared October 15, 2026 against the reviewed streaming implementation at `dfb176f8`. The user-specified initial budget is **30,000,000 USTC**. This document calculates configuration values; it does not enact governance, transfer funds, or change chain parameters. No launch date has been selected here.

## Assumptions and units

One USTC equals 1,000,000 uusd. Funding moves existing USTC from the distribution community pool into the unallocated reserve. Principal is held separately. The rate is an integer number of uusd per 86,400 seconds for the entire program, divided among active weighted shares.

The primary scenario uses **730 days**, continuous positive active shares, no pause, no rate change, and no additional funding. Two calendar years may instead span 731 days; choose the duration from the intended dates before submitting parameters.

## 730-day configuration

| Measure | Value |
|---|---:|
| Initial funding | 30,000,000 USTC |
| Funding amount in uusd | `30000000000000` |
| Integer `reward_rate_per_day` | `41095890410` |
| Program release per day | 41,095.890410 USTC |
| Hourly equivalent, approximate | 1,712.328767 USTC |
| Per-minute equivalent, approximate | 28.538813 USTC |
| Per-second equivalent, approximate | 0.475647 USTC |
| Seven-day release | 287,671.232870 USTC |
| 30-day release | 1,232,876.712300 USTC |
| 365-day release | 14,999,999.999650 USTC |
| Initial query runway | 63,072,000 seconds |
| Unallocated remainder after exactly 730 active days | 0.000700 USTC |

Calculation: `floor(30000000000000 / 730) = 41095890410 uusd/day`. Rounding down avoids spending the budget before the target duration. The tiny remaining 700 uusd is released at a subsequent block with sufficient elapsed time (approximately 1.47 milliseconds beyond 730 days under continuous accrual). Block times determine when that settlement is observable.

For two **calendar years containing 731 days**, use `41039671682 uusd/day` = **41,039.671682 USTC/day**. Initial query runway is 63,158,400 seconds; 458 uusd (0.000458 USTC) remains after exactly 731 active days. October 2026 to October 2028 is one example containing a leap day; it is not an assumed launch schedule.

## Governance inputs

The following are budget inputs, **not a complete transaction or proposal**:

```json
{
  "funding": {"denom": "uusd", "amount": "30000000000000"},
  "reward_rate_per_day": "41095890410",
  "duration_basis_days": 730
}
```

`duration_basis_days` is explanatory metadata, not an on-chain parameter. `MsgFundRewards` supplies the positive coin and the actual governance-module authority. `MsgUpdateParams` must contain the complete current Params with the chosen rate, preserving intended tiers and other fields. The fresh rate is zero. Governance may prefund at zero rate, then enable releases after operators verify balances and configuration. These actions can be proposed together if desired; the implementation does not require recurring funding proposals.

Before any releases, a fresh 30M funding leaves unallocated rewards U=30M, allocated/protected rewards A=0, and reward-pool bank balance B=30M USTC, assuming no pre-existing balances. Accrual moves U to A without moving bank coins; claiming reduces A and B. Display U as the remaining release budget and A as allocated/protected rewards. The physical bank balance alone is not the remaining program budget.

## Individual reward scenarios

For an interval with unchanged shares:

`individual reward ≈ program release × individual shares / total active shares`

Shares reflect principal and the position's snapshotted tier multiplier, subject to integer rounding. With all positions at 1×, steady total active principal, no compounding, and uninterrupted emission, the 730-day schedule yields these approximate annual token returns:

| Total active principal | Annualized token return |
|---|---:|
| 30 million USTC | 50% |
| 100 million USTC | 15% |
| 300 million USTC | 5% |
| 1 billion USTC | 1.5% |

Example: 100,000 USTC out of 100 million active USTC at 1× receives approximately 41.095890 USTC/day, 15,000 USTC over 365 days, or 30,000 USTC over the full run, before transaction fees and assuming the share fraction stays constant. These are arithmetic scenarios, not guaranteed interest rates, USD returns, or APY. The system does not automatically compound rewards.

A higher multiplier increases a position's fraction of the fixed release budget; it does not increase total emission. Participation changes individual rewards, not global spending. With only one active staker, that position can receive nearly the entire daily allocation. A fixed rate therefore spends the budget even during low participation.

## Timing, exhaustion, and controls

- There is no hard-coded two-year end date. The budget and rate imply potential active runtime.
- No active shares, a governance pause, or a zero rate stops release and extends calendar runway. Those idle intervals do not create backpay for future stakers.
- A chain halt is different: elapsed consensus time may accrue on the next block against existing shares, capped by remaining reserve.
- Unclaimed earnings are already allocated. Delaying claims does not extend runway; physical coins can remain in the reward pool after the unallocated reserve is exhausted.
- Governance can change the rate, pause, or top up without another software upgrade. A top-up at the same rate extends runway; it does not instantly pay the new deposit. A later top-up can restart emission after exhaustion without backpay.
- Unallocated reserve cannot be swept using the dust-return API, including while paused. There is no separate reserve-cancellation API in this design.
- Fresh defaults are a 21-day unbonding tier at 1×, immediate eligibility, and no extra minimum beyond a positive amount producing positive shares. Users can start unbonding immediately; earnings then stop. The two-year budget is not a two-year user lock.
- Exhaustion stops new earnings. Earned rewards remain claimable; principal still follows explicit unbonding and withdrawal. No minting fills a shortfall.
- Active claims preserve sub-uusd fractions. Ending an active position truncates its remaining fraction below one uusd, which stays protected as rounding dust until reconciliation.

Monitor U, A, B, rate, pause state, total shares, pending claims, and potential runway. Require `B >= U + A`; principal custody is checked separately. Governance can choose a different rate as participation changes, but doing so changes the two-year budget projection.

## References

- [Approved streaming design](../superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md)
- [Final implementation review](final-streaming-review-2026-10-15.md)
- [Operator handoff](streaming-testnet-deployment.md)
- [Rate calculation source](../../x/ustcstaking/keeper/streaming.go)
- [Defaults](../../x/ustcstaking/types/params.go)
- [Runway query](../../x/ustcstaking/keeper/query_server.go)
