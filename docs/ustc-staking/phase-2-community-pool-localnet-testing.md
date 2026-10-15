# USTC Staking Phase 2 Community-Pool Validator Campaign (superseded)

> **Do not execute the legacy expectations in this document for streaming rewards.** Its instructions assume funding immediately updates the reward index, funding while paused or without active shares fails, and fresh tiers are empty. Those behaviors are obsolete. Use the current [streaming-rewards localnet campaign](phase-2-streaming-rewards-localnet-testing.md), which preserves the seven-validator, restart, supply, and LUNC-isolation checks with current economics.

This runbook is for independent validators testing governance-authorized
community-pool reward funding for native USTC staking on a disposable,
seven-validator local network. It exercises the staking lifecycle needed to
verify Phase 2 funding and accounting, including governance execution, failure
cases, pause/resume, and process recovery. The old-state-to-new-binary upgrade
rehearsal is a separate release gate, not part of this campaign.

This campaign does not change LUNC staking or validator consensus state.
Passing it is functional test evidence, not approval to activate on a public
network.

## Campaign overview: governance community-pool funding

Users continue to stake USTC directly from their own accounts with native
`MsgStake` transactions, without sending principal through a contract. Phase 2
changes reward funding: a governance proposal executes `MsgFundRewards`, and
the application moves the approved existing `uusd` amount from the distribution
community pool into `ustcstaking_reward_pool`. Only after that transfer
succeeds does the staking module update its reward index. The operation must
not mint USTC, debit user principal, or affect LUNC validator staking.

The procedures below test that end-to-end governance flow, exact pool and
supply reconciliation, expected failure behavior, claims, pause/resume,
unbonding and withdrawal, and recovery after validator and network restarts.
Funding is not a wallet-signed user transaction: compose it in a governance
proposal and derive the governance authority from the running local chain as
the steps specify.

## Candidate and evidence rules

Before testing, the release coordinator publishes the exact 40-character
candidate commit SHA and the candidate build or its reproducible build recipe.
Replace the placeholder below with that published SHA. Do not choose a branch
name, update to a moving branch, or silently substitute a later commit.

```bash
export CANDIDATE_SHA='<published-40-character-commit-sha>'
test "${#CANDIDATE_SHA}" -eq 40
git rev-parse HEAD
test "$(git rev-parse HEAD)" = "$CANDIDATE_SHA"
git status --short
```

The tree must be clean. If the published artifact is supplied as a binary,
record its publisher-provided SHA-256 and verify it before use. If validators
build locally, use the exact SHA and record the toolchain and binary hash as
shown below. Any source or build difference starts a new campaign record.

Use a fresh clone or disposable checkout. The localnet stop target deletes
generated `build/node*` homes and `build/gentxs`; never point it at a valued
node home or a shared environment.

## Campaign settings and expected values

Use seven validators on chain ID `localterra`. Set commit timeout to `2s`,
governance voting and max-deposit periods to `5m`, and expedited voting to
`2m`. The longer periods allow validators to submit and inspect votes on slower
machines. Wait for proposal status and position maturity; do not use fixed
short sleeps as proof that voting or unbonding has completed.

Seed each of the seven validator accounts with `1,000,000 USTC` (`1000000000000uusd`).
Governance will configure tier 1 at `300s / 1x` and tier 2 at `600s / 2x`.
Use two user accounts for the weighted reward scenario:

| Action | Exact amount |
|---|---:|
| User 0 stake, tier 1 | `100000000uusd` principal; `100000000` shares |
| User 1 stake, tier 2 | `100000000uusd` principal; `200000000` shares |
| Authorized community-pool funding | `300000000uusd` |
| User 0 expected reward | `100000000uusd` |
| User 1 expected reward | `200000000uusd` |

For every comparison, use integer strings for bank amounts and shares. The
distribution community pool is decimal-valued; reconcile its `uusd` delta with
decimal arithmetic, never binary floating point. Transactions pay fees in
`stake`, so the `uusd` balance comparisons below are not affected by fees.

## 1. Record the test environment and build

Run from the candidate repository root. Keep test output and evidence outside
the repository, for example in `/tmp/ustc-campaign-$CANDIDATE_SHA`.

```bash
export EVIDENCE_DIR="/tmp/ustc-campaign-$CANDIDATE_SHA"
mkdir -p "$EVIDENCE_DIR"
git rev-parse HEAD | tee "$EVIDENCE_DIR/candidate-sha.txt"
git status --short | tee "$EVIDENCE_DIR/git-status.txt"
go version | tee "$EVIDENCE_DIR/go-version.txt"
go env GOOS GOARCH GOVERSION | tee "$EVIDENCE_DIR/go-env.txt"
docker version > "$EVIDENCE_DIR/docker-version.txt"
docker compose version > "$EVIDENCE_DIR/docker-compose-version.txt"
jq --version | tee "$EVIDENCE_DIR/jq-version.txt"
curl --version | head -n 1 | tee "$EVIDENCE_DIR/curl-version.txt"
uname -a | tee "$EVIDENCE_DIR/uname.txt"
```

Record host OS/architecture, CPU and memory, Docker version, Compose version,
Go version, jq version, candidate SHA, build command or artifact source, and
whether the image was rebuilt or reused. The localnet image is built from
`contrib/localnet/terrad-env`; record its image ID:

```bash
docker image inspect classic-terra/terrad-env \
  --format '{{.Id}} {{.Os}}/{{.Architecture}}' \
  | tee "$EVIDENCE_DIR/localnet-image.txt"
```

If no Linux AMD64 image exists, build it with the repository target. Build the
candidate binary from the pinned checkout and record its hash/version:

```bash
docker image inspect classic-terra/terrad-env >/dev/null 2>&1 \
  || make -C contrib/localnet terrad-env
make build-linux
sha256sum build/terrad | tee "$EVIDENCE_DIR/terrad-sha256.txt"
file build/terrad | tee "$EVIDENCE_DIR/terrad-file.txt"
./build/terrad version --long --home /tmp/ustc-version-check \
  | tee "$EVIDENCE_DIR/terrad-version.txt"
```

Expected: Linux AMD64 `terrad`, version metadata identifies the candidate
revision, and the source SHA still equals `$CANDIDATE_SHA`.

## 2. Generate and configure a fresh seven-validator network

Run only after confirming the worktree is disposable. The commands mirror the
repository's localnet generator and pause before Compose starts the nodes:

```bash
make localnet-stop
docker run --platform linux/amd64 --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD/build:/terrad:Z" \
  -v /etc/group:/etc/group:ro \
  -v /etc/passwd:/etc/passwd:ro \
  -v /etc/shadow:/etc/shadow:ro \
  classic-terra/terrad-env \
  testnet --chain-id localterra --v 7 -o . \
  --starting-ip-address 192.168.10.2 --keyring-backend test
```

