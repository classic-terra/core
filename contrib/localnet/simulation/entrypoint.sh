#!/bin/sh

export HOME=${HOME:-~/.terra}
export SIMULATION_FOLDER=$(dirname $(realpath "$0"))
export TESTNET_FOLDER=$(echo $SIMULATION_FOLDER | sed 's|\(.*core\).*|\1|')/build
export KEYRING_BACKEND=test
export CHAIN_ID=${CHAIN_ID:-localterra}

if [ ! -d "$HOME" ]; then
    terrad init test0 --chain-id $CHAIN_ID --home $HOME
fi

# initialize keys
for i in $(seq 0 3); do
    key=$(jq ".keys[$i] | tostring" $SIMULATION_FOLDER/keys.json )
    keyname=$(echo $key | jq -r 'fromjson | ."keyring-keyname"')
    mnemonic=$(echo $key | jq -r 'fromjson | .mnemonic')
    # Add new account
    echo $mnemonic | terrad keys add $keyname --keyring-backend $KEYRING_BACKEND --home $HOME --recover
done

# copy genesis.json to $HOME
cp $TESTNET_FOLDER/node0/terrad/config/genesis.json $HOME/config/genesis.json

# tx_send
sh $SIMULATION_FOLDER/tx_send.sh

echo "DONE TX SEND SIMULATION (1/5)"

# delegate
sh $SIMULATION_FOLDER/delegate.sh

echo "DONE DELEGATION SIMULATION (2/5)"

# create-validator
sh $SIMULATION_FOLDER/create-validator.sh

echo "DONE CREATE VALIDATOR SIMULATION (3/5)"

# contracts
sh $SIMULATION_FOLDER/contract.sh

echo "DONE CONTRACT SIMULATION (4/5)"

#governance
sh $SIMULATION_FOLDER/gov.sh

echo "DONE GOV SIMULATION (5/5)"

