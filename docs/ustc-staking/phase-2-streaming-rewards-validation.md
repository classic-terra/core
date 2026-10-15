# USTC Streaming Rewards Validation Evidence

Validation was run against the uncommitted implementation worktree on 2026-10-13 and 2026-10-14. This is engineering evidence, not a release approval.

## Review follow-up (2026-10-14)

The earlier evidence below describes the initial dirty implementation, not the final candidate. Senior review found and corrected:

- Stake admission now projects the future reserve/index against the post-stake total shares. A backed, invariant-valid boundary fixture reproduced an admitted future BeginBlock failure before the fix.
- Missing runtime RewardState is corruption; the getter no longer fabricates empty balances and a new clock.
- A new independent multi-user reference test runs three seeds with 1,200 operation choices each, plus adversarial prefixes. It tracks rational interval earnings, each wallet/principal/payout, reserve and allocation, rounding, cohort changes, snapshots, pruning, donations, and budget conservation. Its bank is explicitly a fixture; real bank/distribution tests remain separate.
- The app integration fixture now includes USTC genesis, checks InitChain errors, and flushes the SDK v0.53 finalize cache before its deliberate genesis-only Commit. Upgrade corruption fixtures preserve existing account numbers.
- The original seed-1 import panic was traced to retained distribution height 31 being processed after restarting at height 1. Simulation-after-import now performs a zero-height export and restores full-module operations, with no FuzzSeed/evidence suppression. Export itself now retrieves committed height and time, including after process restart, instead of using an empty SDK context header.
- Strict continuous simulation tests reject skips and incomplete block counts. The original corpus contains three intentional empty-validator outcomes (582: height 34; 7601778: height 27; 89182391: height 48). They have explicit terminal-state tests and do not count as 50-block import/export evidence. Short CI runs both the 35-seed continuous corpus and the three terminal fixtures.

The corrected original seed 1 completed a 50-block ordinary-height round-trip with an imported block, and 50 blocks before plus 50 blocks after zero-height import, followed by cold reopen/export: exit 0, 338.157 seconds. Dedicated terminal regressions passed in 99.712 seconds. Both spec/security and independent code-quality reviews approved the corrected source.

Final sweeps and operational proof are collected outside the source tree:

- `/mnt/d/ustc-streaming-release-campaign/sim-final/`
- `/mnt/d/ustc-streaming-release-campaign/final-affected-suites.log`
- `/home/sofoli/lbunproject/ustc-campaign-evidence/streaming-20261014/`

The candidate manifest and campaign report at those locations identify the exact source SHA and binary hash. Check their terminal results; an in-progress run or runner exit without executed assertions is not a pass.

Swagger regeneration is an accepted documentation deferral for the testnet candidate at the user's direction. The binary and generated Go/gateway APIs do not depend on fetching Buf schemas at runtime. Disable Swagger UI and smoke-test CLI, gRPC and REST. Do not publish the old embedded schema as current streaming API documentation. This deferral does not waive runtime accounting, upgrade or consensus checks.

## Initial implementation checks (historical)

