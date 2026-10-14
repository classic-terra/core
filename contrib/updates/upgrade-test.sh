#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$SCRIPT_DIR/upgrade_assertions.sh"
: "${OLD_VERSION:?Set OLD_VERSION explicitly}"
: "${OLD_REVISION:?Set OLD_REVISION explicitly}"
: "${SOFTWARE_UPGRADE_NAME:?Set SOFTWARE_UPGRADE_NAME explicitly}"
[[ "$OLD_REVISION" =~ ^[0-9a-f]{40}$ ]] || exit 1
TESTNET_NVAL=${1:-7}
[[ "$TESTNET_NVAL" == 7 ]] || { echo 'Rehearsal requires all seven validators' >&2; exit 1; }
UPGRADE_FIXTURE=${UPGRADE_FIXTURE:-clean}
[[ "$UPGRADE_FIXTURE" == clean || "$UPGRADE_FIXTURE" == prefunded ]] || exit 1
EVIDENCE_DIR=${EVIDENCE_DIR:-build/upgrade-evidence}
mkdir -p "$EVIDENCE_DIR"
capture_logs() {
    local code=$?
    trap - EXIT
    for ((i=0;i<TESTNET_NVAL;i++)); do docker logs "terradnode$i" > "$EVIDENCE_DIR/node$i.log" 2>&1 || true; done
    exit "$code"
}
trap capture_logs EXIT
old() { timeout 20 docker exec terradnode1 ./old/terrad "$@"; }
new() { local i=$1; shift; timeout 20 docker exec "terradnode$i" ./terrad "$@" --home "node$i/terrad"; }
rpc() { timeout 15 docker exec "terradnode$1" curl --fail --silent --max-time 10 "http://localhost:26657/$2"; }
status_ready() { local status; status=$(old status --home node1/terrad 2>/dev/null) || return 1; assert_height "$status" 1; }
wait_until 120 status_ready
STATUS=$(old status --home node1/terrad)
CHAIN_ID=$(jq -er '(.node_info // .NodeInfo).network' <<< "$STATUS")
# Allow bounded voting and inclusion checks to finish well before the halt.
UPGRADE_HEIGHT=$(( $(jq -er '(.sync_info // .SyncInfo).latest_block_height|tonumber' <<< "$STATUS") + 100 ))
CANDIDATE_SHA=$(docker exec terradnode1 sha256sum ./terrad | awk '{print $1}')
CANDIDATE_VERSION=$(docker exec terradnode1 ./terrad version)
OLD_METADATA=$(old version --long --output json)
jq -e --arg revision "$OLD_REVISION" --arg version "${OLD_VERSION#v}" '.commit==$revision and .version==$version' <<< "$OLD_METADATA" >/dev/null
printf '%s\n' "$OLD_METADATA" > "$EVIDENCE_DIR/predecessor.json"
TX_SEQ=0
# Broadcast success is insufficient; wait for inclusion and inspect DeliverTx code.
submit_tx() {
    local response hash included
    response=$(old tx "$@" --keyring-backend test --chain-id "$CHAIN_ID" --fees 100000uluna --gas auto --gas-adjustment 1.5 --broadcast-mode sync --output json -y)
    jq -e 'has("code") and (.code|tonumber)==0' <<< "$response" >/dev/null
    hash=$(jq -er '.txhash | select(length>0)' <<< "$response")
    TX_SEQ=$((TX_SEQ+1))
    tx_included() { included=$(old query tx "$hash" --home node1/terrad --output json 2>/dev/null) || return 1; }
    wait_until 60 tx_included
    printf '%s\n' "$included" > "$EVIDENCE_DIR/tx-$TX_SEQ.json"
    assert_tx_success "$included"
    LAST_TX=$included
}
# SHA256(module name)[:20], bech32 terra; SDK NewModuleAddress addresses.
POOLS=(terra1e0a6f2ust9zhyq4rsldzx9he9q069fka77dszc terra14e3gu5c7rrquce8h0vn3ujk2v37jf80d626z0d)
if [[ "$UPGRADE_FIXTURE" == prefunded ]]; then
    SENDER=$(old keys show node1 -a --keyring-backend test --home node1/terrad)
    for address in "${POOLS[@]}"; do
        submit_tx bank send "$SENDER" "$address" 1uusd,1uluna --from node1 --home node1/terrad
        balances=$(old query bank balances "$address" --home node1/terrad --output json)
        jq -e 'any(.balances[]; .denom=="uusd" and .amount=="1") and any(.balances[]; .denom=="uluna" and .amount=="1")' <<< "$balances" >/dev/null
    done
fi
AUTHORITY=$(old query auth module-account gov --home node1/terrad --output json | jq -er '.account.base_account.address')
[[ "$(old query upgrade authority --home node1/terrad --output json | jq -er '.address')" == "$AUTHORITY" ]]
jq -n --arg authority "$AUTHORITY" --arg name "$SOFTWARE_UPGRADE_NAME" --arg height "$UPGRADE_HEIGHT" '{messages:[{"@type":"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",authority:$authority,plan:{name:$name,height:$height,info:""}}],metadata:"",deposit:"20000000uluna",title:"USTC staking rehearsal",summary:"Activate native USTC staking",expedited:false}' > "$EVIDENCE_DIR/proposal.json"
docker cp "$EVIDENCE_DIR/proposal.json" terradnode1:/terrad/upgrade-proposal.json
submit_tx gov submit-proposal /terrad/upgrade-proposal.json --from node1 --home node1/terrad
PROPOSAL_ID=$(proposal_id_from_tx "$LAST_TX")
for ((i=0;i<TESTNET_NVAL;i++)); do
    submit_tx gov vote "$PROPOSAL_ID" yes --from "node$i" --home "node$i/terrad"
