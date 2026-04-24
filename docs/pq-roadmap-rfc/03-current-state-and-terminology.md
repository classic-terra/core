# Current State and Terminology (Informative)

## Which Signature Paths This RFC Covers
Before describing the status quo, this chapter defines the signature paths covered by the rest of the RFC. In this context, a path is not a single code section but a connected technical and operational domain in which signatures are created, transmitted, verified, and used as a basis for decisions.

For Terra Classic, three perspectives are most relevant. First, the consensus path, where validator signatures secure shared chain truth directly. Second, the wallet/tx path, where user transactions are signed and validated in backend flows. Third, the user-facing client path, where signing workflows are visible and operable for end users. In this chapter, the status-quo focus is on the two technically critical core paths, consensus and wallet/tx; the client path is deepened later in target states and roadmap phases.

This boundary definition is important because these paths have different risk profiles and migration effort. A failure in the consensus path has systemic impact on chain stability, while failures in the wallet/tx path more often appear as integration, compatibility, or usability problems. The RFC therefore does not discuss "signatures" in the abstract, but signatures in their specific operating context.

## Status Quo of the Consensus Path
In the consensus path, signatures secure block production and finality between validators. Signature verification in vote-, proposal-, and commit-related flows is therefore part of the network's security-critical core function. If verification in this area is no longer consistent or deterministic, this affects not only individual participants but potentially the stability of the whole system.

In the current Terra Classic context, **Ed25519** is the relevant scheme in the consensus path. The operational scope includes not just verification code, but also connected components for validator operations, key custody, and signature provision. This yields a clear baseline for the RFC: consensus-side signature changes must always be evaluated through safety, liveness, upgrade determinism, and operational stability. For this reason, the consensus path is treated as a high-criticality path in the roadmap.

## Status Quo of the Wallet/Tx Path
In the wallet/tx path, signatures are created and validated for user transactions. This path is strongly ecosystem-shaped: wallets, custody environments, exchanges, APIs, SDKs, and integrations must work consistently in practice. Signature verification is security-relevant here as well, but impacts often first appear in compatibility, user flow, and support overhead.

In the current Terra Classic context, **secp256k1** is the relevant scheme in the wallet/tx path. The baseline is therefore different from the consensus path. In the wallet/tx domain, cryptographic correctness alone is not sufficient; it also matters how broadly new signing behavior can be integrated into existing toolchains and interfaces. For the RFC, this means this path always requires integration and operations argumentation alongside security argumentation.

## Why This Baseline Is Decisive for the Roadmap
The roadmap is built on this status-quo analysis. Only clear path separation allows priority, risk, and migration logic to be justified transparently. Without this separation, technical decisions would easily become over-generalized and harder to compare.

This chapter therefore provides the shared foundation for following chapters: target states and options build on the same path boundaries, audit gates follow the same risk profiles, and governance decisions are evaluated along the same structure.

## Terms for Following Chapters
- **Path** - In this RFC, a path is a connected technical and operational domain where signatures are created, transmitted, and verified.
- **Consensus path** - The consensus path includes signing and verification flows that secure shared chain truth directly, especially coordinated validation and production of blocks.
- **Wallet/tx path** - The wallet/tx path includes user-side transaction signatures and their validation in backend and protocol flows. These signatures ensure individual transactions are authorized by individual users.
- **Client path** - The client path is the user-facing layer where signature processes are visible and operable in wallets and interfaces. On the client side, signature generation is more central than verification.
- **Cryptographic signature** - A cryptographic signature is digital proof binding sender and content.
- **Signature verification** - Signature verification is the check that message, signature, and public key are logically consistent.
- **Private key** - The private key is a party's secret key and is used to produce signatures.
- **Public key** - The public key is the distributable counterpart to the private key and is used for verification.
- **Authenticity** - Authenticity means a message truly comes from the claimed sender.
- **Integrity** - Integrity means a message was not altered unnoticed after signing.
- **Validator** - A validator is a network participant that creates and verifies signatures in the consensus process.
- **Voting Power (VP)** - Voting power is a validator's consensus weight.
- **Safety** - Safety in consensus means the network does not accept contradictory valid states.
- **Liveness** - Liveness means the network continues producing blocks under valid operating assumptions and does not halt permanently.
- **Upgrade determinism** - Upgrade determinism means a protocol or software switch yields identical results for all participants under identical conditions.
- **Cutover** - A cutover is a planned switch point between two operating modes.
- **Audit gate** - An audit gate is a mandatory validation and release checkpoint before moving to the next phase.
- **In-place upgrade** - An in-place upgrade is a migration path where the existing live chain is continued in a controlled way while being technically switched over.
- **Re-genesis** - Re-genesis is the fallback path if in-place upgrade is not viable.
- **Key binding** - Key binding is the traceable assignment of a new key to an existing validator identity.
- **CometBFT** - CometBFT is a Terra Classic software component where core consensus and validator signing flows are anchored.
- **Cosmos SDK** - Cosmos SDK is the modular application framework Terra Classic uses for core chain logic, transaction handling, and module composition.
- **wasmd** - wasmd is the CosmWasm integration layer used by Terra Classic to run and manage smart contracts in the chain runtime.
- **IBC** - IBC (Inter-Blockchain Communication) is the protocol stack used to exchange packets and verify state proofs across connected chains.
- **ibc-go** - ibc-go is the Go implementation of IBC modules and middleware used by SDK-based chains, including Terra Classic.
- **Ante** - In SDK context, Ante is the transaction pre-check layer before execution.
- **RFC** - RFC stands for "Request for Comments" and in this context means a structured discussion and decision document that makes direction, options, risks, and open points transparent before binding delivery decisions are made.
- **Ed25519** - Ed25519 is the relevant signature scheme in the current Terra Classic consensus path.
- **secp256k1** - secp256k1 is the relevant signature scheme in the current Terra Classic wallet/tx path.
- **PQ (Post-Quantum)** - PQ denotes cryptographic methods intended to be more robust against threat models with capable quantum computers than many widely used classical methods.
- **ML-DSA** - ML-DSA is defined in this plan as the target profile for the consensus path.
