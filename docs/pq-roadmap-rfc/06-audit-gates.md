# Audit Gates as a Mandatory Component (Formal) {#ch-audit-gates}

## Purpose and Core Principle
Audit gates are the binding security and quality mechanism between roadmap phases. They prevent insufficiently validated assumptions from prototype or migration work from moving into live operation.

Core principle: without a passed gate, no recommendation is issued to proceed into the next risk-relevant step. Exceptions are only allowed as explicit governance risk acceptance and must be documented with rationale, residual risk, and remediation plan.

## Audit Logic Across Phases
The RFC uses different gate types by phase:
- `Phase Omega`: consistency and governance gate (content/process, no code security audit).
- `Phase A`: two mandatory technical security gates (prototype and migration components) for consensus path.
- `Phase B`: one mandatory technical security gate after prototype operation for hybrid wallet/tx path.
- `Phase C`: no single rigid one-shot gate, but an ongoing assurance model for product, operations, and supply-chain risks.

This keeps gate strictness aligned with phase criticality: strictest in consensus, strict in wallet/tx, continuous operational assurance in client ecosystem.

## Gate Matrix (Binding Minimum Logic)
1. `Omega Gate`: Phase Omega must be complete before A-C start.
2. `A Gate 1` (after A2): audit of PQ consensus prototype as prerequisite for A4.
3. `A Gate 2` (after A5): audit of consensus migration components as prerequisite for production migration decision.
4. `B Gate` (after B2): audit of hybrid wallet/tx path as prerequisite for B4.
5. `B4` intentionally has no second separate mandatory gate; with high complexity, governance may require additional re-audit.
6. `C Assurance`: recurring review cycles instead of single release gate.

## Mandatory Components of Every Audit Package
Each gate must include at least:
- Audit objectives and explicit scope (including out-of-scope).
- Audit type (internal, external, or combined) with role split.
- Auditor input artifacts (code, architecture, test evidence, threat model, operations documentation).
- Findings by criticality class.
- Re-audit rules per criticality class.
- Formal Go/No-Go recommendation with residual-risk assessment.

## Finding Classes and Blocker Rules
Binding minimum classes:
- `Critical`: direct blocker; no transition without fix and re-audit.
- `High`: blocker for production-near or production steps; exception only via explicit governance risk acceptance.
- `Medium`: no automatic blocker, but requires deadline and tracking.
- `Low/Info`: documentation and backlog required, no gate blocker.

A gate counts as passed only when all `Critical` findings are closed and no uncontrolled `High` risk exposure remains.

## Re-Audit and Change Rules
Re-audit is mandatory for:
- Fixes for `Critical` or `High` findings.
- Changes in cryptographic core paths (signing or verification logic).
- Changes in cutover, migration, or key-binding mechanisms.
- Changes affecting determinism, safety/liveness, or key management.

After a passed gate, relevant code and configuration changes must be documented against audit scope up to the next milestone, to prevent scope drift.

## Mandatory Review Points in Consensus Path (Phase A)
- Safety/liveness properties of forked CometBFT under PQ integration.
- Correct verification in proposal, vote, and commit paths.
- Determinism across upgrade and cutover windows.
- DoS risks from larger signatures, changed verification costs, and load spikes.
- Hardening of PrivValidator/remote-signer interfaces.
- Correctness and tamper resistance of key binding (`classical -> PQ`) and snapshot/activation logic.
- IBC continuity risks from consensus-signature transition, including dependency on counterparty client upgrades and non-participation scenarios.
- Relayer-readiness and continuity controls for migration, including concentration risk treatment for the currently known single paid Terra Classic relayer operator (`LuncGoblins`).
- Counterparty test-campaign evidence, including rehearsal outcomes for partial-adoption fallback modes.
- Negative tests (malformed input, boundary sizes, mixed-version scenarios).

## Mandatory Review Points in Wallet/Tx Path (Phase B)
- Correct signature verification and ante-handler behavior in hybrid signature paths.
- Replay, malleability, and domain-separation checks.
- Gas/fee hardening and anti-spam/DoS behavior in hybrid mode.
- Correct codec, serialization, and relevant derivation rules.
- Key-management and recovery flows for users and integrators.
- Compatibility tests across introduction and migration windows.

## Mandatory Review Points in Client/Operations Context (Phase C)
Phase C focuses long-term operation of a Terra-Classic-native wallet/explorer stack and the carrier structure. It requires continuous assurance on:
- Operations and hosting security (web, backend, node operations, monitoring).
- Supply-chain security and release integrity (build, distribution, signing, publisher accounts).
- Governance and ownership control of critical assets (DNS, repositories, package/app distribution).
- Incident, patch, and vulnerability response processes with traceable escalation paths.
- External PQ standards monitoring and technical decision feedback.

## Documentation Duty per Gate
Each gate must produce an auditable decision dataset:
- Scope and version of reviewed artifacts.
- Findings list with status and ownership.
- Risk acceptances (if any) including deciding authority.
- Go/No-Go decision and approved next step.

This ensures the roadmap is not only implemented technically, but also governed reproducibly and traceably across its risk-critical transitions.
