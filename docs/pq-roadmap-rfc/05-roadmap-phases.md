# Roadmap Phases (Formal) {#ch-roadmap-phases}

## Phase Omega: RFC Freeze and Start Authorization (Preceding)
Phase Omega is the upstream completion of RFC drafting before any technical implementation starts. It is not a delivery plan and not an implementation phase, but the formal transition from "discussion and variants" to a "binding implementation baseline." Without a completed Omega gate, implementation phases A through C do not begin.

The purpose of this phase is to establish a robust content baseline for all downstream technical decisions. Open conflicts are not left implicit; they are explicitly decided, deferred, or rejected. At the same time, this phase defines which parts of the RFC are frozen and under which conditions post-freeze changes are allowed.

### Work Package O0: RFC Launch and Formal Comment Solicitation
O0 is the formal start package for this RFC process. The RFC was initially drafted by the author in single-author mode and is then presented and published through a Terra Classic governance proposal as the formal kickoff for stakeholder review.

The proposal explicitly asks whether Terra Classic stakeholders should formally enter the RFC comment process and requests relevant stakeholder groups (see Chapter {{chapter:governance-process}}, Stakeholder Groups) to submit comments through the defined channels (see Chapter {{chapter:comment-log}}, Submission Channel).

The mandatory result of O0 is a published kickoff baseline with explicit governance-level invitation to comment, clear stakeholder addressing, and active comment intake.

### Work Package O1: Consolidation and Decision Closure
O1 formally closes the RFC feedback cycles. Comments and opposing positions from stakeholder rounds are classified and transferred into document state with reasoned decisions. The Decision Log is maintained as a binding reference, including status per point (closed, deferred, rejected) and traceable rationale.

The mandatory result of O1 is a content-consistent RFC state without hidden conflicts in core questions. What remains open is explicitly marked as deferred and assigned to a clear follow-up process. O1 is formally closed through a dedicated governance proposal that confirms decision closure and the consolidated RFC baseline.

### Work Package O2: Phase and Gate Consistency Review
O2 ensures consistency of implementation logic across chapter boundaries. It reviews target states, scope boundaries, audit gates, and exit criteria in phases A through C, as well as compatibility with governance and migration chapters.

The mandatory result of O2 is a documented consistency review showing clearly that there are no blocking conflicts between Informative and Formal chapters, and no silent scope shifts being carried into implementation.

### Work Package O3: Governance Freeze and Reference Publication
O3 executes the formal governance event for freeze (for example proposal and vote) and converts the consolidated state into a frozen reference version. This version is the binding basis for implementation in phases A through C.

Part of O3 is defining post-freeze change rules: which changes count as editorial only, and which require a new RFC revision and renewed governance approval.

### Dependencies and Order
Omega packages run in clear sequence. O1 depends on a completed kickoff and formal comment solicitation from O0. O2 depends on formally approved O1 closure and the resulting consolidated baseline. O3 depends on a passed consistency review from O2. This sequence ensures unresolved conflicts do not enter a formal freeze decision.

### Preliminary Exit Criteria (Omega Go/No-Go)
- O0 publishes the RFC kickoff through governance and formally opens stakeholder comment intake through defined channels.
- O1 is complete, the Decision Log is finally consolidated including explicitly marked deferrals, and formal closure is approved through governance proposal.
- O2 documents and confirms consistency between Informative and Formal chapters and between phase and gate logic.
- O3 is formally approved and published as frozen reference version.
- Post-freeze change rules are bindingly documented.

### Out of Scope in Phase Omega
- Implementation work in code.
- Commitments on timeline, budget, or resources.
- Premature detailed technical design decisions that belong to implementation phases A through C.

## Phase A: Consensus Path PQ-Enabled
Phase A establishes the technical decision and implementation foundation for a PQ-enabled consensus path. The RFC intentionally works in downstream chapters with the premise that a PQ-resistant CometBFT fork is fundamentally feasible. At the same time, this assumption is not taken for granted in Phase A; it must be explicitly validated. The feasibility study acts as an early invalidation mechanism: if it is negative, the RFC in its current form is stopped and moved into an alternative roadmap.

Implementation in Phase A is split into clearly bounded work packages that can be delivered by internal and/or external software development providers. The focus is secure transition of the consensus path. End-user wallet/tx rollout and broad user-client ecosystem expansion are not part of this phase.

