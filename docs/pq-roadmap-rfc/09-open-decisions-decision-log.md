# Decision Log (Formal)

This chapter is the operational decision register of the RFC. It tracks closed, open, and deferred points in a form directly usable for governance, audit, and phase steering. The Decision Log is updated with each relevant RFC revision and is the binding reference for status transitions.

### Q-01
**Question:** Which cryptographic profile is selected as the target profile for the consensus path?  
**Decision:** ML-DSA is selected as the target profile for the consensus path.  
**Rationale:** The consensus path is the highest-criticality path in this RFC. A single explicit target profile reduces ambiguity in architecture, testing, migration planning, and audit scope. This avoids fragmented implementation assumptions across phases and creates a clear baseline for gate-based validation.  
**Status:** closed  
**Reference:** Chapters 04, 05

### Q-02
**Question:** Should the consensus path run in permanent hybrid mode or move to a clear PQ-only target state after cutover?  
**Decision:** No permanent hybrid operation in consensus; the target state is a clear cutover to PQ-only operation.  
**Rationale:** Permanent hybrid operation would increase rule complexity, increase attack surface, and make deterministic validation and incident handling harder in the most security-sensitive path. A single post-cutover signature regime is easier to reason about, test, audit, and operate safely.  
**Status:** closed  
**Reference:** Chapters 04, 05

### Q-03
**Question:** Which cutover mechanism is required to preserve validator identity continuity during consensus migration?  
**Decision:** Existing validators must be bound to new PQ consensus keys before cutover, and a voting-power threshold must be satisfied before halt/restart.  
**Rationale:** Safe restart requires unambiguous continuity from existing validator identities to post-cutover PQ keys. Binding plus threshold gating reduces the risk of identity ambiguity, incomplete participation, and unsafe restart conditions, and provides measurable readiness criteria before operational execution.  
**Status:** closed  
**Reference:** Chapters 05, 07

### Q-04
**Question:** What is the target transition model for wallet/tx signatures during PQ introduction?  
**Decision:** The wallet/tx path adopts a hybrid signature model with staged rollout rather than an early hard PQ-only cutover.  
**Rationale:** The wallet/tx path is ecosystem-dependent and includes diverse operational actors with different readiness levels. A staged hybrid model enables earlier participation by advanced actors (for example exchanges and custodians) while preserving operational continuity for slower-moving integrations and end-user tooling.  
**Status:** closed  
**Reference:** Chapters 04, 05

### Q-05
**Question:** Should Terra Classic depend on external wallet/client release cycles for PQ support or establish its own PQ-capable client stack?  
**Decision:** Terra Classic will build a native wallet/explorer ecosystem with explicit PQ compatibility as a long-term target state.  
**Rationale:** Exclusive dependence on third-party release priorities increases strategic and operational risk for user-facing adoption. A native stack improves control over roadmap alignment, release responsiveness, operational ownership, and long-term continuity of PQ-capable user infrastructure.  
**Status:** closed  
**Reference:** Chapters 04, 05

### Q-06
**Question:** Which formal prerequisite must be met before implementation phases A-C can start?  
**Decision:** A formal RFC freeze in preceding Phase Omega is required as start authorization before phases A-C.  
**Rationale:** A mandatory freeze step ensures unresolved conflicts are handled before implementation, preserves scope integrity, and establishes a traceable baseline for downstream delivery and auditing. This reduces rework and prevents silent scope drift under execution pressure.  
**Status:** closed  
**Reference:** Chapters 05, 08

### Q-07
**Question:** Is re-genesis currently accepted as a robust fallback migration strategy for Terra Classic?  
**Decision:** No. Re-genesis is currently treated as an operational dead end and not as a robust fallback strategy.  
**Rationale:** For current state size and operational constraints, no robust proof exists that safe, reproducible export/import can be completed in acceptable and reliable conditions. Depending on such an unproven fallback would shift migration risk into an unvalidated emergency path.  
**Status:** closed  
**Reference:** Chapter 07

