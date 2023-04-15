#!/bin/sh

# Loop through each node* folder
for folder in "${TESTNET_FOLDER}"/node*/
do
    val_addr_name=$(basename $folder)
    val_addr=$(terrad keys show $val_addr_name -a --home ${folder}terrad --keyring-backend $KEYRING_BACKEND)
    for i in $(seq 0 3); do
        addr=$(terrad keys show test$i -a --keyring-backend $KEYRING_BACKEND)
        terrad tx bank send $val_addr $addr 100000000uluna --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y
        sleep 10
    done
done