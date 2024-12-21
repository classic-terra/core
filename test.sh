#!/bin/bash

BINARY=terrad
CONTINUE=${CONTINUE:-"false"}
HOME_DIR=mytestnet
ENV=${ENV:-""}


# check DENOM is set. If not, set to uluna
DENOM=${2:-uluna}

COMMISSION_RATE=0.01
COMMISSION_MAX_RATE=0.02


CHAIN_ID="localterra"
KEYRING="test"
KEY="test0"
KEY1="test1"
KEY2="test2"

terrad tx distribution fund-community-pool "50000000000000${DENOM}" --from $KEY1 --keyring-backend test  --chain-id $CHAIN_ID --gas 201421 --fees 6665000uluna --home $HOME_DIR -y 

sleep 2
terrad tx gov submit-proposal send_prop.json --from $KEY1 --keyring-backend test  --chain-id $CHAIN_ID --gas 201421 --fees 6665000uluna --home $HOME_DIR -y 


#deposit 
sleep 2
terrad tx gov deposit 1 "4000000${DENOM}" --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME_DIR -y --fees 5665000uluna 

# vote 
sleep 2
terrad tx gov vote 1 yes --from test0 --keyring-backend test --chain-id $CHAIN_ID --home $HOME_DIR -y --fees 5665000uluna

sleep 2

terrad tx gov vote 1 yes --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME_DIR -y --fees 5665000uluna
