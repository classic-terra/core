# USTC Staking: Executive Summary

**Status as of October 13, 2026**

## At a glance

USTC staking is being developed as a native Terra Classic feature: users stake
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
coverage. The Phase 2 validator campaign will retest the user lifecycle on the
current candidate while exercising the community-pool funding path.

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
their selected tier's lock. Principal and rewards remain in separate pools;
nothing is minted.

## What is happening now

The current implementation work must pass module, app, upgrade-migration,
simulation, and seven-validator localnet checks. The runbook evidence should
identify the exact candidate revision, governance proposals, reward rate,
block-time observations, resulting balances and events, restart outcomes, and
any unexpected behavior.

After validation, the team can prepare the operator-led testnet upgrade
rehearsal and governance operating procedures. Local functional testing is
evidence for that rehearsal, not activation approval by itself.

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

- [Phase 2 community-pool validator campaign](phase-2-community-pool-localnet-testing.md)
- [Phase 2 community-pool release readiness](phase-2-community-pool-release-readiness.md)
- [C-suite decision briefing](briefings/c-suite-briefing.md)
- [Streaming rewards design](../superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md)
- [Streaming rewards implementation plan](../superpowers/plans/2026-10-13-ustc-staking-streaming-rewards.md)
