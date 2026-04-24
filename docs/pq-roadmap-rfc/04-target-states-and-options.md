# Target States and Options (Formal)

## Consensus Target State
For the consensus path, the target state is a clear PQ-only operation using ML-DSA from a defined switch day (Day X). This means that after cutover, the consensus path should not run with two signature regimes in parallel as a permanent mode, but should have one unambiguous end state.

To make the target state traceable, the RFC considers two options. Option A is a direct cutover into PQ-only consensus mode from Day X. Option B is a hybrid transition mode in which classical and new signature methods are accepted in parallel for a limited time in the consensus path.

The hybrid model is not adopted as the consensus target state because it expands rule complexity in the most security-critical path and therefore increases complexity, attack surface, and coordination risk. In addition, hybrid operation creates extra testing, audit, and operational overhead because multiple valid signature modes must be processed correctly and deterministically at the same time. The RFC therefore prioritizes a clear end state with unambiguous verification behavior.

## Wallet/Tx Target State
In the wallet/tx path, the target state is a hybrid transition path rather than an early hard PQ-only cutover. Classical and PQ-resistant signatures are supported in parallel during a defined transition period, so adoption can be matched to security needs, technical maturity, and operational capability of each participant group.

The focus is therefore on orderly introduction, clear compatibility boundaries, and transparent migration logic for users and integrators. In the early phase, support for PQ-resistant methods in client and user-facing environments may still be patchy, while UX, tooling, and recovery processes mature only in later RFC phases.

This target state therefore prioritizes staged adoption: security-critical and technically advanced actors such as CEXs, custody providers, and infrastructure projects can use PQ signatures early in production and secure their systems earlier. Less technical retail participants can follow step by step as wallet ecosystem, standards, and usability mature.

## PQ-Native Client Target State
The target state for PQ-native clients is the long-term buildout of a Terra-Classic-native wallet and user-facing client ecosystem with native PQ compatibility. Signature generation, signature verification, and key management should become available and practical for end users early, instead of waiting for external maturity cycles in the broader Cosmos or blockchain environment.

Buildout happens with active involvement of infrastructure actors already operating productively in the Terra Classic ecosystem and having proven operational maturity. This ensures that the rollout of PQ-compatible wallet systems does not depend on third parties, large external ecosystem actors, or externally set standards in a still dynamic PQ landscape.

The target state is that individual actors and retail participants within Terra Classic can participate in PQ-friendly signature methods as early as possible to protect their assets, without being blocked by potentially slow PQ adoption outside Terra Classic.

## Intentionally Open Points
Despite clear core direction decisions, this chapter intentionally keeps some points open that should only be finalized through further technical elaboration and stakeholder alignment. This includes concrete integration details per ecosystem component, exact transition conditions for individual sub-paths, and the granularity of rollout and communication logic.

All open points are tracked explicitly in the Decision Log and marked as open, closed, or deferred. This keeps transparent which target states are already binding and where decision work remains.
