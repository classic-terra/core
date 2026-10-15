# USTC Staking: Executive Summary

**Status as of October 15, 2026**

## At a glance

USTC staking is implemented as a native Terra Classic testnet candidate: users stake
USTC directly from their own accounts, while rewards are paid only from USTC
that has already been funded. It does not change LUNC validator power or
standard validator staking.

The approved design streams governance-funded rewards over time. Governance
may fund before users stake and top up later; a configured daily rate controls
how quickly the existing reserve is shared. The default rate is zero, so
governance must set it before releases begin. No Phase 3 is planned.

## What has been done

### Phase 1 — Native USTC staking

The native module supports direct USTC staking into configurable lock tiers,
reward claims, unbonding, and withdrawal after maturity. Principal and reward
balances are kept in separate module accounts. Governance controls module
parameters and pause/resume.

The initial local campaign, completed September 3, 2026, recorded a functional
pass on a seven-validator network. It exercised staking, reward funding and
share-weighted claims, governance changes, expected rejection cases,
unbonding and withdrawals, supply conservation, validator-state isolation, and
single-node and full-network restarts. Subsequent Phase 1 hardening expanded
state validation, owner queries, events, and application-level accounting
coverage. The completed streaming campaign exercised the current candidate through a
seven-validator upgrade, governance funding, lifecycle checks, and restarts.

### Phase 2 — Governance community-pool funding

The funding approach is now native and governance-controlled. Users continue
to stake directly with `MsgStake`; governance proposals execute
`MsgFundRewards` to transfer an approved amount of existing USTC from the
distribution community pool to the USTC reward pool. Reward accounting updates
only after that transfer succeeds. No reward is minted, and user principal is
not used to fund rewards.

Streaming rewards replace immediate allocation at funding. `MsgFundRewards`
transfers existing USTC from the distribution community pool into an
unallocated reserve; BeginBlock releases that reserve at the governance-set
rate using consensus time. Stakers remain immediately eligible, can claim at
any time, stop earning when unbonding begins, and withdraw principal after
their selected tier's unbonding wait. There is no separate lock before
requesting unbonding. Principal and rewards remain in separate pools;
nothing is minted.

## What is happening now

Module, app, adapter, upgrade, simulation, and seven-validator rehearsal
checks have passing evidence. The final source review found no new blocking
USTC logic defect in the paths reviewed. Two terminal simulation cases and
normalization of imported LUNC test fixtures qualify the coverage; Swagger
remains an accepted testnet documentation deferral.

The candidate is ready for an operator-led, governed testnet rehearsal. Public
activation still requires the agreed plan/height, validator coordination, and
governance approval. No public deployment or mainnet certification is claimed.

The intended reward budget is 30 million USTC for approximately two years.
For 730 active days, the global rate is about 41,095.89 USTC/day. The rate is
configurable and does not guarantee an individual interest rate. See the
[budget analysis](30m-ustc-two-year-rewards-analysis.md) for exact values,
calendar-year treatment, and participation assumptions.

## Scope boundary

The approved scope ends with native USTC staking and streamed
community-pool-funded staker rewards. There will be no Phase 3 validator
incentive overlay. This module does not change LUNC validator power,
self-bonding, commission, slashing, jailing, or distribution rewards.

## Executive takeaway

Stakers can join immediately, earn a share of rewards released over time, claim
whenever they choose, and exit under their lock tier. Governance controls the
funded reserve, daily release rate, and pause state. No Phase 3 work is planned.

## Related documents

- [Final implementation review](final-streaming-review-2026-10-15.md)
- [Validation evidence](phase-2-streaming-rewards-validation.md)
- [Streaming localnet campaign](phase-2-streaming-rewards-localnet-testing.md)
- [Testnet operator handoff](streaming-testnet-deployment.md)
- [30M USTC analysis](30m-ustc-two-year-rewards-analysis.md)
- [C-suite decision briefing](briefings/c-suite-briefing.md)
- [Streaming rewards design](../superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md)
- [Streaming rewards implementation plan](../superpowers/plans/2026-10-13-ustc-staking-streaming-rewards.md)
