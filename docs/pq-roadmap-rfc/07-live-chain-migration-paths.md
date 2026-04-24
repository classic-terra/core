# Live-Chain Migration Paths (Formal)

## Positioning of This Chapter
This chapter is categorized as formal. It provides a more detailed view of the consensus cutover path and explains how emergency scenarios are evaluated if the primary migration path cannot be executed safely or in time.

It is not a delivery plan, but a technical orientation for transitioning from current consensus operation to PQ-based consensus operation.

## Primary Path: In-Place Cutover
The primary migration path is a coordinated in-place upgrade with planned chain halt and subsequent restart on the forked, PQ-enabled consensus stack. This path prioritizes live-chain continuity and minimizes ecosystem disruption.

A prerequisite is that preparatory safety conditions from Phase A are satisfied, especially valid and robustly reviewed binding from existing validator identities to new PQ consensus keys.

## Core Element: Binding Existing Validators to New Consensus Keys
The security-critical core of cutover is unambiguous assignment of "existing validator -> new PQ consensus key." Without this mapping, post-switch operation cannot verify which new key belongs to which previous validator identity.

Binding is handled as a prepared and traceable registration process:
- Each existing validator registers its new PQ consensus key through the defined mechanism.
- The process enforces uniqueness and prevents conflicting duplicate mappings.
- A final deterministic snapshot of valid bindings is produced before cutover.
- Cutover release requires sufficient registered voting power (per Phase A logic).

## Consensus Cutover Sequence (Informative)
1. Activate binding window and run ongoing PQ consensus key registration.
2. Validate binding quality and reach required voting-power threshold.
3. Create and verify final binding snapshot.
4. Execute coordinated chain halt at defined switch point.
5. Deploy/start PQ-enabled consensus stack with approved snapshot.
6. Resume block production with PQ consensus keys.

## Why No Direct Immediate Chain Halt Without Binding
An immediate halt and switch without prior binding is not a robust path because secure identity continuity for validators would be missing. This increases risk of misassignment, operational failure, and disputes during validator activation after restart.

## Re-Genesis Assessment: Currently Not a Robust Fallback
For Terra Classic, re-genesis is currently not assessed as a practical fallback but as an operational dead end. For the existing chain state in the range of hundreds of gigabytes, there is still no robust proof that safe, reproducible genesis export plus stable import can be done in acceptable time.

Available experience reports indicate that even under extreme RAM and disk-I/O conditions, genesis import was aborted after long runtime (for example around 24 hours). Therefore there is currently no proof that re-genesis is a safe emergency strategy under realistic operating conditions.

Consequence for this RFC: migration must not depend on re-genesis as a reliable rescue path. Risk reduction must be achieved primarily inside in-place cutover itself (binding quality, gate discipline, deterministic snapshot, coordinated operational transition).

## Decision Criteria and Emergency Principles
The decision on in-place cutover follows a strict Go/No-Go principle:
- Safety before speed.
- Determinism before ad-hoc workarounds.
- Documented governance decision before operational shortcuts.

If a Go cannot be robustly justified, no unsafe cutover is forced. Instead of switching to a currently unvalidated re-genesis path, hardening, re-audit, and renewed decision cycle are required.
