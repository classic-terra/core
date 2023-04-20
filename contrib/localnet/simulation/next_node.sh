#!/bin/sh
# during one of simulation, the node that receives transaction died, probably due to having to handle too much.
# this script is to choose the next node for transaction to use

RPC=("http://localhost:26657" "http://localhost:26660" "http://localhost:26662" "http://localhost:26664" "http://localhost:26666" "http://localhost:26668" "http://localhost:26670")

retry=0
while true; do
    if [ $retry -eq 3 ]; then
        echo "Maximum retry reached, cannot choose next active node..."
        exit 1
    fi

    NODE=${RPC[$((RANDOM % 7))]}

    # check if chosen node is alive
    curl -s $NODE/status &> /dev/null

    if [ $? -eq 0 ]; then
        echo $NODE
        exit 0
    else
        retry=$((retry + 1))
    fi
done