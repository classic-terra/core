#!/bin/bash

while true
do
    # Your script logic here
    echo "Running script..."
    
    ./build/terrad tx oracle aggregate-prevote 1234 0.1uusd --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 20
    echo "Aggregating vote"
    ./build/terrad tx oracle aggregate-vote 1234 0.1uusd $(./build/terrad q staking validators --output json | jq -r '.validators[0].operator_address') --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 10
    echo "Done"
done
