# USTC staking: C-suite briefing

Updated October 15, 2026. The implementation is ready for a governed testnet rehearsal. It has not been deployed to public testnet by this work; mainnet approval is a separate decision.

## What users receive

Users stake USTC directly into a native chain module. Governance supplies existing USTC from the community pool, and a daily program budget releases rewards gradually among active stakers. Nothing is minted. Staked principal and reward funding are held separately.

Users start earning immediately when funding and a positive release rate are available. They may claim earned rewards at any time. They may begin unbonding immediately, stop earning at that point, and explicitly withdraw principal after their tier's wait. The fresh default is 21 days at a 1× reward weight. There is no additional lock before starting that wait, and no automatic compounding or principal return.

## Funding and participation

The intended starting budget is 30 million USTC for approximately two years. A 730-day schedule releases about 41,095.89 USTC/day for all stakers combined. Exact calendar dates may require 731 days. See the [budget analysis](../30m-ustc-two-year-rewards-analysis.md) for exact chain units and assumptions.

The rate does not promise a fixed individual return. More active weighted stake reduces each existing position's share. With very few participants, the same daily budget is concentrated among them. Governance can change the release rate, pause, or top up the reserve without another software upgrade. Zero participation or pauses extend calendar runway; unclaimed earned rewards do not.

## Evidence and launch decisions

Module/app/upgrade tests, simulation checks, and a seven-validator disposable upgrade/lifecycle/restart rehearsal have passing evidence. The [final review](../final-streaming-review-2026-10-15.md) explains scope and qualifications. Swagger documentation remains deferred for testnet; runtime API checks are separate and passed during the campaign.

Operators and governance must still select the public upgrade plan/height, coordinate validators, approve the funding and rate, and monitor balances after activation. Funding alone does not start rewards: the default emission rate is zero. Engineering evidence supports a testnet trial and does not establish guaranteed returns or mainnet certification.

## Scope

The approved scope ends with native USTC staking and governance-funded streaming rewards. There is no Phase 3 or validator incentive overlay. LUNC validator power, commission, slashing, jailing, and rewards remain separate. No treasury contract, POL, DEX revenue, minting, or automated revenue generation is included.

## References

- [Technical advisor briefing](technical-advisor-briefing.md)
- [Executive summary](../executive-summary.md)
- [Final review](../final-streaming-review-2026-10-15.md)
- [Operator handoff](../streaming-testnet-deployment.md)