### Work Package A1: CometBFT Fork Feasibility Study (Gate 0)
A1 robustly tests whether a PQ-resistant consensus path based on a CometBFT fork is technically, security-wise, and operationally viable for Terra Classic. The study covers not only the fork core but the full dependency chain of the consensus stack: primarily CometBFT, additional interfaces extending into Cosmos SDK, and potential effects on smart-contract layer integrations including wasmd-near interfaces.

The mandatory result of A1 is a documented Go/No-Go evaluation with clear no-go criteria, risk classes, and invalidation signals. A1 also includes structured decomposition of follow-up work into consistent packages for prototype, audit, and migration.

### Work Package A2: PQ Prototype on Independent Testnet
A2 delivers a working prototype of a PQ-resistant CometBFT fork using ML-DSA. The prototype must prove technical integration in a realistic stack and therefore includes required dependencies extending into Cosmos SDK and wasmd.

The prototype runs on a genesis testnet independent of `rebel-2`. Migration of an existing live chain is not the focus of this package. The focus is functional behavior, integration consistency, reproducible operation, and robust measurement data for next decisions.

### Work Package A3: Audit Gate for Prototype (Gate 1)
A3 is the formal security and quality gate between prototype and migration development. The prototype is independently reviewed before migration components for a production path are built.

Audit focus includes safety/liveness impact, cryptographic correctness of ML-DSA integration, deterministic behavior in critical consensus paths, and failure modes under load and malformed inputs. The output is a documented Go/No-Go decision for transition to A4.

### Work Package A4: Migration Components for Consensus Path
A4 develops all technical components and operational strategies required for orderly transition to the consensus target state. This includes upgrade and migration handlers, mechanisms to register PQ signatures or PQ keys for existing validators, and formal rule sets for cutover, activation, and safe post-switch operation.

This package ensures the transition is not only technically possible but operable within production-grade governance and operations logic. The focus is deterministic behavior in the upgrade window and unambiguous rules for valid signatures before, during, and after cutover.

### Work Package A5: Audit Gate for Migration Components (Gate 2)
A5 is the final audit gate before productive consensus-path transition. It reviews correctness and tamper resistance of registration and migration mechanics, consistency of cutover rules, and secure deterministic post-switch startup.

The result is a robust Go/No-Go recommendation for the production migration path. Critical findings block transition to downstream rollout and governance steps until fixed and successfully re-audited.

### Dependencies and Order
Packages are intentionally sequential. A2 requires a positive A1 result. A4 requires a passed audit gate from A3. A5 is mandatory before any production migration decision. This sequence prevents unverified assumptions from early development from leaking into live migration.

### Preliminary Exit Criteria (Phase A Go/No-Go)
- A1 is complete and includes an explicit continuation decision based on traceable no-go criteria.
- A2 delivers a reproducibly running prototype on an independent genesis testnet.
- A3 passes without critical blockers for entering migration development.
- A4 fully specifies and implements migration components and cutover logic.
- A5 passes without critical blockers for production migration path.

### Out of Scope in Phase A
- Wallet/tx rollout for end users.
- Buildout of broad PQ-native user-client ecosystem.
- Delivery details such as owners, dates, or budgets.

## Phase B: Wallet/Tx Path PQ-Enabled
Phase B creates the technical implementation foundation for the hybrid signature path in wallet/tx. The goal is not an early hard PQ-only cutover, but an orderly parallel capability of classical signature verification and ML-DSA in the production-relevant transaction path.

The baseline differs from Phase A: Cosmos SDK already provides a fundamentally suitable framework for alternative signature and verification mechanisms via abstract account/key types, flexible data structures, customizable tx types and extensions, and a programmable ante handler. Even though secp256k1 is predominant in current transaction signatures, Phase B therefore does not need a separate upstream feasibility gate. It starts with coordinated requirements and architecture planning.

### Work Package B1: Coordinated Planning Phase (Requirements)
B1 consolidates functional and technical requirements needed to reach the hybrid signature target state. Core focus is target architecture for ante handler and signature verification, plus required extensions in key/account types, tx extensions, and adjacent interfaces.

The mandatory result of B1 is an aligned implementation and migration work backlog. This includes clear compatibility boundaries, error and fallback behavior, and a robust test approach for subsequent prototype operation.

### Work Package B2: Development, Component Transition, and Prototype Test Run
B2 implements hybrid validation including ML-DSA support across all affected wallet/tx components along the real transaction flow. The goal is not isolated integration, but consistent transition of dependent components in combined operation.

As technical proof, an end-to-end prototype is operated on a dedicated testnet. This package is successful when hybrid signature validation runs stably and reproducibly with traceable operational behavior.

