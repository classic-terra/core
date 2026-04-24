# 8) Governance, Communication, and RFC Process

## Governance Decisions
This chapter defines how decisions across the RFC lifecycle are prepared, made, documented, and communicated. The goal is a traceable process combining technical quality, stakeholder participation, and formal decisionability.

In this RFC, governance does not mean operational delivery steering. It means content-level direction decisions, release logic, and transparent risk acceptance across phases Omega through C.

Binding principle: traceability is prioritized over speed, risk acceptance must always be explicit, no phase-critical continuation is allowed without satisfied gate logic, and post-freeze technical deviations are only permitted via formal change rules.

## Stakeholder Groups
The following groups are central for feedback, impact, and practical feasibility:

- `Users`: end users whose asset protection, usability, and migration capability are central to wallet/client strategy.
- `Validators`: operators of consensus infrastructure, critical for key binding, cutover stability, and safe post-switch startup.
- `Wallet providers`: teams/products implementing signing, key management, and recovery flows for hybrid and later PQ-native operation.
- `Exchanges`: central trading/settlement actors with high security requirements and early integration relevance for PQ-capable signature paths.
- `Custody providers`: professional custodians with elevated responsibility for key control, operational security, and auditable processes.
- `Integrators`: infrastructure and product teams that must keep nodes, APIs, SDK-adjacent services, and operations processes compatible with new paths.
- `External infrastructure and standardization actors`: references from broader ecosystem (for example Cosmos/Ethereum-adjacent projects, security research, tooling providers) relevant for Phase C and ongoing PQ alignment.

## Roles in the RFC Process
The process distinguishes functional roles: RFC editors consolidate text and maintain versions/change log; technical maintainers assess coherence, risks, and gate readiness; the governance authority makes formal release and freeze decisions; auditors provide internal or external gate assessments per Chapter 06. Roles may overlap in persons, but documentation must still clearly separate who proposed, reviewed, and finally decided.

## RFC Feedback Cycle
The feedback process runs iteratively until freeze readiness and follows a stable four-step cycle: publish an RFC version and open a feedback window, classify comments, publish follow-up revision with change log, then run a short review round for changed sections. Each round ends with a transparent delta showing what changed, what remains intentionally open, and what was transferred into the Decision Log.

## Freeze
RFC freeze is defined as preceding `Phase Omega` and must be completed before implementation phases A-C begin.

Freeze is triggered by a formal governance proposal on freeze decision. The RFC is freeze-ready only when critical open decisions are resolved or reasonedly deferred, no blocking inconsistencies remain between Public and Technical Layer, prioritization and phase logic are stable, audit-gate logic per phase is complete, and stakeholder feedback plus scope and decision-log updates are cleanly integrated. Freeze output is a published "frozen" reference version; changes to frozen core decisions are only allowed via new RFC revision.

## Change Control After Freeze
After freeze, a two-level change model applies. Editorial changes without impact on target state, gate logic, or scope may be applied directly. Content changes affecting target states, phase logic, gate criteria, risk assumptions, or migration principles require new RFC revision with formal governance approval. Each content change must explicitly reference impacted chapters at least through target state, roadmap phase, audit gate, and decision-log entry.

## Decision and Escalation Logic During Implementation
Even after freeze, governance decisions remain required, especially for critical audit findings needing risk acceptance, gate blockers that trigger replanning, external PQ developments affecting assumptions, and deviations from defined cutover/security principles. Escalation logic is strict: first technical clarification and documentation, then explicit governance decision with Go/No-Go or deferral. Silent continuation with unresolved critical deviations is excluded.

## Communication Framework
Communication follows the principle "one process, two layers": Public Layer focuses on understandable purpose, risk, and impact for community/users; Technical Layer focuses on precise decision basis, gate status, and residual risk. For every phase-relevant decision, communication must include at least decision and scope, technical rationale, affected phases/gates, and open residual risks with next review point.

## Link to Decision Log
This chapter defines process; Chapter 09 is the operational register. Every relevant governance step must create a traceable status change in the Decision Log (`open`, `closed`, `deferred`) including short rationale.
