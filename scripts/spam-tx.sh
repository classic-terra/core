#!/bin/bash
set -ue

# Configuration
BINARY="./build/terrad"
CHAIN_ID="localterra"
HOME_DIR="mytestnet"
KEYRING="test"
FROM_KEY="test0"
TO_ADDRESS="terra1yy0tteehf5xyg64mj48mxuygxeyj73a9skue4v"
GAS="500000"
GAS_PRICES="1000uluna"

# Get validator address
VALIDATOR_ADDR=$($BINARY q staking validators -o json | jq -r '.validators[0].operator_address')

# Get initial sequence number
SEQUENCE=$($BINARY q account $($BINARY keys show $FROM_KEY --keyring-backend $KEYRING --home $HOME_DIR -a) -o json | jq -r '.sequence')
# Get account number
ACCOUNT_NUMBER=$($BINARY q account $($BINARY keys show $FROM_KEY --keyring-backend $KEYRING --home $HOME_DIR -a) -o json | jq -r '.account_number')

for i in $(seq $SEQUENCE 2 100)
do
    echo "sequence number is $i"

    # Generate send tx
        $BINARY tx bank send $FROM_KEY $TO_ADDRESS 100000uluna \
            --from=$FROM_KEY \
            --gas=$GAS \
            --gas-prices=$GAS_PRICES \
            --chain-id=$CHAIN_ID \
            --home $HOME_DIR \
            --keyring-backend $KEYRING \
            --sequence=$((i)) \
            --generate-only > send_tx.json

        # Sign send tx
        $BINARY tx sign send_tx.json \
            --from=$FROM_KEY \
            --chain-id=$CHAIN_ID \
            --home=$HOME_DIR \
            --keyring-backend=$KEYRING \
            --sequence=$((i)) \
            --offline \
            --account-number=$ACCOUNT_NUMBER > signed_send_tx.json

        # Broadcast send tx
        $BINARY tx broadcast signed_send_tx.json > send_result.log
        cat send_result.log

        # Check for sequence mismatch
        if [ $(grep -c "mismatch" send_result.log) -eq 1 ]
        then
            echo "sequence number mismatch"
            break
        fi

    # Generate oracle prevote tx
    $BINARY tx oracle aggregate-prevote 10 10ukrw $VALIDATOR_ADDR \
        --from=$FROM_KEY \
        --gas=$GAS \
        --gas-prices=$GAS_PRICES \
        --chain-id=$CHAIN_ID \
        --home $HOME_DIR \
        --keyring-backend $KEYRING \
        --sequence=$((i+1)) \
        --generate-only > oracle_tx.json

    # Sign oracle tx
    $BINARY tx sign oracle_tx.json \
        --from=$FROM_KEY \
        --chain-id=$CHAIN_ID \
        --home=$HOME_DIR \
        --keyring-backend=$KEYRING \
        --sequence=$((i+1)) \
        --offline \
        --account-number=$ACCOUNT_NUMBER > signed_oracle_tx.json

    # Broadcast oracle tx
    $BINARY tx broadcast signed_oracle_tx.json > oracle_result.log
    cat oracle_result.log

    # Check for sequence mismatch
    if [ $(grep -c "mismatch" oracle_result.log) -eq 1 ]
    then
        echo "sequence number mismatch"
        break
    fi

    sleep 3
done

echo "Script finished due to sequence mismatch"