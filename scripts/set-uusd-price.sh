#!/bin/bash

while true
do
    # Your script logic here
    echo "Running script..."
    
    ./build/terrad tx oracle aggregate-prevote 1234 0.1uusd --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 20
    echo "Aggregating vote"
    ./build/terrad tx oracle aggregate-vote 1234 0.1uusd terravaloper1n5zc4r4nl2ug9jnkgjtepl5pztrfw3q7q79hp9 --from test0 --chain-id localterra --home mytestnet --keyring-backend test -y
    sleep 10
    echo "Done"
done
