# Governance, Communication, and RFC Process (Formal)

## Governance Decisions
This chapter defines how decisions across the RFC lifecycle are prepared, made, documented, and communicated. The goal is a traceable process combining technical quality, stakeholder participation, and formal decisionability.

In this RFC, governance does not mean operational delivery steering. It means content-level direction decisions, release logic, and transparent risk acceptance across phases Omega through C.

This chapter distinguishes two governance scopes to avoid ambiguity:
- **RFC Governance Authority** - the process-level authority for active RFC operations (editing flow, versioning flow, comment processing flow, and preparation of closure/freeze decisions).
- **Terra Classic Chain Governance** - the on-chain governance body that can observe the RFC process and decide formal proposals affecting RFC direction, closure, or freeze.

During the open RFC process, document editing and versioning are intentionally not operated as day-to-day chain-governance actions. This preserves operational agility while comments and decisions are still being incorporated. Chain governance remains fully entitled to question, challenge, or revise RFC decisions through formal proposals, but this is defined as an escalation path, not the default operating mode.

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
The process distinguishes functional roles:
- **RFC Editors** - consolidate text contributions, maintain chapter consistency, and maintain versions and change logs.
- **Technical Maintainers** - assess technical coherence, risk assumptions, and gate readiness across chapters and phases.
- **RFC Governance Authority** - makes process-level release, closure, and freeze preparations and coordinates formal decision packages for governance steps.
- **Terra Classic Chain Governance** - retains final on-chain decision rights where formal proposals are required and can challenge or revise RFC direction when needed.
- **Auditors** - perform internal, external, or combined gate assessments as defined in Chapter 06.

Roles may overlap in persons, but documentation must still clearly separate who proposed, reviewed, and finally decided.

## RFC Feedback Cycle
The RFC feedback process is continuous during the open RFC phase, not round-based. Incoming comments are reviewed on an ongoing basis, incorporated where accepted, and reflected in updated document versions and decision records.

In the default flow, each relevant comment can trigger a new document revision. Where appropriate, multiple comments may be consolidated into one combined revision, provided traceability per comment is preserved in the Comment Log and corresponding decision updates remain explicit in the Decision Log.

This model is intentionally lightweight and operationally flexible for a decentralized environment: no fixed feedback windows and no mandatory institutional review rounds are required as the default operating pattern.

## Comment Submission and Review Process
Comments are submitted through an RFC mailing list (`to-be-done`, list details will be published before the next feedback round). The process is:

1. A commenter submits a comment to the mailing list with a clear subject, chapter reference, and rationale.
2. RFC editors review the comment for scope fit, clarity, and actionability.
3. Editors assign one of two outcomes: `accepted` or `rejected`.
4. Every reviewed comment is recorded in Chapter 10 (Comment Log) with status and short rationale.
5. Accepted comments usually result in a document change; the corresponding chapter/revision reference is recorded in Chapter 10.

This process ensures that comments are not only discussed but traceably processed and linked to resulting document updates.

## Freeze
RFC freeze is defined as preceding `Phase Omega` and must be completed before implementation phases A-C begin.

Freeze is triggered by a formal governance proposal on freeze decision. The RFC is freeze-ready only when critical open decisions are resolved or reasonedly deferred, no blocking inconsistencies remain between Informative and Formal chapters, prioritization and phase logic are stable, audit-gate logic per phase is complete, and stakeholder feedback plus scope and decision-log updates are cleanly integrated. Freeze output is a published "frozen" reference version; changes to frozen core decisions are only allowed via new RFC revision.

## Change Control After Freeze
After freeze, a two-level change model applies. Editorial changes without impact on target state, gate logic, or scope may be applied directly. Content changes affecting target states, phase logic, gate criteria, risk assumptions, or migration principles require new RFC revision with formal governance approval. Each content change must explicitly reference impacted chapters at least through target state, roadmap phase, audit gate, and decision-log entry.

## Decision and Escalation Logic During Implementation
Even after freeze, governance decisions remain required, especially for critical audit findings needing risk acceptance, gate blockers that trigger replanning, external PQ developments affecting assumptions, and deviations from defined cutover/security principles. Escalation logic is strict: first technical clarification and documentation, then explicit governance decision with Go/No-Go or deferral. Silent continuation with unresolved critical deviations is excluded.

## Communication Framework
Communication follows the principle "one process, two categories": Informative communication focuses on understandable purpose, risk, and impact for community/users; Formal communication focuses on precise decision basis, gate status, and residual risk. For every phase-relevant decision, communication must include at least decision and scope, technical rationale, affected phases/gates, and open residual risks with next review point.

## Link to Logs
This chapter defines process; Chapter 09 is the operational decision register and Chapter 10 is the operational comment register. Every relevant governance step must create a traceable status change in the Decision Log (`open`, `closed`, `deferred`) including short rationale. Every submitted comment must be tracked in the Comment Log (`accepted`, `rejected`) including short rationale and, when applicable, document-change references.