Check that all seven generated homes exist:

```bash
for i in {0..6}; do
  test -f "build/node$i/terrad/config/genesis.json"
  test -f "build/node$i/terrad/config/config.toml"
done
```

Add `1,000,000 USTC` to each generated validator balance and `7,000,000 USTC`
to total supply in canonical node 0 genesis. Set governance timing in the
actual generated genesis, then copy it byte-for-byte to the other six nodes:

```bash
GENESIS="$PWD/build/node0/terrad/config/genesis.json"
jq '
  (.app_state.bank.balances[].coins) += [
    {"denom":"uusd","amount":"1000000000000"}
  ]
  | (.app_state.bank.balances[].coins) |= sort_by(.denom)
  | .app_state.bank.supply += [
      {"denom":"uusd","amount":"7000000000000"}
    ]
  | .app_state.bank.supply |= sort_by(.denom)
  | .app_state.gov.params.voting_period = "300s"
  | .app_state.gov.params.expedited_voting_period = "120s"
  | .app_state.gov.params.max_deposit_period = "300s"
' "$GENESIS" > "$GENESIS.tmp"
mv "$GENESIS.tmp" "$GENESIS"
for i in {1..6}; do
  cp "$GENESIS" "$PWD/build/node$i/terrad/config/genesis.json"
  cmp "$GENESIS" "$PWD/build/node$i/terrad/config/genesis.json"
done
jq '.app_state.bank.supply[] | select(.denom == "uusd")' "$GENESIS"
jq '.app_state.gov.params | {voting_period, expedited_voting_period, max_deposit_period}' "$GENESIS"
jq '.app_state.ustcstaking' "$GENESIS"
```

Expected genesis has `7000000000000uusd`, seven balances of
`1000000000000uusd`, empty lock tiers, zero reward index/shares, and the
timing settings above. If this candidate's generated genesis uses different
JSON paths, stop and report the candidate SHA and observed schema; do not
guess or edit a different field. The standalone `validate-genesis` command is
not a campaign gate for this candidate: it currently panics in the genutil
message-validator callback. Successful InitChain is the available full-genesis
check.

Set commit timeouts to `2s` in all seven Tendermint configs, then verify every
file:

```bash
for config_file in build/node*/terrad/config/config.toml; do
  sed -i 's/^timeout_commit = ".*"/timeout_commit = "2s"/' "$config_file"
done
rg '^timeout_commit' build/node*/terrad/config/config.toml
```

Start the network and save its initial status:

```bash
docker compose up -d
docker compose ps -a | tee "$EVIDENCE_DIR/compose-initial.txt"
until curl -fsS http://localhost:26657/status \
  | jq -e '.result.sync_info.catching_up == false' >/dev/null; do sleep 2; done
curl -fsS http://localhost:26657/status | tee "$EVIDENCE_DIR/status-initial.json"
```

All seven containers must be running and node 0 must advance blocks. Record
the initial logs if startup is delayed:

```bash
docker compose logs --tail=200 > "$EVIDENCE_DIR/logs-initial.txt"
```

## 3. Set CLI context and strict transaction helpers

The commands below use CLI names registered by this candidate's USTC module and
Cosmos SDK v0.53.6. Each account signs from its own generated test keyring.

```bash
export TERRAD="$PWD/build/terrad"
export CHAIN_ID=localterra
export RPC=tcp://localhost:26657
export KEYRING_BACKEND=test
export NODE0_HOME="$PWD/build/node0/terrad"
export TX_FEE=1000000stake
set -euo pipefail

QUERY_FLAGS=(--chain-id "$CHAIN_ID" --node "$RPC" --home "$NODE0_HOME" --output json)
TX_FLAGS=(--chain-id "$CHAIN_ID" --node "$RPC" --keyring-backend "$KEYRING_BACKEND" \
  --gas auto --gas-adjustment 1.5 --fees "$TX_FEE" --broadcast-mode sync --yes --output json)

query_tx() { "$TERRAD" query tx "$1" "${QUERY_FLAGS[@]}"; }

wait_for_tx() {
  local hash="$1" result
  for _ in $(seq 1 60); do
    if result=$(query_tx "$hash" 2>/dev/null); then
      printf '%s\n' "$result"
      return 0
    fi
    sleep 1
  done
  echo "transaction $hash was not indexed within 60 seconds" >&2
  return 1
}

expect_ok() {
  local output rc code hash result
  if output=$("$@" "${TX_FLAGS[@]}" 2>&1); then rc=0; else rc=$?; fi
  if ! jq -e . >/dev/null 2>&1 <<<"$output"; then
    printf 'CLI did not return JSON (exit %s):\n%s\n' "$rc" "$output" >&2
    return 1
  fi
  code=$(jq -r '.code // .tx_response.code // empty' <<<"$output")
  if [ "$rc" -ne 0 ] || [ "$code" != 0 ]; then
    printf 'expected success, got exit=%s code=%s:\n%s\n' "$rc" "$code" "$output" >&2
    return 1
  fi
  hash=$(jq -r '.txhash // .tx_response.txhash // empty' <<<"$output")
  test -n "$hash" || { echo 'successful broadcast had no tx hash' >&2; return 1; }
  result=$(wait_for_tx "$hash") || return 1
  code=$(jq -r '.tx_response.code // .code // empty' <<<"$result")
  if [ "$code" != 0 ]; then
    printf 'DeliverTx failed code=%s:\n%s\n' "$code" "$result" >&2
    return 1
  fi
  printf '%s\n' "$result"
}

expect_chain_error() {
  local expected_space="$1" expected_code="$2"; shift 2
  local output rc code space hash result
  if output=$("$@" "${TX_FLAGS[@]}" 2>&1); then rc=0; else rc=$?; fi
  if ! jq -e . >/dev/null 2>&1 <<<"$output"; then
    printf 'NOT A CHAIN ERROR: CLI/encoding/network failure (exit %s):\n%s\n' "$rc" "$output" >&2
    return 1
  fi
  code=$(jq -r '.code // .tx_response.code // empty' <<<"$output")
  space=$(jq -r '.codespace // .tx_response.codespace // empty' <<<"$output")
  hash=$(jq -r '.txhash // .tx_response.txhash // empty' <<<"$output")
  if [ "$code" = 0 ] && [ -n "$hash" ]; then
    result=$(wait_for_tx "$hash") || return 1
    code=$(jq -r '.tx_response.code // .code // empty' <<<"$result")
    space=$(jq -r '.tx_response.codespace // .codespace // empty' <<<"$result")
  fi
  if [ "$code" != "$expected_code" ] || [ "$space" != "$expected_space" ]; then
    printf 'wrong rejection: expected %s/%s, got %s/%s (exit %s):\n%s\n' \
      "$expected_space" "$expected_code" "$space" "$code" "$rc" "$output" >&2
    return 1
  fi
  printf '%s\n' "$output" | jq '{code, codespace, raw_log, txhash}'
}

latest_proposal_id() {
  "$TERRAD" query gov proposals "${QUERY_FLAGS[@]}" \
    | jq -er '[.proposals[].id | tonumber] | max'
}

vote_all_yes() {
  local proposal_id="$1" i
  for i in {0..6}; do
    expect_ok "$TERRAD" tx gov vote "$proposal_id" yes \
      --from "node$i" --home "$PWD/build/node$i/terrad" >/dev/null
  done
}

wait_for_proposal_terminal() {
  local proposal_id="$1" result status
  for _ in $(seq 1 900); do
    result=$("$TERRAD" query gov proposal "$proposal_id" "${QUERY_FLAGS[@]}") || return 1
    status=$(jq -r '.proposal.status' <<<"$result")
    case "$status" in
      PROPOSAL_STATUS_PASSED|PROPOSAL_STATUS_REJECTED|PROPOSAL_STATUS_FAILED)
        printf '%s\n' "$result"
        return 0 ;;
    esac
    sleep 1
  done
  echo "proposal $proposal_id did not reach terminal status within 15 minutes" >&2
  return 1
}

uusd_balance() {
  "$TERRAD" query bank balance "$1" uusd "${QUERY_FLAGS[@]}" | jq -er '.balance.amount'
}

module_address() {
  "$TERRAD" query auth module-account "$1" "${QUERY_FLAGS[@]}" \
    | jq -er '.account.value.address // .account.base_account.address // .account.base_vesting_account.base_account.address // .account.address'
}
```

