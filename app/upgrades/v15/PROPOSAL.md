# Governance Proposal — v15: Packet Forward Middleware

> Draft for the on-chain `MsgSoftwareUpgrade` proposal and the accompanying
> forum post. Replace `<UPGRADE_HEIGHT>` and the binary `info`/checksum before
> submission.

## Title

Software Upgrade v15: Packet Forward Middleware (multi-hop IBC routing)

## Rationale (forum post)

Terra Classic speaks IBC, but today it can only move assets across channels it
holds **directly**. Reaching a chain it has no direct channel with requires
multiple manual transactions — or isn't possible in a single flow at all.

This upgrade adds **Packet Forward Middleware (PFM)**, the same multi-hop routing
component run by Osmosis, Stride, Injective, Sei and most major Cosmos chains. It
lets a single transfer route *through* Terra Classic to a third chain
automatically, with proper refund and timeout handling.

**What it gives the chain:**

- **Reach.** LUNC and other assets can move to the entire IBC network in one hop
  (e.g. via Osmosis or the Cosmos Hub) without Terra Classic opening a channel to
  every counterparty.
- **It strengthens the interoperability roadmap.** As assets are bridged *onto*
  Terra Classic (e.g. via Hyperlane), PFM ensures they are not stranded — they
  can flow onward across IBC. PFM makes the broader interop strategy more
  valuable.
- **Composability.** PFM sits alongside the existing IBC-hooks module, so a
  routed packet can also trigger a CosmWasm contract — a building block for
  cross-chain flows that touch Terra Classic contracts.
- **Optional community-pool revenue.** PFM supports a `fee_percentage` parameter
  that skims a share of *forwarded* packets to the community pool. This upgrade
  ships it at **0** (free forwarding); governance can raise it later via a simple
  params change if desired.

This is standard, low-risk infrastructure that closes a real capability gap and
modernizes the chain's IBC stack.

## Technical summary

- Integrates `packet-forward-middleware/v10 v10.6.0` into the transfer stack
  (`channel → ibc-hooks → packet-forward → transfer`).
- Adds the `packetfowardmiddleware` module (store, module account, params).
- Transitively bumps `ibc-go` `v10.5.0 → v10.6.0` (minor patch).
- Default `fee_percentage = 0`. No changes to any existing module's parameters.

## Testing

- Builds clean (`go build ./...`); the upgrade binary runs and registers the
  module in genesis.
- A live 3-chain interchaintest (`gaia → Terra → osmosis`) confirms Terra Classic
  forwards a multi-hop transfer end-to-end (`TestTerraAsPFMHop`, passing).

## Risk

Low. Consensus-breaking only in that it adds a module store; applied at the
governance-set height via the standard Cosmovisor / coordinated-swap drill. No
existing module parameters change. Non-forwarded IBC transfers are unaffected.
Validators must run the v15 release binary at the upgrade height (RAM headroom
per the standard upgrade checklist).

## Upgrade parameters

| Field | Value |
|-------|-------|
| Upgrade name | `v15` (must match the handler exactly) |
| Upgrade height | `<UPGRADE_HEIGHT>` (set ~5–7 days after expected pass) |
| Deposit | 5,000,000 LUNC (`5000000000000uluna`) |
| Voting period | 7 days |
| Quorum / threshold / veto | 40% / 50% / 33.4% |

## Submitting the proposal (runbook)

`MsgSoftwareUpgrade` is gov-authority gated. Submit via `gov v1`:

```bash
# Derive the gov module authority address from the binary (verify it matches):
#   expected: terra10d07y265gmmuvt4z0w9aw880jnsr700juxf95n
```

`proposal.json`:

```json
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "terra10d07y265gmmuvt4z0w9aw880jnsr700juxf95n",
      "plan": {
        "name": "v15",
        "height": "<UPGRADE_HEIGHT>",
        "info": "{\"binaries\":{\"linux/amd64\":\"<RELEASE_URL>?checksum=sha256:<SHA256>\"}}"
      }
    }
  ],
  "metadata": "ipfs://<CID>",
  "deposit": "5000000000000uluna",
  "title": "Software Upgrade v15: Packet Forward Middleware",
  "summary": "Add Packet Forward Middleware for multi-hop IBC routing. See README."
}
```

```bash
terrad tx gov submit-proposal proposal.json \
  --from <key> --chain-id columbus-5 \
  --gas auto --gas-adjustment 1.4 --fees 30000000uluna
```

The `info.binaries` entry lets Cosmovisor auto-stage the binary, but
**`DAEMON_ALLOW_DOWNLOAD_BINARIES` should remain `false`** — validators stage the
release binary by hand and verify the sha256, exactly as in prior upgrades.

> Prerequisite: this code must be merged into the canonical `classic-terra/core`
> release that validators run; otherwise a passing vote halts the chain at the
> upgrade height with no matching handler. The PR + L1JTF review is upstream of
> this proposal.