done
proposal_passed() {
    local proposal status
    proposal=$(old query gov proposal "$PROPOSAL_ID" --home node1/terrad --output json) || return 1
    printf '%s\n' "$proposal" > "$EVIDENCE_DIR/proposal-status.json"
    status=$(jq -r '(.proposal // .).status' <<< "$proposal")
    case "$status" in PROPOSAL_STATUS_REJECTED|PROPOSAL_STATUS_FAILED) echo "Governance failed: $status" >&2; exit 1;; esac
    assert_passed "$proposal"
}
wait_until 90 proposal_passed
assert_plan "$(old query upgrade plan --home node1/terrad --output json)" "$SOFTWARE_UPGRADE_NAME" "$UPGRADE_HEIGHT"
# The passed plan must be observed while the old chain is still below H.
[[ $(old status --home node1/terrad | jq -er '(.sync_info // .SyncInfo).latest_block_height|tonumber') -lt "$UPGRADE_HEIGHT" ]]
all_nodes_advanced() {
    local i status
    for ((i=0;i<TESTNET_NVAL;i++)); do
        status=$(new "$i" status 2>/dev/null) || return 1
        assert_height "$status" "$((UPGRADE_HEIGHT+10))" || return 1
    done
}
wait_until 900 all_nodes_advanced
CONSENSUS_HASH=''
verify_nodes() {
    local i actual abci applied params versions hash common_hash=''
    for ((i=0;i<TESTNET_NVAL;i++)); do
        # Inspect the running daemon executable, independently of the query client.
        actual=$(docker exec "terradnode$i" sh -c 'for p in /proc/[0-9]*; do if [ "$(cat "$p/comm" 2>/dev/null)" = terrad ]; then sha256sum "$p/exe"; fi; done' | awk '{print $1}')
        abci=$(rpc "$i" abci_info)
        assert_candidate "$actual" "$CANDIDATE_SHA" "$abci" "$CANDIDATE_VERSION"
        applied=$(new "$i" query upgrade applied "$SOFTWARE_UPGRADE_NAME" --output json)
        assert_applied "$applied" "$UPGRADE_HEIGHT"
        params=$(new "$i" query ustcstaking params --output json)
        versions=$(new "$i" query upgrade module-versions ustcstaking --output json)
        assert_ustc_store "$params" "$versions"
        hash=$(rpc "$i" "block?height=$((UPGRADE_HEIGHT+10))" | jq -er '.result.block.header.app_hash | select(length>0)')
        if [[ -n "$common_hash" ]]; then assert_common_hash "$hash" "$common_hash"; else common_hash=$hash; fi
        jq -n --arg sha "$actual" --argjson abci "$abci" --argjson applied "$applied" --argjson params "$params" --argjson versions "$versions" --arg app_hash "$hash" '{sha256:$sha,abci:$abci,applied:$applied,params:$params,module_versions:$versions,app_hash:$app_hash}' > "$EVIDENCE_DIR/node$i-proof.json"
    done
    if [[ -n "$CONSENSUS_HASH" ]]; then assert_common_hash "$common_hash" "$CONSENSUS_HASH"; else CONSENSUS_HASH=$common_hash; fi
}
verify_nodes
for address in "${POOLS[@]}"; do
    balances=$(new 1 query bank balances "$address" --height "$UPGRADE_HEIGHT" --output json)
    expected_uluna=0; [[ "$UPGRADE_FIXTURE" == clean ]] || expected_uluna=1
    jq -e --arg expected "$expected_uluna" '([.balances[] | select(.denom=="uusd") | .amount][0] // "0")=="0" and ([.balances[] | select(.denom=="uluna") | .amount][0] // "0")==$expected' <<< "$balances" >/dev/null
    printf '%s\n' "$balances" > "$EVIDENCE_DIR/$address.json"
done
before=$(new 1 query distribution community-pool --height "$((UPGRADE_HEIGHT-1))" --output json)
after=$(new 1 query distribution community-pool --height "$UPGRADE_HEIGHT" --output json)
python3 - "$before" "$after" "$UPGRADE_FIXTURE" <<'PY'
import json,sys
from decimal import Decimal, getcontext
getcontext().prec=200
def uusd(raw): return sum((Decimal(c['amount']) for c in json.loads(raw)['pool'] if c['denom']=='uusd'), Decimal(0))
expected=Decimal(2 if sys.argv[3]=='prefunded' else 0)
assert uusd(sys.argv[2])-uusd(sys.argv[1])==expected, 'activation failed to credit FeePool with exact USTC sweep'
PY
printf '%s\n' "$before" > "$EVIDENCE_DIR/community-pool-before.json"
printf '%s\n' "$after" > "$EVIDENCE_DIR/community-pool-after.json"
# Restart one validator and require catchup plus unchanged fixed-height consensus proof.
RESTART_HEIGHT=$(( $(new 0 status | jq -er '(.sync_info // .SyncInfo).latest_block_height|tonumber') + 5 ))
docker restart terradnode6
restarted_caught_up() { local status; status=$(new 6 status 2>/dev/null) || return 1; assert_height "$status" "$RESTART_HEIGHT"; }
wait_until 180 restarted_caught_up
verify_nodes
echo "Upgrade $SOFTWARE_UPGRADE_NAME at $UPGRADE_HEIGHT verified on all seven nodes ($UPGRADE_FIXTURE); evidence: $EVIDENCE_DIR"