- `make proto-gen` — passed; generated protobuf Go and gateway artifacts.
- `go test ./x/ustcstaking/... ./app/upgrades/ustc_staking -count=1` — passed before adding the alternate existing-store upgrade registration.
- `go test ./app/upgrades/ustc_staking -count=1` — passed after adding that registration.
- `go test ./app -run 'Test.*(USTC|Streaming)' -count=1` — passed, including after wiring both upgrade alternatives.
- `go test ./custom/bank/simulation/... -count=1` — passed.
- `go vet ./x/ustcstaking/... ./app/upgrades/ustc_staking` — passed.
- `go vet ./app` — passed.
- `git diff --check` — passed.
- `make test-sim-nondeterminism` — passed (`TestAppStateDeterminism`).
- `go test ./app -run '^TestAppImportExport$' -Enabled=true -NumBlocks=50 -BlockSize=200 -Commit=true -Seed=32 -Period=5 -v -timeout 24h` — passed on 2026-10-14 with output discarded to avoid storing the verbose trace.
- `make test-sim-import-export SIM_LOG_DIR=/mnt/d/ustc-streaming-rewards-sim/import-export` — passed on 2026-10-14 (exit 0); 38 per-seed stdout logs were retained on D: (about 6.3 GB total). Review found only 35 PASS and 3 SKIP; those skipped cases did not execute export/import comparisons.
- `go test ./app -run '^TestAppSimulationAfterImport$' -Enabled=true -NumBlocks=50 -BlockSize=200 -Commit=true -Seed=1 -Period=5 -Verbose=false -timeout 24h` — passed after focusing the test harness on USTC operations (exit 0, 567.481 seconds).
- `go test ./x/ustcstaking/keeper -run '^$' -bench BenchmarkStreamingAccrual -benchtime=1x -count=1` — passed: one position 61,603 ns/op; 100,000 positions 55,703 ns/op. This supports constant work with respect to the position store.
- `go test ./x/ustcstaking/... -count=1` — passed on 2026-10-14, including the new 3,000-step deterministic accrual test against an independent rational reference model.
- `go test -mod=readonly ./x/ustcstaking/simulation -count=1` — passed after the simulation harness changes.
- `make test-sim-after-import SIM_LOG_DIR=/mnt/d/ustc-streaming-rewards-sim/after-import-final-100accounts` — passed (runner exit 0; all 38 configured seeds passed with no skips). Logs are under `/mnt/d/ustc-streaming-rewards-sim/after-import-final-100accounts/sim-logs-529725432/`.
- `make test-sim-multi-seed-short SIM_LOG_DIR=/mnt/d/ustc-streaming-rewards-sim/full-app-short-terminal-skip-ok` — passed (runner exit 0; 35 seeds passed, and 3 seeds—582, 7601778, and 89182391—stopped with the Cosmos SDK's expected skip after randomized operations exhausted the validator set; no seed failed). Logs are under `/mnt/d/ustc-streaming-rewards-sim/full-app-short-terminal-skip-ok/sim-logs-1530260851/`.
- `make build` — passed. The stripped x86-64 Linux binary is `build/terrad`; SHA-256 is `f7756f2ac1d81dcb3670c3e50061c4a2544c452829825c4cc02dcbd236fbc84d`. With a clean temporary home, `terrad version --long` reports version `4.1.0-34-g3d8f65ec` and embedded commit `3d8f65eccec3fe713b04e604a148319b7049d4e8`. The worktree has uncommitted source changes, so this metadata is not a unique candidate SHA.

## Incomplete checks and blockers

- An initial `make test-sim-after-import` run failed reproducibly at seed 1. The imported full-app simulation had pending randomized equivocation evidence; its continuation triggered a Cosmos SDK distribution panic while slashing (`calculated final stake ... greater than current stake`). The after-import test now exercises USTC operations before and after import, fixes its simulation-only account count at 100, suppresses unrelated randomized duplicate-vote evidence, promotes SDK skips to failures, and asserts that every requested block ran. Full-app and import/export simulations still cover their own operations and consensus randomness.
- Long and custom-genesis multi-seed simulations were not run; the implementation plan's four required Task 10 simulation checks (nondeterminism, import/export, after-import, and short multi-seed) now have passing evidence. Run the longer targets if they are required by the release owner.
- Swagger generation remains blocked. Neither `BUF_TOKEN` nor `BUF_API_TOKEN` is present in the environment; a Buf generation attempt reports an invalid/missing API token. A local, module-pinned Swagger plugin was built and generated the USTC query document with direct `protoc`, but full local generation stops at the missing `cosmos/ics23/v1/proofs.proto` include. Swagger YAML and embedded statik were not regenerated. This documentation issue is deferred for testnet as described above. Public Buf dependencies may be readable anonymously; inspect stale credentials before assuming a token is required.
- Seven-validator localnet campaign — not run. This worktree remains dirty and uncommitted, so there is no final candidate SHA. The campaign skill also requires explicit authorization before Docker stops, fresh genesis generation, governance proposals, or transactions.

The initial blocker assessment is superseded by the follow-up above. Testnet acceptance requires the clean candidate manifest, successful corrected validation, and completed seven-validator upgrade/lifecycle evidence; Swagger may remain deferred. The original `/home/sofoli/core` main worktree was left untouched; this evidence and implementation remain in the isolated feature worktree.
