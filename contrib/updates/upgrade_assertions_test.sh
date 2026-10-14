#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/upgrade_assertions.sh"
reject() { if "$@" >/dev/null 2>&1; then echo "unexpected acceptance: $*" >&2; exit 1; fi; }
assert_tx_success '{"code":0,"height":"8"}'
reject assert_tx_success '{"code":9,"height":"8"}'
reject assert_tx_success '{"code":0,"height":"0"}'
[[ $(proposal_id_from_tx '{"events":[{"type":"submit_proposal","attributes":[{"key":"proposal_id","value":"17"}]}]}') == 17 ]]
reject proposal_id_from_tx '{"events":[]}'
assert_passed '{"proposal":{"status":"PROPOSAL_STATUS_PASSED"}}'
reject assert_passed '{"proposal":{"status":"PROPOSAL_STATUS_REJECTED"}}'
assert_plan '{"plan":{"name":"v14_5","height":"100"}}' v14_5 100
reject assert_plan '{"plan":{"name":"v99","height":"100"}}' v14_5 100
assert_applied '{"height":"100"}' 100
reject assert_applied '{"height":"99"}' 100
assert_candidate abc abc '{"result":{"response":{"version":"4.1"}}}' 4.1
reject assert_candidate old abc '{"result":{"response":{"version":"4.1"}}}' 4.1
reject assert_candidate abc abc '{"result":{"response":{"version":"4.0"}}}' 4.1
assert_height '{"sync_info":{"latest_block_height":"110"}}' 110
reject assert_height '{"SyncInfo":{"latest_block_height":"100"}}' 110
assert_common_hash ABCD ABCD
reject assert_common_hash ABCD DEFA
reject assert_common_hash '' ''
assert_ustc_store '{"params":{"bond_denom":"uusd","authority":"terra1gov"}}' '{"module_versions":[{"name":"ustcstaking","version":"1"}]}'
reject assert_ustc_store '{}' '{"module_versions":[]}'
reject assert_ustc_store '{"params":{}}' '{"module_versions":[]}'
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cat > "$tmp/genesis.json" <<'JSON'
{"app_state":{"gov":{"params":{"voting_period":"172800s"}},"bank":{"balances":[{"address":"user","coins":[{"denom":"uluna","amount":"900719925474099312345"},{"denom":"uusd","amount":"7"}]}],"supply":[{"denom":"uluna","amount":"900719925474099312345"},{"denom":"uusd","amount":"7"}]}}}
JSON
prepare_upgrade_genesis "$tmp/genesis.json" user
python3 - "$tmp/genesis.json" <<'PY'
import json,sys
s=json.load(open(sys.argv[1]))['app_state']
assert s['gov']['params']['voting_period']=='60s'
assert s['bank']['balances'][0]['coins']==[{'denom':'uluna','amount':'900719925474099312345'},{'denom':'uusd','amount':'1000007'}]
assert s['bank']['supply']==[{'denom':'uluna','amount':'900719925474099312345'},{'denom':'uusd','amount':'1000007'}]
PY
cat > "$tmp/empty-supply.json" <<'JSON'
{"app_state":{"gov":{"params":{"voting_period":"172800s"}},"bank":{"balances":[{"address":"user","coins":[{"denom":"uluna","amount":"999999999999999999999999999999"}]}],"supply":[]}}}
JSON
prepare_upgrade_genesis "$tmp/empty-supply.json" user
python3 - "$tmp/empty-supply.json" <<'PYTEST'
import json,sys
b=json.load(open(sys.argv[1]))['app_state']['bank']
assert b['supply']==[]
assert b['balances'][0]['coins']==[{'denom':'uluna','amount':'999999999999999999999999999999'},{'denom':'uusd','amount':'1000000'}]
PYTEST
reject prepare_upgrade_genesis "$tmp/genesis.json" missing_account
# Polling failure must be bounded and nonzero, including a permanently stalled node.
reject wait_until 0 false
printf 'upgrade assertion fixtures passed\n'
