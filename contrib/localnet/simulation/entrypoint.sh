#!/bin/sh

export NODE_HOME=${NODE_HOME:-~/.terra}
export SIMULATION_FOLDER=$(dirname $(realpath "$0"))
export TESTNET_FOLDER=$(echo $SIMULATION_FOLDER | sed 's|\(.*core\).*|\1|')/build
export KEYRING_BACKEND=test
export CHAIN_ID=${CHAIN_ID:-localterra}

echo $CHAIN_ID

if [ ! -d "$NODE_HOME" ]; then
    terrad init moniker --chain-id $CHAIN_ID --home $NODE_HOME
fi

# initialize keys
for i in $(seq 0 3); do
    # check if test$i exists
    terrad keys show test$i --keyring-backend $KEYRING_BACKEND --home $NODE_HOME >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "y" | terrad keys delete test$i --keyring-backend $KEYRING_BACKEND --home $NODE_HOME
    fi

    key=$(jq ".keys[$i] | tostring" $SIMULATION_FOLDER/network/$CHAIN_ID/keys.json )
    keyname=$(echo $key | jq -r 'fromjson | ."keyring-keyname"')
    mnemonic=$(echo $key | jq -r 'fromjson | .mnemonic')
    # Add new account
    echo $mnemonic | terrad keys add $keyname --keyring-backend $KEYRING_BACKEND --home $NODE_HOME --recover
done

# if chain-id is localterra
if [ "$CHAIN_ID" = "localterra" ]; then
    # copy genesis.json to $NODE_HOME
    cp $TESTNET_FOLDER/node0/terrad/config/genesis.json $NODE_HOME/config/genesis.json

    # tx_send
    # need better design to send transaction for all kind of network
    sh $SIMULATION_FOLDER/tx_send.sh

    echo "DONE TX SEND SIMULATION (1/5)"
fi

# create-validator
sh $SIMULATION_FOLDER/create-validator.sh

echo "DONE CREATE VALIDATOR SIMULATION (2/5)"

# delegate
sh $SIMULATION_FOLDER/delegate.sh

echo "DONE DELEGATION SIMULATION (3/5)"

# contracts
sh $SIMULATION_FOLDER/contract.sh

echo "DONE CONTRACT SIMULATION (4/5)"

#governance
sh $SIMULATION_FOLDER/gov.sh

echo "DONE GOV SIMULATION (5/5)"