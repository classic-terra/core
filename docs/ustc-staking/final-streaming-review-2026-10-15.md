# USTC streaming rewards: final implementation review

Review date: October 15, 2026. Reviewed source: `dfb176f8552ae40e6799d978b3dbc683e6a1b1f1`.

## Assessment

The core implementation follows the [approved design](../superpowers/specs/2026-10-13-ustc-staking-streaming-rewards-design.md) and [implementation plan](../superpowers/plans/2026-10-13-ustc-staking-streaming-rewards.md) well. No new blocking USTC logic defect was found in the paths reviewed. Evidence supports a governed testnet rehearsal; this is not a mainnet security certification or proof of public deployment.

The review compared the plan with source, inspected prior simulation and localnet artifacts, and freshly ran affected tests. It did not repeat the full simulation corpus or seven-validator campaign. Graphify at production-source commit `34d0703` was used for navigation; current source was authoritative. The later `dfb176f8` commit contains simulation and documentation changes only.

## Plan coverage

| Plan area | Source and assessment |
|---|---|
| Tasks 1–3: fixtures, schema, streaming arithmetic | Deterministic fixtures, initialized reserve fields, integer elapsed-time calculation, emission/index carries, arithmetic admission guards, and constant work per accrual are present. |
| Tasks 4–5: funding and lifecycle | Funding enters unallocated reserve; old intervals settle before top-ups and parameter/share changes. Active claims preserve fractions. Unbonding ends earning immediately; withdrawal remains explicit. |
| Task 6: custody and invariants | Principal and rewards remain separate. Claims and dust return protect unallocated funding and earned liabilities. Exact fractional liabilities are checked. |
| Task 7: blocks and queries | BeginBlock accrual is wired. Queries use stored consensus state; runway and allocated/unallocated balances are exposed. CLI validation aggregates pinned pages and returns errors for invalid accounting. |
| Task 8: migration | Absent-store activation and existing-store migration have distinct upgrade registrations. Legacy rewards remain allocated/protected rather than becoming new stream funding. |
| Task 9: accounting and adversarial checks | Real bank/distribution integration, failed-operation rollback, supply conservation, independent lifecycle/reference tests, and accrual benchmarks have evidence. |
| Task 10: simulations | 35 import/export and 33 full after-import seed logs independently checked for PASS. Two after-import terminal fixtures are separately verified. Prior short-corpus and determinism evidence is recorded in validation notes. |
| Tasks 11–12: documentation and handoff | Runtime API artifacts, module specs, operator guidance, and validation evidence exist. Swagger is an explicit testnet deferral. Missing linked documents and outdated stakeholder status were identified and corrected in the accompanying documentation commit. |

Principal source paths: `x/ustcstaking/keeper/{streaming,rewards,msg_server,invariants,query_server,migrations}.go`, `x/ustcstaking/types/params.go`, `x/ustcstaking/module.go`, `x/ustcstaking/client/cli/query.go`, and `app/upgrades/ustc_staking/`.

The USTC keeper interfaces expose no mint or LUNC-staking capability. Both pool module accounts have no mint/burn permissions. The default is a single 21-day, 1× tier and zero emission rate. Governance funding alone does not start emission.

## Fresh verification

Both commands completed with exit code 0 on the reviewed tree:

```sh
go test -mod=readonly ./x/ustcstaking/... ./app/keepers ./app/upgrades/ustc_staking -count=1
go test -mod=readonly ./app -run 'Test.*(USTC|Streaming)' -count=1
```

Build-host logs: `/mnt/d/ustc-streaming-release-campaign/plan-double-check.log` and `plan-double-check-app.log`. These are local evidence paths, not files distributed with the repository.

The review independently inspected all 35 PASS tails in `sim-final/verified-import-export/` and all 33 PASS tails in `sim-final/verified-after-import-normalized/`, beneath that evidence root. Seeds 124 and 37827 are separate terminal tests: each ends with an empty validator set at committed height 19 and does not count as 50-block after-import coverage.

The after-import harness adjusts exported LUNC unbonding creation heights from zero to one to satisfy the SDK cancellation simulator. It therefore tests a normalized simulation fixture, not an untouched zero-height export. The ordinary import/export comparison remains separate. This qualification must accompany coverage claims.

Localnet final artifacts under `/home/sofoli/lbunproject/ustc-campaign-evidence/streaming-20261014/` include `final-lifecycle.json` (height 1285), `single-restart.json` (1290), `network-restart.json` (1297), and `final-seven-status.json` (all seven at height 1301, none catching up). The three accounting snapshots report valid state. Earlier campaign logs contain superseded harness failures; use the completed snapshots and campaign assertions rather than treating every historical log as successful.

## Handoff corrections and remaining limits

- Include the approved streaming spec/plan and updated C-suite/technical briefings in Git, with working relative links. Older design documents referenced by the historical implementation instructions remain local background; the streaming design is the published behavior contract.
- Update the executive summary to reflect completed validation and no Phase 3.
- Preserve the Swagger deferral: keep Swagger UI disabled until generated Swagger/statik are refreshed and reviewed. CLI, REST, and gRPC campaign checks are separate runtime evidence.
- Longer/custom-genesis simulation variants were not run. Test coverage is substantial for a controlled testnet rehearsal, not an exhaustive proof of all future chain states.
- Public activation still needs a verified predecessor/module-version map, an unused upgrade plan, an agreed height, governance approval, and validator coordination. No public deployment is recorded here.

The packaged production binary remains built from `34d0703758cdf1a094318fadbee34ea18a7d6f55`; subsequent test/documentation commits do not change production Go sources. Binary SHA-256: `e35a9c0d9db03c1e8e18f4f0e14fa25a3bd10fd5a0f44048c6578a45299714d0`.

See [validation evidence](phase-2-streaming-rewards-validation.md), [operator handoff](streaming-testnet-deployment.md), and [30M USTC budget analysis](30m-ustc-two-year-rewards-analysis.md).