`expect_chain_error` only accepts a parsed chain response with the expected
codespace and numeric code. A shell error, malformed JSON, timeout, missing
transaction, or unrelated code is a failed test, never a successful negative
case. For governance-executed messages, record proposal terminal status and
the execution failure reason; do not label a CLI proposal-submission error as
a keeper rejection.

## 4. Baseline: accounts, supply, pools, rewards, and validators

Resolve all generated account addresses and the two module accounts:

```bash
for i in {0..6}; do
  NODE_ADDR[$i]=$("$TERRAD" keys show "node$i" -a \
    --keyring-backend test --home "$PWD/build/node$i/terrad")
  printf 'node%s %s %s uusd\n' "$i" "${NODE_ADDR[$i]}" "$(uusd_balance "${NODE_ADDR[$i]}")"
done
export NODE0_ADDR="${NODE_ADDR[0]}" NODE1_ADDR="${NODE_ADDR[1]}" NODE2_ADDR="${NODE_ADDR[2]}"
export PRINCIPAL_POOL="$(module_address ustcstaking)"
export REWARD_POOL="$(module_address ustcstaking_reward_pool)"
export GOV_AUTH=$("$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" | jq -er '.params.authority')
GOV_MODULE_ADDR=$(module_address gov)
test "$GOV_AUTH" = "$GOV_MODULE_ADDR"
```

Each validator account must have `1000000000000uusd`; `GOV_AUTH` must match
the governance module account. Save raw query results at the baseline and after
each campaign phase. These are the verified queries for the supplies and
accounts involved:

```bash
snapshot() {
  local label="$1" dir
  dir="$EVIDENCE_DIR/$label"
  mkdir -p "$dir"
  "$TERRAD" query bank total-supply-of uusd "${QUERY_FLAGS[@]}" > "$dir/supply-uusd.json"
  "$TERRAD" query bank total-supply-of stake "${QUERY_FLAGS[@]}" > "$dir/supply-stake.json"
  "$TERRAD" query bank balances "$NODE0_ADDR" "${QUERY_FLAGS[@]}" > "$dir/node0-balances.json"
  "$TERRAD" query bank balances "$NODE1_ADDR" "${QUERY_FLAGS[@]}" > "$dir/node1-balances.json"
  for i in {2..6}; do
    "$TERRAD" query bank balances "${NODE_ADDR[$i]}" "${QUERY_FLAGS[@]}" > "$dir/node$i-balances.json"
  done
  for i in {0..6}; do
    "$TERRAD" query ustcstaking positions "${NODE_ADDR[$i]}" "${QUERY_FLAGS[@]}" > "$dir/node$i-positions.json"
  done
  "$TERRAD" query bank balance "$PRINCIPAL_POOL" uusd "${QUERY_FLAGS[@]}" > "$dir/principal-pool.json"
  "$TERRAD" query bank balance "$REWARD_POOL" uusd "${QUERY_FLAGS[@]}" > "$dir/reward-pool.json"
  "$TERRAD" query bank balance "$(module_address distribution)" uusd "${QUERY_FLAGS[@]}" > "$dir/distribution-account.json"
  "$TERRAD" query distribution community-pool "${QUERY_FLAGS[@]}" > "$dir/community-pool.json"
  "$TERRAD" query ustcstaking reward-state "${QUERY_FLAGS[@]}" > "$dir/reward-state.json"
  "$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" > "$dir/params.json"
  # validate-state fetches bounded pages (500 positions per RPC) at one pinned height.
  "$TERRAD" query ustcstaking validate-state "${QUERY_FLAGS[@]}" > "$dir/accounting-validation.json"
  jq -e '.valid == true' "$dir/accounting-validation.json" >/dev/null || {
    cat "$dir/accounting-validation.json" >&2
    return 1
  }
  "$TERRAD" query staking validators "${QUERY_FLAGS[@]}" > "$dir/validators.json"
}

assert_ustc_unchanged() {
  local before="$EVIDENCE_DIR/$1" after="$EVIDENCE_DIR/$2" file i
  for file in supply-uusd.json principal-pool.json reward-pool.json \
    distribution-account.json reward-state.json params.json; do
    diff -u <(jq -S . "$before/$file") <(jq -S . "$after/$file")
  done
  diff -u <(jq -S '[.pool[]? | select(.denom == "uusd")]' "$before/community-pool.json") \
    <(jq -S '[.pool[]? | select(.denom == "uusd")]' "$after/community-pool.json")
  for i in {0..6}; do
    diff -u <(jq -S '[.balances[]? | select(.denom == "uusd")]' "$before/node$i-balances.json") \
      <(jq -S '[.balances[]? | select(.denom == "uusd")]' "$after/node$i-balances.json")
    diff -u <(jq -S . "$before/node$i-positions.json") <(jq -S . "$after/node$i-positions.json")
  done
}

snapshot baseline
```

