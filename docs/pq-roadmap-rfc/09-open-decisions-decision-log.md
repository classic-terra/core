# Open Decisions (Decision Log, Formal)

This chapter is the operational decision register of the RFC. It tracks closed, open, and deferred points in a form directly usable for governance, audit, and phase steering. The Decision Log is updated with each relevant RFC revision and is the binding reference for status transitions.

## Closed Direction Decisions

### G-01
**Topic:** Consensus crypto profile  
**Decision:** ML-DSA as target profile in the consensus path.  
**Status:** closed  
**Reference:** Chapters 04, 05  
**Short rationale:** Binding PQ target state for the most security-critical path.

### G-02
**Topic:** Consensus migration mode  
**Decision:** No permanent hybrid operation in consensus; clear cutover to PQ-only state.  
**Status:** closed  
**Reference:** Chapters 04, 05  
**Short rationale:** Reduces rule complexity, attack surface, and operational risk.

### G-03
**Topic:** Consensus cutover mechanics  
**Decision:** Key binding of existing validators to PQ consensus keys plus voting-power threshold before halt.  
**Status:** closed  
**Reference:** Chapters 05, 07  
**Short rationale:** Unambiguous identity continuity as prerequisite for safe restart.

### G-04
**Topic:** Wallet/tx target state  
**Decision:** Hybrid signature path with staged adoption instead of early PQ-only cutover.  
**Status:** closed  
**Reference:** Chapters 04, 05  
**Short rationale:** Ecosystem-sensitive rollout with early usability for technically advanced actors.

### G-05
**Topic:** Client target state  
**Decision:** Buildout of Terra-Classic-native wallet/explorer ecosystem with native PQ compatibility.  
**Status:** closed  
**Reference:** Chapters 04, 05  
**Short rationale:** Reduces dependency on third parties and external release cycles.

### G-06
**Topic:** RFC freeze logic  
**Decision:** Preceding Phase Omega as formal start authorization before A-C.  
**Status:** closed  
**Reference:** Chapters 05, 08  
**Short rationale:** Secures content baseline and prevents scope drift during implementation.

### G-07
**Topic:** Re-genesis assessment  
**Decision:** Re-genesis currently treated as operational dead end, not a robust fallback.  
**Status:** closed  
**Reference:** Chapter 07  
**Short rationale:** Missing proof of safe, reproducible export/import at large state size.

### G-08
**Topic:** Audit-gate baseline rule  
**Decision:** Gate obligation along defined gate matrix including blocker and re-audit rules.  
**Status:** closed  
**Reference:** Chapter 06  
**Short rationale:** Enforces risk-adequate transitions between phases.

## Open Architecture and Process Decisions

### O-01
**Decision question:** Exact design of wallet/tx hybrid path (key types, compatibility windows, activation logic).  
**Options (working state):** conservative entry / staged activation / aggressive rollout.  
**Evaluation criteria:** security, integration effort, UX risk, operational complexity.  
**Target phase:** B1  
**Status:** open

### O-02
**Decision question:** Formal trigger set for consensus cutover Go/No-Go (beyond minimum thresholds).  
**Options (working state):** strict criteria / weighted criteria model / multi-stage approval.  
**Evaluation criteria:** safety/liveness risk, determinism, operational controllability.  
**Target phase:** A4-A5  
**Status:** open

### O-03
**Decision question:** Governance form of carrier-entity relationship in Phase C.  
**Options (working state):** direct governance binding / mandated sub-structure / hybrid model.  
**Evaluation criteria:** accountability, ability to act, asset control, transparency.  
**Target phase:** C2  
**Status:** open

### O-04
**Decision question:** Regime for additional re-audits when B3/B4 are merged.  
**Options (working state):** mandatory on triggers / case-by-case governance decision.  
**Evaluation criteria:** security effect, time impact, review effort.  
**Target phase:** B3-B4  
**Status:** open

### O-05
**Decision question:** External PQ alignment mechanics (cadence, decision impact).  
**Options (working state):** periodic cycle / event-driven / combined.  
**Evaluation criteria:** responsiveness, process overhead, decision quality.  
**Target phase:** C3  
**Status:** open

## Deferred Decisions

### V-01
**Topic:** Specification of a re-genesis emergency path.  
**Reason for deferral:** No robust technical proof yet for safe export/import at Terra Classic state size.  
**Re-trigger for reopening:** Robust end-to-end evidence under realistic operating conditions.  
**Status:** deferred

### V-02
**Topic:** Final long-term profile after ML-DSA (post-ML-DSA options).  
**Reason for deferral:** External PQ development and cryptanalysis still evolving.  
**Re-trigger for reopening:** Relevant new research results or standard changes with concrete risk impact.  
**Status:** deferred

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
