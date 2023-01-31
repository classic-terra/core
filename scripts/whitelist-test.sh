#!/bin/bash

HOME=mytestnet
ROOT=$(pwd)
DENOM=uluna
KEY="test"
KEY1="test1"
KEY2="test2"
KEYRING="test"
CHAIN_ID="test"

# underscore so that go tool will not take gocache into account
mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

# install new binary
if ! command -v _build/new/terrad &> /dev/null
then
    GOBIN="$ROOT/_build/new" go install -mod=readonly ./...
fi

# start old node
screen -L -Logfile mytestnet/log-screen.txt -dmS node1 bash scripts/run-node.sh _build/new/terrad $DENOM

sleep 20

# add test1 as whitelist
test1=$(./_build/new/terrad keys show $KEY1 -a --keyring-backend $KEYRING --home $HOME)
test2=$(./_build/new/terrad keys show $KEY2 -a --keyring-backend $KEYRING --home $HOME)
echo "addresses = $test1,$test2"
./_build/new/terrad tx gov submit-proposal whitelist-add "$test1,$test2" --title "whitelist" --description "whitelist"  --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

./_build/new/terrad tx gov deposit 1 "20000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

./_build/new/terrad tx gov vote 1 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

./_build/new/terrad tx gov vote 1 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

while true; do 
    PROPOSAL_STATUS=$(./_build/new/terrad q gov proposal 1 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

# check whitelist again
./_build/new/terrad q treasury whitelist -o json | jq ".addresses"