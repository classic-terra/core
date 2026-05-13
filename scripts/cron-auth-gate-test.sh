#!/bin/bash

set -euo pipefail

ROOT=$(pwd)
HOME_DIR=mytestnet
DENOM=${2:-uluna}
BINARY=${1:-_build/new/terrad}
KEYRING="test"
CHAIN_ID="localterra"
KEY="test0"
KEY1="test1"
KEY2="test2"
SED_BINARY=sed

if [[ "$OSTYPE" == "darwin"* ]]; then
	if ! command -v gsed &> /dev/null
	then
		echo "gsed could not be found. Please install it with 'brew install gnu-sed'"
		exit 1
	else
		SED_BINARY=gsed
	fi
fi

mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

if ! command -v "$BINARY" &> /dev/null && [[ ! -x "$BINARY" ]]
then
	GOBIN="$ROOT/_build/new" go install -mod=readonly ./...
	BINARY="$ROOT/_build/new/terrad"
fi

rm -rf "$HOME_DIR"
pkill terrad || true

"$BINARY" init --chain-id "$CHAIN_ID" moniker --home "$HOME_DIR"

"$BINARY" keys add "$KEY" --keyring-backend "$KEYRING" --home "$HOME_DIR"
"$BINARY" keys add "$KEY1" --keyring-backend "$KEYRING" --home "$HOME_DIR"
"$BINARY" keys add "$KEY2" --keyring-backend "$KEYRING" --home "$HOME_DIR"

"$BINARY" add-genesis-account "$KEY" "1000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$HOME_DIR"
"$BINARY" add-genesis-account "$KEY1" "1000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$HOME_DIR"
"$BINARY" add-genesis-account "$KEY2" "1000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$HOME_DIR"

update_test_genesis() {
	cat "$HOME_DIR/config/genesis.json" | jq "$1" > "$HOME_DIR/config/tmp_genesis.json" && mv "$HOME_DIR/config/tmp_genesis.json" "$HOME_DIR/config/genesis.json"
}

update_test_genesis '.app_state["mint"]["params"]["mint_denom"]="'$DENOM'"'
update_test_genesis '.app_state["gov"]["deposit_params"]["min_deposit"]=[{"denom":"'$DENOM'","amount":"1000000"}]'
update_test_genesis '.app_state["gov"]["params"]["min_deposit"]=[{"denom":"'$DENOM'","amount":"1000000"}]'
update_test_genesis '.app_state["gov"]["params"]["voting_period"]="30s"'
update_test_genesis '.app_state["gov"]["voting_params"]["voting_period"]="30s"'
if cat "$HOME_DIR/config/genesis.json" | jq -e '.app_state["gov"]["params"]["expedited_voting_period"]' > /dev/null 2>&1; then
	update_test_genesis '.app_state["gov"]["params"]["expedited_voting_period"]="4s"'
fi
update_test_genesis '.app_state["crisis"]["constant_fee"]={"denom":"'$DENOM'","amount":"1000"}'
update_test_genesis '.app_state["staking"]["params"]["bond_denom"]="'$DENOM'"'
update_test_genesis '.app_state["cron"]={
  "params":{"limit":"1","max_execution_gas":"5000000"},
  "schedule_list":[]
}'

"$SED_BINARY" -i '0,/enable = false/s//enable = true/' "$HOME_DIR/config/app.toml"
"$SED_BINARY" -i 's/swagger = false/swagger = true/' "$HOME_DIR/config/app.toml"
"$SED_BINARY" -i -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' "$HOME_DIR/config/app.toml"
"$SED_BINARY" -i -e 's/max-txs = 5000/max-txs = 3/g' "$HOME_DIR/config/app.toml"
"$SED_BINARY" -i -e 's/timeout_commit = "5s"/timeout_commit = "500ms"/g' "$HOME_DIR/config/config.toml"

"$BINARY" gentx "$KEY" "900000000000${DENOM}" --commission-rate=0.01 --commission-max-rate=0.02 --keyring-backend "$KEYRING" --chain-id "$CHAIN_ID" --home "$HOME_DIR"
"$BINARY" collect-gentxs --home "$HOME_DIR"

LOGFILE="$HOME_DIR/log-screen.txt"
"$BINARY" start --home "$HOME_DIR" --log_level debug >"$LOGFILE" 2>&1 &
CRON_NODE_PID=$!

cleanup() {
	if kill -0 "$CRON_NODE_PID" 2>/dev/null; then
		kill "$CRON_NODE_PID" || true
		wait "$CRON_NODE_PID" || true
	fi
}
trap cleanup EXIT

sleep 20

echo "QUERY GOV AUTHORITY"
GOV_AUTHORITY=$("$BINARY" q auth module-account gov --home "$HOME_DIR" --output json | jq -r '.account.value.address // .account.base_account.address // .account.address')
echo "$GOV_AUTHORITY"

echo "EXPECT CRON WRITE TO FAIL FOR NON-AUTHORITY SENDER"
set +e
OUT=$("$BINARY" tx cron add-schedule bad-cron 999999 end-blocker terra1contract '{"ping":{}}' --from "$KEY" --keyring-backend "$KEYRING" --chain-id "$CHAIN_ID" --home "$HOME_DIR" --fees "20000${DENOM}" --gas auto --gas-adjustment 1.5 --broadcast-mode block -y --output json 2>&1)
CODE=$?
set -e

echo "$OUT"
if [ "$CODE" -eq 0 ]; then
	echo "expected cron add-schedule to fail for non-authority sender" >&2
	exit 1
fi

if echo "$OUT" | jq -e '.raw_log | test("invalid authority")' >/dev/null 2>&1 || echo "$OUT" | grep -q "invalid authority"; then
	echo "auth gate check passed"
else
	echo "expected invalid authority error in tx output" >&2
	exit 1
fi
