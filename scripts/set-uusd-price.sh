#!/bin/bash

while true
do
    # Your script logic here
    echo "Running script..."
    min=0.0008
    max=0.0009
    random_price=$(awk -v min="$min" -v max="$max" 'BEGIN{srand(); print min+rand()*(max-min)}')
    formatted_float=$(printf "%.8f" "$random_price")  # Adjust precision as needed
    echo "$formatted_float"
    
    ./build/terrad tx oracle aggregate-prevote 1234 $formatted_float'uusd' --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 20
    echo "Aggregating vote"
    ./build/terrad tx oracle aggregate-vote 1234 $formatted_float'uusd' $(./build/terrad q staking validators --output json | jq -r '.validators[0].operator_address') --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 10
    echo "Done"
done