At baseline, assert supply `7000000000000uusd`, principal pool zero, reward
pool zero, reward index zero, total shares zero, and empty lock tiers. Preserve
the baseline JSON and record the `uusd` amounts for all seven user accounts,
distribution module account, and community pool.

Capture protected validator state. Canonically sort by operator address and
retain consensus key, status, jailed flag, tokens, delegator shares,
commission, and min-self-delegation:

```bash
jq '[.validators[] | {
  operator_address, consensus_pubkey, status, jailed, tokens, delegator_shares,
  commission, min_self_delegation
}] | sort_by(.operator_address)' \
  "$EVIDENCE_DIR/baseline/validators.json" \
  > "$EVIDENCE_DIR/validators-protected-before.json"
"$TERRAD" query slashing signing-infos "${QUERY_FLAGS[@]}" \
  > "$EVIDENCE_DIR/slashing-signing-info-before.json"
for i in {0..6}; do
  "$TERRAD" query staking delegations "${NODE_ADDR[$i]}" "${QUERY_FLAGS[@]}" \
    > "$EVIDENCE_DIR/delegations-node$i-before.json"
done
```

Also record module-account permissions. Both accounts must have empty
permissions (no minter or burner):

```bash
for name in ustcstaking ustcstaking_reward_pool; do
  "$TERRAD" query auth module-account "$name" "${QUERY_FLAGS[@]}" \
    | tee "$EVIDENCE_DIR/module-account-$name.json" \
    | jq '.account | {name, permissions}'
  test "$(jq -r '.account.value.permissions // .account.permissions // [] | length' \
    "$EVIDENCE_DIR/module-account-$name.json")" -eq 0
done
```

## 5. Governance proposal workflow and lock tiers

The current governance CLI uses `tx gov submit-proposal FILE`,
`tx gov vote ID yes`, `query gov proposals`, `query gov proposal ID`, and
`query gov votes ID`. A successful proposal submission is not proof of
execution. Capture submission response, proposal ID, votes, terminal status,
and post-execution params for every governance action.

Create a proposal with the full replacement params object. Durations below are
long enough for slow local hosts while keeping the campaign practical:

```bash
jq -n --arg authority "$GOV_AUTH" '
{
  messages: [{
    "@type": "/terra.ustcstaking.v1.MsgUpdateParams",
    authority: $authority,
    params: {
      bond_denom: "uusd",
      lock_tiers: [
        {id: 1, duration: "300s", multiplier: "1.000000000000000000"},
        {id: 2, duration: "600s", multiplier: "2.000000000000000000"}
      ],
      authority: $authority,
      paused: false
    }
  }],
  metadata: "",
  deposit: "10000000stake",
  title: "Configure USTC staking validator test tiers",
  summary: "Set two temporary lock tiers for the Phase 2 localnet validator campaign",
  expedited: false
}' > "$EVIDENCE_DIR/params-tiers-proposal.json"

expect_ok "$TERRAD" tx gov submit-proposal "$EVIDENCE_DIR/params-tiers-proposal.json" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/params-submit.json"
PARAMS_PROPOSAL_ID=$(latest_proposal_id)
vote_all_yes "$PARAMS_PROPOSAL_ID"
"$TERRAD" query gov votes "$PARAMS_PROPOSAL_ID" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/params-votes.json"
PARAMS_RESULT=$(wait_for_proposal_terminal "$PARAMS_PROPOSAL_ID")
printf '%s\n' "$PARAMS_RESULT" | tee "$EVIDENCE_DIR/params-result.json" | jq '.proposal | {id, status, title}'
test "$(jq -r '.proposal.status' <<<"$PARAMS_RESULT")" = PROPOSAL_STATUS_PASSED
"$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/params-applied.json"
```

Expected: tier durations and multipliers exactly match the proposal, authority
is unchanged, and paused is false. Snapshot as `tiers-applied`.

## 6. Direct staking, exact balances, and user failure cases

Capture a pre-action snapshot. Verify wrong denomination and unknown tier
return the expected module error and create no positions:

```bash
snapshot before-negative-stakes
expect_chain_error ustcstaking 1 "$TERRAD" tx ustcstaking stake 1000000stake 1 \
  --from node0 --home "$PWD/build/node0/terrad"
expect_chain_error ustcstaking 2 "$TERRAD" tx ustcstaking stake 1000000uusd 99 \
  --from node0 --home "$PWD/build/node0/terrad"
snapshot after-negative-stakes
assert_ustc_unchanged before-negative-stakes after-negative-stakes
```

Expected codes are `ustcstaking/1` (invalid denomination) and
`ustcstaking/2` (invalid lock tier). Compare all account and pool balances,
supply, reward state, and owner-position queries with the snapshot. No value or
state may change. The `positions [owner]` query is an owner-scoped query; its
response should have no positions for node 0 yet.

Run the direct native user path; no contract or governance action is involved:

```bash
NODE0_BEFORE_STAKE=$(uusd_balance "$NODE0_ADDR")
NODE1_BEFORE_STAKE=$(uusd_balance "$NODE1_ADDR")
SUPPLY_BEFORE_STAKES=$("$TERRAD" query bank total-supply-of uusd "${QUERY_FLAGS[@]}" | jq -er '.amount.amount')
expect_ok "$TERRAD" tx ustcstaking stake 100000000uusd 1 \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/node0-stake.json"
expect_ok "$TERRAD" tx ustcstaking stake 100000000uusd 2 \
  --from node1 --home "$PWD/build/node1/terrad" | tee "$EVIDENCE_DIR/node1-stake.json"
"$TERRAD" query ustcstaking positions "$NODE0_ADDR" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/node0-positions-active.json"
"$TERRAD" query ustcstaking positions "$NODE1_ADDR" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/node1-positions-active.json"
NODE0_POSITION=$(jq -er '.positions[-1].id' "$EVIDENCE_DIR/node0-positions-active.json")
NODE1_POSITION=$(jq -er '.positions[-1].id' "$EVIDENCE_DIR/node1-positions-active.json")
test "$(($(uusd_balance "$NODE0_ADDR") - NODE0_BEFORE_STAKE))" -eq -100000000
test "$(($(uusd_balance "$NODE1_ADDR") - NODE1_BEFORE_STAKE))" -eq -100000000
test "$(uusd_balance "$PRINCIPAL_POOL")" -eq 200000000
test "$(uusd_balance "$REWARD_POOL")" -eq 0
test "$("$TERRAD" query bank total-supply-of uusd "${QUERY_FLAGS[@]}" | jq -er '.amount.amount')" = "$SUPPLY_BEFORE_STAKES"
```

