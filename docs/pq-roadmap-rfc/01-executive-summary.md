# Executive Summary (Informative) {#ch-executive-summary}

## Context and Purpose
Terra Classic currently uses cryptographic signatures in multiple security-critical areas. These signatures ensure that only authorized actors can execute valid actions and that the chain maintains a shared, verifiable state. Post-quantum cryptography (PQ, meaning cryptographic methods designed with future quantum-computing risks in mind) becomes relevant because sufficiently capable quantum computers may eventually attack and break classical signature algorithms that are widely used today.

This RFC provides a structured orientation framework for that transition. It defines how migration paths are identified, evaluated, and prioritized, and it describes the framework for audit gates, governance, and feedback up to RFC freeze. At the same time, this version is intentionally not a detailed technical specification and not a release, resource, or timeline commitment.

## Goal in Plain Language
The strategic goal is to quantum-harden Terra Classic in a staged and auditable way, so the chain can continue secure operation even as long-term cryptographic risks evolve.

In simple terms: Terra Classic wants to prepare before cryptographic break risk becomes urgent, instead of reacting after ecosystem damage. The roadmap therefore focuses first on the highest-risk path (consensus), then on transaction signing for users/integrators, and finally on a durable Terra-Classic-native client ecosystem.

## Document Structure
In the Terra Classic stack, signatures are used across several areas. In this document, these areas are called paths. The RFC first classifies these paths at a high level, then systematically defines their boundaries and criticality in the following chapters. Based on that, prioritization is derived and justified transparently.

After the executive summary, the document covers the current state and key terminology. This is followed by target states and options, then a phased roadmap. Afterward, audit gates, live-chain migration paths, and governance/feedback process are described. The document concludes with a decision log, where closed and open points are tracked transparently.

## Risks, Impacts, and Benefits
### If the roadmap is implemented
- Benefits: reduced long-term cryptographic exposure, earlier security option for advanced actors, stronger strategic independence from external client release cycles.
- Risks: migration complexity in consensus-critical paths, temporary operational overhead, and IBC continuity risk due to external counterparty dependency.
- Operational impact: larger signatures and additional verification logic may change resource and performance profiles; impacts must be measured and audited before production transitions.

### If the roadmap is not implemented
- Risk concentration remains on classical cryptography over longer horizon.
- Terra Classic becomes more dependent on external timing for future emergency migration pressure.
- Delayed preparation can force compressed, higher-risk transitions later under less favorable conditions.

### Positioning versus other chains
- Advantage potential: earlier preparedness and governance-visible audit discipline can improve trust for security-sensitive operators.
- Constraint: advantage depends on execution quality and ecosystem coordination, especially where interchain dependencies exist.

## How Development and Rollout Work (Layman Terms)
- The roadmap is not "one giant overnight switch."
- Work is split into packages and phases, each with explicit audit checks before the next risk step.
- New components are first built and tested in controlled test environments.
- Only after successful testing and gate approval can production migration decisions be made.
- For IBC-related changes, counterparties must also be coordinated because some required upgrades happen outside Terra Classic.
- If external readiness is partial, predefined fallback operating modes are used instead of forced unsafe cutover.

## Reading Logic and Boundaries
The RFC is split into two chapter categories: Informative and Formal. Informative chapters explain motivation, structure, and decision logic in accessible language. Formal chapters deepen path definitions, risks, validation logic, and decision options for technical and governance decisions.

This formal/informative split is meant as orientation, not as exclusion. Non-technical readers are explicitly encouraged to read formal chapters as well; these chapters are intended to remain readable even when they are technically precise.

This executive summary does not pre-empt technical detail decisions and does not define a fine-grained implementation sequence. It serves as reading guidance. Detailed argumentation, assessments, and decision proposals follow in the subsequent chapters.
