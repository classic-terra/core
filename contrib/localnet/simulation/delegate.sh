#!/bin/sh

# Loop through each node* folder
for folder in "${TESTNET_FOLDER}"/node*/
do
    val_addr_name=$(basename $folder)
    val_addr=$(terrad keys show $val_addr_name -a --bech val --home ${folder}terrad --keyring-backend $KEYRING_BACKEND)
    for i in $(seq 0 3); do
        terrad tx staking delegate $val_addr 1000000uluna --chain-id $CHAIN_ID --from test$i --keyring-backend $KEYRING_BACKEND -y
        sleep 10
    done
done