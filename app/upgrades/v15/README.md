# v15 Upgrade — Packet Forward Middleware (PFM)

## Summary

Integrates [Packet Forward Middleware](https://github.com/cosmos/ibc-apps/tree/main/middleware/packet-forward-middleware)
(`v10.6.0`) into the IBC transfer stack, giving Terra Classic **multi-hop IBC
routing**: a single `MsgTransfer` can route atomically through Terra Classic to
a third chain (e.g. `chain-A → Terra Classic → chain-B`), with automatic
refund/timeout handling.

## What changes

- New module **`packetfowardmiddleware`** (note: the module name is spelled this
  way upstream and is frozen for state compatibility) — adds one KV store, one
  module account (intermediate receiver for in-flight forwarded packets), and a
  params subspace.
- The transfer stack is re-composed as:
  `channel → ibc-hooks → packet-forward → transfer`
  (PFM sits between ibc-hooks and the base transfer module, the same ordering
  used by Osmosis, Stride, et al.)
- Default params: **`fee_percentage = 0`** — forwarding is free. The fee can be
  raised later via the module's `MsgUpdateParams` to skim a share of forwarded
  packets to the community pool, should governance choose to.

### Dependency changes

| Module | Before | After |
|--------|--------|-------|
| `github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10` | — | `v10.6.0` (new) |
| `github.com/cosmos/ibc-go/v10` | `v10.5.0` | `v10.6.0` (transitive minor bump pulled by PFM) |

> The ibc-go `v10.5.0 → v10.6.0` bump is a transitive consequence of PFM and is
> a minor patch release; it is called out explicitly so reviewers are aware the
> upgrade is not *only* PFM.

## Upgrade handler

`CreateV15UpgradeHandler` performs no bespoke state migration. Because
`packetfowardmiddleware` is a newly registered module (absent from the incoming
`module.VersionMap`), `RunMigrations` runs its `InitGenesis` with
`DefaultGenesis`, which sets the default params (`fee_percentage = 0`). The store
is mounted via `StoreUpgrades.Added`.

## Testing

- `go build ./...` — clean.
- `terrad init` — module registers and appears in default genesis.
- **`tests/interchaintest/ibc_pfm_terra_hop_test.go::TestTerraAsPFMHop`** — a live
  3-chain test (`gaia → Terra(this build) → osmosis`) with a relayer. It asserts
  that Terra **itself** emits the onward `send_packet` after receiving from gaia,
  and that osmosis credits the forwarded funds. **Passes end-to-end.** This is
  the coverage the existing PFM tests lacked (they only exercised Osmosis as the
  hop, with Terra as source/destination).

## Risk

Low. This is a consensus-breaking state migration (it adds a module store), so it
must be applied at a governance-set height via Cosmovisor/coordinated binary swap
— the standard upgrade drill. It makes **no parameter changes to existing
modules**, and PFM is battle-tested across the Cosmos ecosystem. Existing IBC
(classic channel v1) behavior is unchanged for non-forwarded packets.
