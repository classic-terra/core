#!/usr/bin/env bash
# Shared checks deliberately fail closed on missing, malformed, or wrong evidence.
assert_tx_success() { jq -e 'has("code") and (.code|tonumber)==0 and (.height|tonumber)>0' <<< "$1" >/dev/null; }
proposal_id_from_tx() {
    jq -er '[((.events // []) + [.logs[]?.events[]?])[] | select(.type == "submit_proposal") | .attributes[] | select(.key == "proposal_id") | .value] | unique | if length == 1 and (.[0]|test("^[1-9][0-9]*$")) then .[0] else error("missing or ambiguous proposal ID") end' <<< "$1"
}
assert_passed() { jq -e '(.proposal // .).status == "PROPOSAL_STATUS_PASSED"' <<< "$1" >/dev/null; }
assert_plan() { jq -e --arg name "$2" --arg height "$3" '.plan.name == $name and (.plan.height|tostring)==$height' <<< "$1" >/dev/null; }
assert_applied() { jq -e --arg height "$2" '(.height|tostring)==$height' <<< "$1" >/dev/null; }
assert_candidate() {
    [[ -n "$1" && "$1" == "$2" ]] || return 1
    jq -e --arg version "$4" '.result.response.version==$version' <<< "$3" >/dev/null
}
assert_height() { jq -e --argjson height "$2" '((.sync_info // .SyncInfo).latest_block_height|tonumber)>=$height' <<< "$1" >/dev/null; }
assert_common_hash() { [[ -n "$1" && "$1" == "$2" ]]; }
assert_ustc_store() {
    jq -e '.params | type == "object" and .bond_denom == "uusd" and (.authority|type == "string" and length>0)' <<< "$1" >/dev/null || return 1
    jq -e 'any(.module_versions[]; .name=="ustcstaking" and (.version|tonumber)>=1)' <<< "$2" >/dev/null
}
wait_until() {
    local timeout=$1 deadline=$((SECONDS + $1)); shift
    while true; do
        if "$@"; then return 0; fi
        if ((SECONDS >= deadline)); then echo "Timed out after ${timeout}s: $*" >&2; return 1; fi
        sleep "${POLL_INTERVAL:-2}"
    done
}
# Python integers preserve SDK coin amounts that exceed IEEE-754/jq precision.
prepare_upgrade_genesis() {
    python3 - "$1" "$2" <<'PY'
import json,sys
from pathlib import Path
p=Path(sys.argv[1]); g=json.loads(p.read_text()); s=g['app_state']
s['gov']['params']['voting_period']='60s'
bank=s['bank']; entries=[b for b in bank['balances'] if b['address']==sys.argv[2]]
if len(entries)!=1: raise SystemExit('expected exactly one funded node1 account')
def add(coins):
    found=[c for c in coins if c['denom']=='uusd']
    if len(found)>1: raise SystemExit('duplicate uusd genesis coin')
    if found: found[0]['amount']=str(int(found[0]['amount'])+1000000)
    else: coins.append({'denom':'uusd','amount':'1000000'})
    coins.sort(key=lambda c:c['denom'])
add(entries[0]['coins'])
# An empty supply means SDK derives it. Preserve that representation.
if bank.get('supply'): add(bank['supply'])
t=p.with_suffix('.json.tmp'); t.write_text(json.dumps(g,indent=2)+'\n'); t.replace(p)
PY
}
