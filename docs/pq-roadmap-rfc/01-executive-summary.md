## Scope Note
This RFC bundle defines direction, decision logic, and validation paths for the Terra Classic PQ roadmap. It is intentionally not an implementation plan, delivery schedule, or resource commitment.

RFC stands for “Request for Comments.” In this context, the document is published to proactively gather feedback from all relevant Terra Classic stakeholders in order to improve scope clarity, content quality, readability, and practical feasibility before final freeze decisions are made.

# 1) Executive Summary

## Context and Purpose
Terra Classic currently uses cryptographic signatures in multiple security-critical areas. These signatures ensure that only authorized actors can execute valid actions and that the chain maintains a shared, verifiable state. Post-quantum cryptography (PQ, meaning cryptographic methods designed with future quantum-computing risks in mind) becomes relevant because sufficiently capable quantum computers may eventually attack and break classical signature algorithms that are widely used today.

This RFC provides a structured orientation framework for that transition. It defines how migration paths are identified, evaluated, and prioritized, and it describes the framework for audit gates, governance, and feedback up to RFC freeze. At the same time, this version is intentionally not a detailed technical specification and not a release, resource, or timeline commitment.

## Document Structure
In the Terra Classic stack, signatures are used across several areas. In this document, these areas are called paths. The RFC first classifies these paths at a high level, then systematically defines their boundaries and criticality in the following chapters. Based on that, prioritization is derived and justified transparently.

After the executive summary, the document covers the current state and key terminology. This is followed by target states and options, then a phased roadmap. Afterward, audit gates, live-chain migration paths, and governance/feedback process are described. The document concludes with a decision log, where closed and open points are tracked transparently.

## Reading Logic and Boundaries
The RFC is split into a Public Layer and a Technical Layer. The Public Layer explains motivation, structure, and decision logic in accessible language. The Technical Layer deepens path definitions, risks, validation logic, and decision options. Both layers are aligned so readers can enter at different levels of detail without losing the overall thread.

This executive summary does not pre-empt technical detail decisions and does not define a fine-grained implementation sequence. It serves as reading guidance. Detailed argumentation, assessments, and decision proposals follow in the subsequent chapters.
