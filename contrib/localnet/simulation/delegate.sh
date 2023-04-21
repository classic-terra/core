#!/bin/sh

# get all validators on the network
VALIDATORS=($(terrad q staking validators --node $(sh $SIMULATION_FOLDER/next_node.sh) -o json | jq -r '.validators[].operator_address'))

# Loop through each node* folder
for operator_address in ${VALIDATORS[@]}
do
    for i in $(seq 0 3); do
        terrad tx staking delegate $operator_address 1000000uluna --chain-id $CHAIN_ID --from test$i --gas auto --gas-adjustment 2.3 --fees 20000000uluna --keyring-backend $KEYRING_BACKEND --home $NODE_HOME --node $(sh $SIMULATION_FOLDER/next_node.sh) -y
        sleep 10
    done
done