Record the returned numeric position IDs and query each with
`terrad query ustcstaking position ID`. Assert:

- active principal is `100000000uusd` for each user;
- node 0 shares are `100000000`, node 1 shares are `200000000`;
- shares are snapshotted with multipliers `1x` and `2x`;
- total active shares are `300000000` and reward index is zero;
- node 0 and node 1 each lost exactly `100000000uusd`;
- principal pool gained `200000000uusd`; reward pool remains zero;
- total `uusd` supply is unchanged.

Snapshot as `after-stakes`, then reconcile exact integer deltas from the raw
bank queries. Any difference is a failure even if the transaction returned
code zero.

## 7. Seed and inspect the distribution community pool

The permissionless distribution command is used only to seed this disposable
local test network. It is not the Phase 2 reward-funding path. Save all balances
before and after this explicit seed action:

```bash
snapshot before-community-pool-seed
expect_ok "$TERRAD" tx distribution fund-community-pool 300000000uusd \
  --from node0 --home "$PWD/build/node0/terrad" \
  | tee "$EVIDENCE_DIR/community-pool-seed-tx.json"
snapshot after-community-pool-seed
```

Reconcile exactly: node 0 decreases by `300000000uusd`; distribution module
bank balance and community-pool decimal accounting increase by exactly that
amount; total supply, principal pool, reward pool, and reward index do not
change. Seed only enough for the planned successful tranche and later failure
scenarios; record the actual source amounts.

Use decimal string arithmetic for the community pool. Example for validating
one before/after `uusd` delta after extracting the `uusd` amount string from
the saved `.pool[]` response:

```bash
python3 -c 'from decimal import Decimal; import sys; before,after,expected=map(Decimal,sys.argv[1:]); assert before-after==expected, (before,after,expected)' \
  "$COMMUNITY_POOL_UUSD_BEFORE" "$COMMUNITY_POOL_UUSD_AFTER" 300000000
```

Set the two variables from the response's `pool` array without passing values
through jq numeric arithmetic. The distribution module account bank balance is
an integer and should be checked independently from FeePool accounting.

## 8. Governance funding proposal, execution, events, and claims

Only governance can fund the USTC reward pool. Do not use a wallet-signed
funding message or create a new `fund-rewards` CLI command. Build a governance
proposal containing the native `MsgFundRewards` type URL and governance
authority:

```bash
jq -n --arg authority "$GOV_AUTH" '
{
  messages: [{
    "@type": "/terra.ustcstaking.v1.MsgFundRewards",
    authority: $authority,
    amount: {denom: "uusd", amount: "300000000"}
  }],
  metadata: "",
  deposit: "10000000stake",
  title: "Fund USTC staking rewards for validator campaign",
  summary: "Transfer the approved localnet tranche from the distribution community pool",
  expedited: false
}' > "$EVIDENCE_DIR/fund-rewards-proposal.json"

snapshot before-governance-funding
expect_ok "$TERRAD" tx gov submit-proposal "$EVIDENCE_DIR/fund-rewards-proposal.json" \
  --from node0 --home "$PWD/build/node0/terrad" \
  | tee "$EVIDENCE_DIR/fund-proposal-submit.json"
FUND_PROPOSAL_ID=$(latest_proposal_id)
"$TERRAD" query gov proposal "$FUND_PROPOSAL_ID" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/fund-proposal-initial.json"
VOTE_START_HEIGHT=$(curl -fsS http://localhost:26657/status | jq -er '.result.sync_info.latest_block_height')
vote_all_yes "$FUND_PROPOSAL_ID"
"$TERRAD" query gov votes "$FUND_PROPOSAL_ID" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/fund-proposal-votes.json"
FUND_RESULT=$(wait_for_proposal_terminal "$FUND_PROPOSAL_ID")
printf '%s\n' "$FUND_RESULT" | tee "$EVIDENCE_DIR/fund-proposal-result.json" \
  | jq '.proposal | {id, status, title, final_tally_result}'
test "$(jq -r '.proposal.status' <<<"$FUND_RESULT")" = PROPOSAL_STATUS_PASSED
```

The USTC module event is emitted when governance executes the proposal, so it
may be in end-block/finalize-block events rather than the proposal-submission
transaction. Scan block results from just before voting through the first
terminal proposal observation and retain the result containing the event:

```bash
LATEST_HEIGHT=$(curl -fsS http://localhost:26657/status | jq -er '.result.sync_info.latest_block_height')
EXECUTION_HEIGHT=
for height in $(seq "$((VOTE_START_HEIGHT - 1))" "$LATEST_HEIGHT"); do
  file="$EVIDENCE_DIR/block-results-$height.json"
  curl -fsS "http://localhost:26657/block_results?height=$height" > "$file"
  if rg -q 'ustcstaking_fund_rewards' "$file"; then
    EXECUTION_HEIGHT="$height"
    cp "$file" "$EVIDENCE_DIR/fund-execution-block-results.json"
  fi
done
test -n "$EXECUTION_HEIGHT"
rg -n 'ustcstaking_fund_rewards|community_pool|reward_index|300000000' \
  "$EVIDENCE_DIR/fund-execution-block-results.json"
```

Record `EXECUTION_HEIGHT` in the report. If the node's RPC has pruned or does not retain block
results, record that limitation and collect the event from a validator's
application logs or an unpruned RPC endpoint; do not claim event verification
without the event data. The successful event must identify the authority,
`source=community_pool`, amount, pre/post reward index, and pre/post reward
pool balance.

Snapshot as `after-governance-funding`. Reconcile these exact outcomes:

| Quantity | Expected change |
|---|---:|
| Distribution FeePool `uusd` | decrease exactly `300000000` |
| Distribution module bank `uusd` | decrease exactly `300000000` |
| USTC reward-pool bank `uusd` | increase exactly `300000000` |
| Principal-pool bank `uusd` | no change |
| Total supply `uusd` | no change |
| Active shares | no change (`300000000`) |
| Reward index | increase exactly `1.000000000000000000` |

Claim node 0's reward now; preserve node 1's full entitlement so the campaign
can verify claiming remains available while paused. Node 1's claim is made
after unbonding and the governance pause test.

```bash
NODE0_BEFORE_CLAIM=$(uusd_balance "$NODE0_ADDR")
expect_ok "$TERRAD" tx ustcstaking claim-rewards "$NODE0_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/node0-claim.json"
test "$(($(uusd_balance "$NODE0_ADDR") - NODE0_BEFORE_CLAIM))" -eq 100000000
snapshot after-claims
```

