Generally we should avoid shell scripting and write tests purely in Golang.
However, some libraries are not Goroutine-safe (e.g. app simulations cannot be run safely in parallel),
and OS-native threading may be more efficient for many parallel simulations, so we use shell scripts here.

## Upgrade Test Scripts

### `run-node.sh`

Initializes and starts a single-validator `terrad` node for local testing.

**Usage:** `bash scripts/run-node.sh <binary> <denom>`

**Arguments:**
- `$1` — path to the `terrad` binary (e.g. `_build/old/terrad`)
- `$2` — staking/gas denom (default: `uluna`)

**Environment variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `CONTINUE` | `false` | Set to `true` to skip init and just restart an existing node (used after upgrades) |
| `TERRAD_HALT_HEIGHT` | — | If set, passes `--halt-height` to terrad (used in fork-mode upgrades) |
| `SELF_DELEGATION` | `900000000000` | Validator self-delegation amount |

The script creates three test accounts (`test0`, `test1`, `test2`), each funded with `1000000000000uluna`, configures 30s voting period and 500ms block commit timeout.

---

### `upgrade-test.sh`

Tests a **single** upgrade from an old binary to the current codebase.

**Usage:** `bash scripts/upgrade-test.sh [--reinstall-old]`

**Environment variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `OLD_VERSION` | `v4.0.0-rc.6` | Git tag of the old binary to download and build |
| `SOFTWARE_UPGRADE_NAME` | `v14_1` | Name of the upgrade handler in the current code |
| `FORK` | `false` | Set to `true` for fork-style upgrade (halt at height, no governance) |
| `FORK_HALT_HEIGHT` | — | Custom halt height for fork mode |
| `ADDITIONAL_PRE_SCRIPTS` | — | Comma-separated scripts to run before upgrade |
| `ADDITIONAL_AFTER_SCRIPTS` | — | Comma-separated scripts to run after upgrade |
| `GAS_PRICE` | `30uluna` | Gas price for transactions |

**Flow:**
1. Downloads and builds old binary → `_build/old/terrad`
2. Builds current code → `_build/new/terrad`
3. Starts old node via `run-node.sh`
4. Runs any pre-upgrade scripts
5. Submits governance upgrade proposal (or halts at height if fork mode)
6. Waits for upgrade height, kills old node
7. Starts new node with `CONTINUE=true`
8. Runs any post-upgrade scripts

**Example — test v14_1 upgrade from rc6:**
```bash
bash scripts/upgrade-test.sh
```

**Example — test with custom old version:**
```bash
OLD_VERSION=v3.6.0-rc.0 SOFTWARE_UPGRADE_NAME=v14 bash scripts/upgrade-test.sh
```

---

### `upgrade-test-multi.sh`

Tests a **chain of sequential upgrades**, simulating the full upgrade history from an old version to the current codebase through intermediate releases.

**Usage:** `bash scripts/upgrade-test-multi.sh [--reinstall]`

**Environment variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `OLD_VERSIONS` | `v3.6.0-rc.0,v4.0.0-rc.6` | Comma-separated git tags for each stage |
| `UPGRADE_NAMES` | `v14,v14_1` | Comma-separated upgrade handler names (must match `OLD_VERSIONS` length) |
| `ADDITIONAL_PRE_SCRIPTS` | — | Comma-separated scripts to run before first upgrade |
| `ADDITIONAL_AFTER_SCRIPTS` | — | Comma-separated scripts to run after all upgrades |
| `CW20_TOKEN_WASM` | `./scripts/cw20_token.wasm` | Path to CW20 wasm file for contract state tests |
| `GAS_PRICE` | `30uluna` | Gas price for transactions |
| `FORK` | `false` | Set to `true` for fork-style upgrade |

`OLD_VERSIONS` and `UPGRADE_NAMES` must have the same length. `OLD_VERSIONS[i]` is the binary running before upgrade `UPGRADE_NAMES[i]` is applied. The **last** upgrade always transitions to the current codebase (`_build/new/`).

**Default upgrade chain (v13 → v14 → v14_1):**

| Step | Binary | SDK | Upgrade Applied |
|------|--------|-----|-----------------|
| Start | `v3.6.0-rc.0` (v13 era) | v0.47.17 | — |
| 1st upgrade | `v4.0.0-rc.6` | v0.53.x | `v14` (handler in rc6 binary) |
| 2nd upgrade | current code | v0.53.6 | `v14_1` (handler in current code) |

**Flow:**
1. Builds all version binaries: first → `_build/old/`, intermediates → `_build/v1/`, `_build/v2/`, …, current → `_build/new/`
2. Starts first node, runs pre-scripts
3. Deploys a CW20 token contract for state migration testing
4. Iterates through upgrades via governance proposals (incrementing proposal IDs)
5. After first upgrade, executes a CW20 transfer and runs intermediate tests
6. After all upgrades, runs final tests (staking params, delegations, wasm queries at current and historic heights)

**Example — full upgrade chain with defaults:**
```bash
bash scripts/upgrade-test-multi.sh
```

**Example — custom 3-step chain:**
```bash
OLD_VERSIONS="v3.6.0-rc.0,v4.0.0-rc.5,v4.0.0-rc.6" \
UPGRADE_NAMES="v14,v14rc4,v14_1" \
bash scripts/upgrade-test-multi.sh
```