### Q-08
**Question:** What minimum gate discipline is mandatory between roadmap phases?  
**Decision:** The defined gate matrix is mandatory, including blocker classes, pass criteria, and re-audit rules.  
**Rationale:** Without binding gate discipline, risk can leak from prototype and migration work into production transitions. A formal gate baseline establishes objective continuation criteria, creates governance traceability, and enforces consistent risk treatment across phases.  
**Status:** closed  
**Reference:** Chapter 06

### Q-09
**Question:** How exactly should the wallet/tx hybrid path be specified regarding key/account types, compatibility windows, and activation logic?  
**Decision:** No final decision yet. Working options include conservative entry, staged activation, and aggressive rollout profiles.  
**Rationale:** The decision depends on balancing security requirements, integration effort, user-experience risk, and operational complexity across heterogeneous ecosystem actors. Finalization requires B1-level architecture and compatibility evidence.  
**Status:** open
**Reference:** Chapters 04, 05, target phase B1

### Q-10
**Question:** Which formal trigger set should govern consensus cutover Go/No-Go decisions beyond minimum threshold criteria?  
**Decision:** No final decision yet. Working options include strict criteria sets, weighted decision models, and multi-stage approval logic.  
**Rationale:** Minimum thresholds alone may not capture full operational readiness and residual risk conditions. The final trigger model must preserve safety/liveness, deterministic behavior, and practical operational controllability in high-pressure cutover conditions.  
**Status:** open
**Reference:** Chapters 05, 07, target phase A4-A5

### Q-11
**Question:** Which governance model should define the relationship between Terra Classic governance and the Phase C carrier entity?  
**Decision:** No final decision yet. Working options include direct governance binding, mandated sub-structure, and hybrid governance models.  
**Rationale:** The relationship must ensure accountability and transparency while preserving operational ability to act and reliable control of critical assets (for example DNS, release accounts, distribution channels). The model must be precise enough to avoid governance ambiguity during incidents.  
**Status:** open
**Reference:** Chapters 05, 08, target phase C2

### Q-12
**Question:** Which additional re-audit regime applies if B3 and B4 are merged into one execution block?  
**Decision:** No final decision yet. Working options include trigger-based mandatory re-audits and case-by-case governance decisions.  
**Rationale:** Merging B3/B4 may improve delivery flow but can compress review separation between implementation and migration logic. The final regime must preserve security assurance without creating disproportionate process overhead or timing risk.  
**Status:** open
**Reference:** Chapters 05, 06, target phase B3-B4

### Q-13
**Question:** How should external PQ monitoring and alignment be operationalized in cadence and decision impact?  
**Decision:** No final decision yet. Working options include periodic cycles, event-driven triggers, and combined operating models.  
**Rationale:** The mechanism must be responsive to external evidence changes without overwhelming governance and technical teams with noise. It must define when new external findings trigger review-only actions versus binding RFC or implementation adjustments.  
**Status:** open
**Reference:** Chapters 05, 08, target phase C3

### Q-14
**Question:** Should a detailed re-genesis emergency specification be finalized now as part of this RFC baseline?  
**Decision:** Deferred. No final specification is adopted in the current RFC baseline.  
**Rationale:** A robust and reproducible technical proof for safe large-state export/import is still missing. The decision is deferred until credible end-to-end evidence under realistic operating conditions is available.  
**Status:** deferred
**Reference:** Chapters 07, 08

### Q-15
**Question:** Should the RFC define a final post-ML-DSA long-term cryptographic profile at this stage?  
**Decision:** Deferred. No final post-ML-DSA profile is fixed in the current RFC revision.  
**Rationale:** External PQ standards and cryptanalysis are still evolving. Prematurely freezing a long-term successor profile would risk locking the roadmap to assumptions that may change with new evidence and standardization outcomes.  
**Status:** deferred
**Reference:** Chapters 04, 08

## Status Scheme and Transition Rules
Status values:
- `open`: question is decision-ready but not finally decided.
- `closed`: question is finally decided and currently binding.
- `deferred`: question is intentionally postponed and linked to a clear re-trigger.

Status transitions:
- `open -> closed` only with documented governance decision and chapter reference.
- `open -> deferred` only with reasoned deferral rationale and explicit re-trigger.
- `deferred -> open` once defined re-trigger occurs.
- Reversal of a `closed` direction decision only via RFC revision under freeze/change-control rules from Chapter 08.