Verify reward pool is `200000000uusd`, principal pool remains
`200000000uusd`, total supply is unchanged, and node 0's claim event reports
`100000000uusd`. A second claim on node 0 should succeed with `0uusd` and leave
all balances and reward accounting unchanged; save its transaction response
and a snapshot. Node 1's expected `200000000uusd` remains payable and backed by
the reward pool.

## 9. Funding rejection cases and what CLI cannot exercise

The CLI has no ordinary `tx ustcstaking fund-rewards` command because funding
must execute as governance. Use governance proposals for execution-time
rejections. Before each negative proposal, save a full snapshot; after the
proposal reaches terminal status, query status and votes and save another
snapshot.

Run these executable funding failures while active shares exist, or after the
specified lifecycle state is established:

| Case | Proposal message/action | Expected result |
|---|---|---|
| Wrong authority | `MsgFundRewards` with a valid but non-governance authority; governance proposal votes yes | proposal execution fails with `ustcstaking/12` unauthorized; all funding state unchanged |
| Paused module | pause via governance, then valid funding proposal | proposal execution fails with `ustcstaking/11`; all funding state unchanged |
| Insufficient community pool | submit a valid amount greater than current FeePool `uusd` | execution failure corresponds to distribution `ErrBadDistribution` (`distribution/9`); all balances/index unchanged |
| No active shares | after both positions are withdrawn, propose a positive `uusd` amount | execution fails with `ustcstaking/8`; all balances/index unchanged |

The proposal status should be `PROPOSAL_STATUS_FAILED` when execution fails;
capture the failed proposal result, execution block events/logs, and exact
error code if exposed by the chain. Do not count only a rejected proposal
(failure to reach voting threshold) as a tested funding rejection.

Use this helper to create and submit each valid-coin execution failure. It
records the proposal lifecycle and rejects a proposal that passed instead of
failing in execution:

```bash
run_funding_failure_proposal() {
  local label="$1" authority="$2" amount="$3" file="$EVIDENCE_DIR/$1-proposal.json"
  jq -n --arg authority "$authority" --arg amount "$amount" '
  {
    messages: [{
      "@type": "/terra.ustcstaking.v1.MsgFundRewards",
      authority: $authority,
      amount: {denom: "uusd", amount: $amount}
    }],
    metadata: "",
    deposit: "10000000stake",
    title: ("USTC funding negative case: " + $amount),
    summary: "Expected execution failure for validator campaign",
    expedited: false
  }' > "$file"
  snapshot "before-$label"
  expect_ok "$TERRAD" tx gov submit-proposal "$file" \
    --from node0 --home "$PWD/build/node0/terrad" \
    | tee "$EVIDENCE_DIR/$label-submit.json"
  local proposal_id result status
  proposal_id=$(latest_proposal_id)
  printf '%s\n' "$proposal_id" > "$EVIDENCE_DIR/$label-proposal-id.txt"
  vote_all_yes "$proposal_id"
  "$TERRAD" query gov votes "$proposal_id" "${QUERY_FLAGS[@]}" \
    > "$EVIDENCE_DIR/$label-votes.json"
  result=$(wait_for_proposal_terminal "$proposal_id") || return 1
  printf '%s\n' "$result" > "$EVIDENCE_DIR/$label-result.json"
  status=$(jq -r '.proposal.status' <<<"$result")
  test "$status" = PROPOSAL_STATUS_FAILED || {
    echo "$label: expected proposal execution failure, got $status" >&2
    return 1
  }
  snapshot "after-$label"
  assert_ustc_unchanged "before-$label" "after-$label"
}
```

Invoke it with a valid generated user address for the authority mismatch and
a positive amount above the current pool for insufficient funds:

```bash
run_funding_failure_proposal wrong-authority "$NODE2_ADDR" 1000000
POOL_UUSD=$("$TERRAD" query distribution community-pool "${QUERY_FLAGS[@]}" \
  | jq -r '[.pool[]? | select(.denom == "uusd") | .amount][0] // "0"')
AMOUNT_ABOVE_POOL=$(python3 -c 'from decimal import Decimal; import sys; print(int(Decimal(sys.argv[1])) + 1)' "$POOL_UUSD")
run_funding_failure_proposal insufficient-pool "$GOV_AUTH" "$AMOUNT_ABOVE_POOL"
```

For each, inspect the failed proposal's execution height and the associated
failure reason from block results/events. If the public response does not
expose the specific expected module/distribution error code, report the state
non-change but mark the error-code assertion **NOT OBSERVABLE / NOT PASSED**;
the runbook does not treat a generic `PROPOSAL_STATUS_FAILED` as proof of the
expected keeper error. `NODE2_ADDR` and `AMOUNT_ABOVE_POOL` are derived from
the generated account and current decimal-valued pool above.

Run the no-active-shares case only after both positions have been withdrawn;
the pause/resume section below gives its command after completing that lifecycle.

**CLI limitation:** wrong-denomination and zero-amount `MsgFundRewards` fail
`ValidateBasic` (`MsgFundRewards` requires a positive `uusd` coin), and there is
no CLI route to broadcast a direct funding transaction signed by the
governance module account. The normal governance proposal route may reject
these messages before keeper execution. Record these two keeper-level cases as
not exercised by this CLI campaign unless the proposal route demonstrably
executes them and returns the expected `ustcstaking` code. Do not count a JSON
parse error, proposal encoding failure, or unrelated CLI error as a pass. These
cases require an automated application-level message test or a purpose-built
authorized test harness.

## 10. Ownership, unbonding, maturity, and withdrawal

Wrong-owner operations must return `ustcstaking/13` and leave both owners'
state and all accounts/pools unchanged:

```bash
snapshot before-wrong-owner
expect_chain_error ustcstaking 13 "$TERRAD" tx ustcstaking begin-unbonding "$NODE1_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad"
snapshot after-wrong-owner
assert_ustc_unchanged before-wrong-owner after-wrong-owner
```

Begin unbonding for node 0 and query its position. It must show
`POSITION_STATUS_UNBONDING`, zero shares, unchanged `100000000uusd` principal,
and a future `unbonding_end_time`. An immediate withdrawal before the queried
maturity time should return `ustcstaking/15`; if the observed chain time has
already passed the timestamp, this check is inconclusive and must be repeated
with a new position at a longer tier.

```bash
expect_ok "$TERRAD" tx ustcstaking begin-unbonding "$NODE0_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/node0-unbond.json"
"$TERRAD" query ustcstaking position "$NODE0_POSITION" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/node0-unbonding-position.json"
snapshot node0-unbond-started
expect_chain_error ustcstaking 15 "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad"
snapshot node0-premature-withdraw-rejected
assert_ustc_unchanged node0-unbond-started node0-premature-withdraw-rejected
```

