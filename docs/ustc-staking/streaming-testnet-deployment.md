# USTC streaming testnet operator handoff

This is a testnet candidate, with zero default emission and no fixed APR. Use the externally generated candidate manifest and SHA256SUMS to identify source and binaries. Engineering test evidence does not submit a public upgrade proposal or authorize a production activation.

## Observed target and upgrade selection

Read-only checks on 2026-10-14 found `rebel-2` at the supplied LCD/RPC, application 4.0.1, commit `9a5ee563874ce3906c3ca7069f0160de51f89c40`, SDK 0.53.6, no `ustcstaking` entry in the persisted module-version map, no current upgrade plan, and no applied `v14_5` height. These are observations, not permanent deployment configuration.

Immediately before proposing an upgrade, independently confirm chain ID, predecessor revision, module versions, current plan and applied-plan history. The observed absent-store case selects `v14_5`, which adds the USTC store. `v14_5_ustcstaking_existing` is an alternative for an already-mounted v1 store, not a later sequential upgrade. Do not use an add-store plan against an existing store or choose an already-applied plan.

## Binary package

The locally built Linux AMD64 binary is dynamically linked to the matching Wasm VM v3.0.8 library. The release package includes that library and a `run-terrad` launcher setting its library path. Test the package on the actual validator OS before the upgrade. The disposable campaign uses Ubuntu 22.04; the previous Ubuntu 18.04 localnet image cannot load the library. A bare binary copied without its library is insufficient.

Verify `SHA256SUMS`, then run `./run-terrad version --long --home <temporary-empty-home>`. Its embedded commit must match the manifest. Use the same launcher or an equivalent `LD_LIBRARY_PATH` in the validator's service configuration. Retain the predecessor binary and its corresponding library separately.

## Coordinated upgrade

1. The testnet coordinator chooses an upgrade height after the governance voting window and publishes the complete proposal, candidate SHA, package checksums, and tested service command. Query the gov module authority from the target chain; never substitute a wallet address.
2. Validators prepare the package and verify metadata/dependencies before the height. Observe the passed proposal and scheduled plan.
3. At the planned halt, stop the predecessor cleanly. Preserve database, config and signing state using the chain's established snapshot procedure. Keep validator signing keys on their owning host.
4. Switch the service to the candidate and matching library, preserving the same home. Do not reset the database, regenerate genesis, or start a second instance of a validator key.
5. Require the applied upgrade at the intended height, module version 2, continuing blocks beyond the height, matching app hashes at a common fixed height, and successful native queries. Check `terrad query ustcstaking validate-state` exits successfully with `valid=true`.
6. Verify restart/catchup on one validator before declaring the rollout accepted. Preserve raw logs and transaction/upgrade results.

A post-migration downgrade is not a routine rollback: the old binary cannot be assumed to read the new schema/store. Recovery must follow the coordinated chain snapshot and signing-state procedure; never blindly restore an old priv_validator_state after signing later blocks. A safe application pause is a governance full-params update, not a database rollback.

## Start the rewards program

Fresh defaults are tier 1, 21-day unbonding, 1x multiplier, rate `0 uusd/day`. Stake eligibility is immediate; users can claim earned rewards anytime and stop earning when they start unbonding. Principal withdrawal is explicit after the position's snapshotted wait. Active claims retain fractional earnings; closing a position truncates its remaining sub-uusd fraction.

Governance must separately approve the release rate and reward-pool funding. Funding alone leaves the default zero rate unchanged. Query the complete current Params and preserve all existing fields/tiers in MsgUpdateParams. `reward_rate_per_day` is integer uusd for the entire program per 86,400 seconds; it is not a per-user amount or interest rate. Funding transfers existing USTC from the community pool using MsgFundRewards. Top-ups add runway without instant allocation or backpay; idle, paused and exhausted intervals do not accumulate a payout for future entrants.

Do not send rewards directly to the module account. Monitor approved unallocated reserve U, allocated protection A, pending claims, physical bank backing B, total shares and rate. Require B >= U+A and exact liabilities <= A; principal custody remains separate. Governance dust return never releases U or earned claims.

## API documentation deferral

Set `swagger = false` in the `[api]` configuration until Swagger/statik regeneration is reviewed. This does not disable REST/gRPC routes. Smoke-test params, reward state, positions and aggregate validation through the actual interfaces. Buf authentication is only a build-time documentation issue. Public dependencies may not need a token; if authenticated generation is desired, refresh local credentials with `buf registry login buf.build`, then run `make apply-swagger`.

## Evidence

Corrected validation notes: [streaming validation](phase-2-streaming-rewards-validation.md). Campaign behavior: [seven-validator streaming campaign](phase-2-streaming-rewards-localnet-testing.md). Durable campaign reports and candidate metadata are generated under `/home/sofoli/lbunproject/ustc-campaign-evidence/streaming-20261014/`; release archives are under `/mnt/d/ustc-streaming-release-campaign/` on the build host. Never put generated node homes or private keys in a release archive.
