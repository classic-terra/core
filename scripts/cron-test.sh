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

# underscore so that go tool will not take gocache into account
mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

# install new binary if it is not already available
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

# Allocate genesis accounts (cosmos formatted addresses)
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
  "params":{"limit":"1"},
  "schedule_list":[
    {
      "name":"cron-smoke",
      "period":"999999",
      "msgs":[{"contract":"terra1contract","msg":"{\"ping\":{}}"}],
      "last_execute_height":"0",
      "execution_stage":"EXECUTION_STAGE_END_BLOCKER"
    },
    {
      "name":"cron-smoke-begin",
      "period":"999998",
      "msgs":[{"contract":"terra1contract","msg":"{\"pong\":{}}"}],
      "last_execute_height":"0",
      "execution_stage":"EXECUTION_STAGE_BEGIN_BLOCKER"
    }
  ]
}'

# enable rest server and swagger
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

echo "QUERY CRON PARAMS"
"$BINARY" q cron params --home "$HOME_DIR" --chain-id "$CHAIN_ID" -o json | jq ".params"

echo "QUERY CRON SCHEDULE"
"$BINARY" q cron schedule cron-smoke-begin --home "$HOME_DIR" --chain-id "$CHAIN_ID" -o json | jq ".schedule"

echo "QUERY CRON SCHEDULES"
"$BINARY" q cron schedules --home "$HOME_DIR" --chain-id "$CHAIN_ID" --limit 10 -o json | jq ".schedules"