Poll `curl http://localhost:26657/status` and compare
`.result.sync_info.latest_block_time` with the position's exact
`unbonding_end_time`. Wait until chain time is later than maturity, then
withdraw. Do not rely on host `sleep` alone:

```bash
NODE0_BEFORE_WITHDRAW=$(uusd_balance "$NODE0_ADDR")
expect_ok "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/node0-withdraw.json"
test "$(($(uusd_balance "$NODE0_ADDR") - NODE0_BEFORE_WITHDRAW))" -eq 100000000
expect_chain_error ustcstaking 14 "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 --home "$PWD/build/node0/terrad"
```

Leave node 1's tier 2 position active for the pause campaign below. At this
point node 0 is withdrawn, node 1 remains active with `200000000` shares, and
the reward pool holds node 1's unpaid `200000000uusd` entitlement.

## 11. Pause and resume through governance

Pause and resume by submitting full `MsgUpdateParams` governance proposals.
Preserve bond denom, both lock tiers, and authority; change only `paused`.
Build the full proposal from current params so the replacement message cannot
accidentally drop tiers or authority:

```bash
make_pause_proposal() {
  local paused="$1" path="$2" params
  params=$("$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" \
    | jq -c --argjson paused "$paused" '.params | .paused = $paused')
  jq -n --arg authority "$GOV_AUTH" --argjson params "$params" \
    --arg title "USTC staking pause state $paused" \
    '{messages:[{"@type":"/terra.ustcstaking.v1.MsgUpdateParams",
      authority:$authority, params:$params}], metadata:"", deposit:"10000000stake",
      title:$title, summary:"Validator campaign pause/resume test", expedited:false}' \
    > "$path"
}

make_pause_proposal true "$EVIDENCE_DIR/pause-proposal.json"
expect_ok "$TERRAD" tx gov submit-proposal "$EVIDENCE_DIR/pause-proposal.json" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/pause-submit.json"
PAUSE_PROPOSAL_ID=$(latest_proposal_id)
vote_all_yes "$PAUSE_PROPOSAL_ID"
PAUSE_RESULT=$(wait_for_proposal_terminal "$PAUSE_PROPOSAL_ID")
printf '%s\n' "$PAUSE_RESULT" | tee "$EVIDENCE_DIR/pause-result.json"
test "$(jq -r '.proposal.status' <<<"$PAUSE_RESULT")" = PROPOSAL_STATUS_PASSED
"$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/params-paused.json"
```

While paused, run the stake rejection against node 2 and compare full
before/after snapshots. Then execute a valid funding proposal and require the
failed-proposal reason to identify `ustcstaking/11`; if the reason is not
observable, mark that error-code assertion not passed:

```bash
snapshot before-paused-stake
expect_chain_error ustcstaking 11 "$TERRAD" tx ustcstaking stake 1000000uusd 1 \
  --from node2 --home "$PWD/build/node2/terrad"
snapshot after-paused-stake
run_funding_failure_proposal paused-funding "$GOV_AUTH" 1000000
```

Node 1's reward claim remains available while paused:

```bash
NODE1_BEFORE_PAUSED_CLAIM=$(uusd_balance "$NODE1_ADDR")
expect_ok "$TERRAD" tx ustcstaking claim-rewards "$NODE1_POSITION" \
  --from node1 --home "$PWD/build/node1/terrad" \
  | tee "$EVIDENCE_DIR/node1-claim-while-paused.json"
test "$(($(uusd_balance "$NODE1_ADDR") - NODE1_BEFORE_PAUSED_CLAIM))" -eq 200000000
snapshot after-node1-claim-while-paused
```

Begin node 1 unbonding while paused, verify its shares leave the active total,
and check that early withdrawal is rejected. Poll chain time against the
queried maturity as in Section 10. Once mature, withdraw while still paused
and verify the exact principal return:

```bash
expect_ok "$TERRAD" tx ustcstaking begin-unbonding "$NODE1_POSITION" \
  --from node1 --home "$PWD/build/node1/terrad" | tee "$EVIDENCE_DIR/node1-unbond-paused.json"
snapshot node1-unbond-started-paused
expect_chain_error ustcstaking 15 "$TERRAD" tx ustcstaking withdraw "$NODE1_POSITION" \
  --from node1 --home "$PWD/build/node1/terrad"
```

After node 1 matures:

```bash
NODE1_BEFORE_PAUSED_WITHDRAW=$(uusd_balance "$NODE1_ADDR")
expect_ok "$TERRAD" tx ustcstaking withdraw "$NODE1_POSITION" \
  --from node1 --home "$PWD/build/node1/terrad" \
  | tee "$EVIDENCE_DIR/node1-withdraw-while-paused.json"
test "$(($(uusd_balance "$NODE1_ADDR") - NODE1_BEFORE_PAUSED_WITHDRAW))" -eq 100000000
snapshot after-node1-withdraw-while-paused
```

Now both positions are withdrawn and active shares are zero. Resume through a
second governance proposal, verify `paused=false`, then run the no-active-
shares funding failure (expected `ustcstaking/8`):

```bash
make_pause_proposal false "$EVIDENCE_DIR/resume-proposal.json"
expect_ok "$TERRAD" tx gov submit-proposal "$EVIDENCE_DIR/resume-proposal.json" \
  --from node0 --home "$PWD/build/node0/terrad" | tee "$EVIDENCE_DIR/resume-submit.json"
RESUME_PROPOSAL_ID=$(latest_proposal_id)
vote_all_yes "$RESUME_PROPOSAL_ID"
RESUME_RESULT=$(wait_for_proposal_terminal "$RESUME_PROPOSAL_ID")
printf '%s\n' "$RESUME_RESULT" | tee "$EVIDENCE_DIR/resume-result.json"
test "$(jq -r '.proposal.status' <<<"$RESUME_RESULT")" = PROPOSAL_STATUS_PASSED
"$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/params-resumed.json"
run_funding_failure_proposal no-active-shares "$GOV_AUTH" 1000000
```

Record proposal IDs, votes, terminal statuses, post-change params,
transaction hashes, events, and before/after snapshots. Confirm that a new
stake succeeds after resume, then close that position so final custody returns
to zero:

```bash
NODE2_BEFORE_RESUMED_STAKE=$(uusd_balance "$NODE2_ADDR")
expect_ok "$TERRAD" tx ustcstaking stake 100000000uusd 1 \
  --from node2 --home "$PWD/build/node2/terrad" \
  | tee "$EVIDENCE_DIR/node2-stake-after-resume.json"
"$TERRAD" query ustcstaking positions "$NODE2_ADDR" "${QUERY_FLAGS[@]}" \
  > "$EVIDENCE_DIR/node2-position-after-resume.json"
NODE2_POSITION=$(jq -er '.positions[-1].id' "$EVIDENCE_DIR/node2-position-after-resume.json")
test "$(($(uusd_balance "$NODE2_ADDR") - NODE2_BEFORE_RESUMED_STAKE))" -eq -100000000
test "$(uusd_balance "$PRINCIPAL_POOL")" -eq 100000000
expect_ok "$TERRAD" tx ustcstaking begin-unbonding "$NODE2_POSITION" \
  --from node2 --home "$PWD/build/node2/terrad"
"$TERRAD" query ustcstaking position "$NODE2_POSITION" "${QUERY_FLAGS[@]}" \
  | tee "$EVIDENCE_DIR/node2-position-unbonding.json"
```

Poll chain time to node 2's recorded maturity, withdraw, and check node 2
received exactly `100000000uusd`. At final lifecycle reconciliation, all three
positions must be withdrawn with zero principal and shares, principal and
reward pools must be zero, community-pool delta must match the approved
tranche, and total `uusd` supply must equal baseline. Save the final snapshot:

```bash
NODE2_BEFORE_RESUMED_WITHDRAW=$(uusd_balance "$NODE2_ADDR")
expect_ok "$TERRAD" tx ustcstaking withdraw "$NODE2_POSITION" \
  --from node2 --home "$PWD/build/node2/terrad" \
  | tee "$EVIDENCE_DIR/node2-withdraw-after-resume.json"
test "$(($(uusd_balance "$NODE2_ADDR") - NODE2_BEFORE_RESUMED_WITHDRAW))" -eq 100000000
test "$(uusd_balance "$PRINCIPAL_POOL")" -eq 0
test "$(uusd_balance "$REWARD_POOL")" -eq 0
snapshot final-lifecycle
```

## 12. Restart and recovery

Take a complete snapshot after successful staking/funding and claims, including
all known position queries, owner queries, reward state, module balances,
community pool, supply, proposal status, and protected LUNC validator fields.
Stop and restart one validator container without deleting its home:

```bash
docker compose stop terradnode3
docker compose start terradnode3
```

Wait until node 3 is caught up, query its RPC endpoint (`localhost:26663` for
node 3), and compare its latest height/hash with node 0. Confirm the shared
state and queries remain identical. Then restart the complete network while
preserving the generated homes:

```bash
docker compose stop
docker compose start
```

Wait for all seven containers to be healthy and block production to resume.
Repeat the full state snapshots. Require identical position records, reward
state, account/pool balances, community-pool accounting, supply, governance
status, and validator snapshots. Record container status, heights, hashes,
logs, and any time to recovery. Never use `make localnet-stop` for a restart;
it deletes generated node data.

## 13. Upgrade rehearsal is a separate release gate

The procedures in this document are the seven-validator functional campaign;
they do not perform an old-state-to-new-binary upgrade rehearsal. Terra
Classic operators do not use Cosmovisor. Do not run a Cosmovisor-based local
upgrade helper as part of this validator campaign or treat its result as
evidence of the Terra Classic upgrade procedure.

Before any public testnet or production activation, the release owner must
schedule a separate rehearsal using the actual Terra Classic operator
deployment process. Record exact old and candidate binary hashes, preserve a
pre-upgrade database snapshot, and compare pre/post USTC supply and balances,
module-account state, validator set and power, delegations, commission,
slashing information, and exported/imported app state. Follow the approved
upgrade and recovery procedure for the target environment; this localnet
runbook does not prescribe or automate that operational handoff.

Validators should report the functional campaign result independently of
this separate gate. A passing local campaign is useful test evidence, but it
does not by itself establish upgrade readiness or authorize activation.

## 14. Final reconciliation and evidence package

For every successful or failed action, retain the raw before/after query JSON,
transaction hash and full result, proposal JSON/ID/votes/status where relevant,
execution-height block results and module events, and an exact reconciliation
table. Every `uusd` supply/account/module amount and share count is an integer
string. Record community-pool `uusd` as a decimal string and compare with
decimal arithmetic. Never use jq or shell arithmetic for decimal pool amounts.

Submit a report with this template:

```text
Candidate commit SHA:
Candidate binary SHA-256 / build source:
Host OS, architecture, CPU, memory:
Go version / Docker version / Compose version / jq version:
Localnet image ID:
Chain ID / validator count / timeout_commit / gov timing:
Run start/end UTC:
Validator operator participating / contact:

Scenario | PASS/FAIL/BLOCKED | Transaction or proposal IDs | Evidence path | Notes
Fresh 7-node InitChain and block production:
Seven funded genesis accounts and baseline totals:
Governance tier proposal execution:
Direct stake and share snapshots:
Wrong-denom and invalid-tier expected errors:
Community-pool seed accounting:
Governance reward funding and execution event:
Exact supply/community-pool/distribution/reward reconciliation:
Reward claims, repeat claim, and claim events:
Wrong-owner, unbonding, premature withdrawal, and matured withdrawal:
Funding failure cases (authority, paused, insufficient pool, no shares):
Wrong-denom/zero funding CLI limitation or separate harness evidence:
Governance pause/resume and allowed operations while paused:
Single-node restart and recovery:
Full-network restart and recovery:
Protected LUNC validator/delegation/slashing snapshots:
Upgrade rehearsal (separate release gate; report status separately):
Final supply/pool/reward/position reconciliation:
Unexpected behavior, logs, and reproducible steps:
Validator conclusion and outstanding concerns:
```

Attach the entire `$EVIDENCE_DIR`, including environment inventory, candidate
hashes, generated genesis (remove secret key material), raw snapshots,
transaction/proposal results, events, Compose status, logs, and reconciliation.
Never publish keyring secrets, mnemonics, or private validator keys.

## 15. Completion criteria and cleanup

The behavioral campaign passes only if all runnable scenarios have the
expected chain result and exact state delta, all seven nodes recover from both
restart checks, total `uusd` supply is conserved across staking/funding/claims,
module account permissions are empty, and protected LUNC validator state is
unchanged. Report the old-state-to-candidate upgrade rehearsal separately: it
is a release gate, not part of this local functional campaign. List
CLI-limited wrong-denom/zero funding cases separately; they need automated
application-level coverage.

Stop containers while keeping generated data:

```bash
docker compose stop
```

Delete only the disposable generated localnet when the evidence is safely
stored:

```bash
make localnet-stop
```

This target deletes `build/node*` and `build/gentxs`. Preserve the test report
and `$EVIDENCE_DIR`; do not delete or publish validator secrets.