### Work Package B3: Audit Gate for Hybrid Wallet/Tx Path
B3 is the mandatory security and quality gate after implementation and prototype operation. It reviews correctness of signature validation, security properties in ante and verification paths, and robustness under load and failure scenarios.

The result is a documented Go/No-Go decision for transition to migration elaboration. Critical findings block progression until remediation and renewed review.

### Work Package B4: Migration Components for ML-DSA Introduction
B4 specifies and implements migration components for orderly ML-DSA introduction in wallet/tx. This includes introduction strategy, activation logic, compatibility windows, and operational transition for hybrid mode.

B4 intentionally has no separate additional mandatory audit gate as an independent milestone. Depending on observed complexity, B3 and B4 may be merged or tightly coupled, provided shared Go/No-Go logic remains documented.

### Dependencies and Order
Packages are sequential. B2 requires a completed B1 requirements package. B4 requires a passed audit gate from B3 unless B3 and B4 are intentionally run as one combined block. This order ensures migration elaboration is based on validated, not merely implemented, mechanisms.

### Preliminary Exit Criteria (Phase B Go/No-Go)
- B1 is complete and provides aligned requirements including architecture and test baseline.
- B2 delivers reproducible end-to-end prototype proof on a dedicated testnet.
- B3 passes without critical blockers and allows transition into migration elaboration.
- B4 is implementation-ready; if B3/B4 are merged, a shared documented Go/No-Go decision applies.

### Out of Scope in Phase B
- Full rollout of a Terra-Classic-native user-client ecosystem.
- Delivery details such as owners, dates, or budgets.

## Phase C: PQ-Native Clients
Phase C translates technical PQ capability from wallet/tx into a durable Terra-Classic-native user ecosystem. Unlike phases A and B, the structure here is intentionally not strictly sequential. Core work packages can start in parallel and be iteratively aligned.

The core objective is that Terra Classic does not remain dependent on third parties or external release cycles in the broader ecosystem for PQ-capable wallet, explorer, and user-facing infrastructure.

### Work Package C1: Terra-Classic-Native Wallet/Explorer Stack
C1 includes requirement definition, procurement, and commissioning of an own wallet/explorer stack that fully reflects the hybrid and later expanded PQ-capable wallet/tx path.

Development may be performed by external providers, but is operated as Terra-Classic-owned infrastructure under a public/open license model. The goal is an independently operable, openly licensed, and traceably maintainable stack for retail and individual users.

### Work Package C2: Public-Law Carrier Entity and Operational Ownership
C2 establishes a public-law entity that institutionally carries the stack. This entity defines requirements, commissions implementation, manages ownership, and is accountable for monitoring and hosting.

Scope includes not only web instances and backend nodes, but also ownership/control of critical digital assets such as DNS names, developer accounts, and publication channels (for example app stores, package registries, repositories). C2 also includes drafting a constitutional-style baseline relationship between this entity and Terra Classic governance module.

### Supporting Work Package C3: External PQ Standards Monitoring and Alignment
C3 runs as a permanent parallel stream to C1 and C2. Its purpose is active contact and relationship management with relevant actors in Cosmos, Ethereum, and broader blockchain ecosystems to monitor PQ standardization early, identify divergence in time, and proactively align Terra Classic decisions.

This stream is security-strategically required: ML-DSA is used as current target algorithm, but without assuming mathematical finality against future cryptanalytic breakthroughs (quantum or classical). Terra Classic decisions must therefore remain revision-capable and responsive to external evidence growth.

### Dependencies and Parallelization
C1 and C2 are fundamentally parallelizable and need not run as a rigid sequence. C3 accompanies both continuously. Where operational coupling emerges (for example release ownership, operational obligations, compliance constraints), synchronization points are documented as governance decisions rather than enforced as implicit ordering.

### Preliminary Exit Criteria (Phase C Go/No-Go)
- C1: Requirements are approved, commissioning is completed, and a functioning publicly licensed wallet/explorer stack is demonstrable.
- C2: The carrier entity is formally established and demonstrably assumes ownership, operational responsibility, and asset control (DNS, developer accounts, distribution channels).
- C2: The baseline relationship between carrier entity and Terra Classic governance module is bindingly formulated and published.
- C3: A running external monitoring and alignment process is established, with documented contacts, review cycles, and feedback into technical decisions.

### Out of Scope in Phase C
- Detailed planning of individual procurement/tender procedures at operational level.
- Full anticipation of all future PQ algorithm shifts; Phase C establishes institutional and technical adaptability for that.
