#!/bin/sh

# create a new validator node in local
# if /Users/thevinhnguyen/.terra/config/priv_validator_key.jso exists
# then remove it
if [ ! -f $HOME/config/priv_validator_key.json ]; then
    terrad init test0 --chain-id $CHAIN_ID --home $HOME
fi

# create a validator for a node
terrad tx staking create-validator --moniker test0 \
--from test0 \
--amount="1000000uluna" \
--fees 20uluna \
--pubkey="$(terrad tendermint show-validator --home $HOME)" \
--details="this is a validator" \
--commission-max-rate="0.10" \
--commission-max-change-rate="0.05" \
--commission-rate="0.05" \
--min-self-delegation 1 \
--chain-id $CHAIN_ID \
--keyring-backend $KEYRING_BACKEND \
--home $HOME \
-y

sleep 10

# check if command `terrad q staking validator $(terrad keys show test0 -a --bech val --keyring-backend test)` success
terrad q staking validator $(terrad keys show test0 -a --bech val --keyring-backend test --home $HOME) >/dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "VALIDATOR CREATED SUCCESSFULLY"
else
    echo "FAILED TO CREATE VALIDATOR"
fi