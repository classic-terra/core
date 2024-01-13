#!/bin/bash
set -o errexit -o nounset -o pipefail -o xtrace
shopt -s expand_aliases

alias chainA="terrad --node http://localhost:26657 --chain-id localterra-a"
alias chainB="terrad --node http://localhost:36657 --chain-id localterra-b"
alias chainC="terrad --node http://localhost:46657 --chain-id localterra-c"

# setup the keys
echo "bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort" | terrad --keyring-backend test keys add validator --recover || echo "key exists"
echo "increase bread alpha rigid glide amused approve oblige print asset idea enact lawn proof unfold jeans rabbit audit return chuckle valve rather cactus great" | terrad --keyring-backend test keys add faucet --recover || echo "key exists"
echo "organ aisle angle end rude robust travel genre team town devote program" | terrad --keyring-backend test keys add receiver --recover || echo "key exists"

VALIDATOR=$(terrad keys show validator --keyring-backend test -a)
RECEIVER=$(terrad keys show receiver --keyring-backend test -a)

args="--keyring-backend test --gas 200000 --gas-prices 0.1uluna --gas-adjustment 1.3 --broadcast-mode block --yes"
TX_FLAGS=($args)

# send money to the validator on both chains
chainA tx bank send faucet "$VALIDATOR" 1000000000uluna "${TX_FLAGS[@]}"
chainB tx bank send faucet "$VALIDATOR" 1000000000uluna "${TX_FLAGS[@]}"
chainC tx bank send faucet "$VALIDATOR" 1000000000uluna "${TX_FLAGS[@]}"

args="--keyring-backend test --gas 2000000 --gas-prices 0.1uluna --gas-adjustment 1.3 --broadcast-mode block --yes"
TX_FLAGS=($args)

# query old bank balances
denom=$(chainC query bank balances "$RECEIVER" -o json | jq -r '.balances[0].denom')
balance=$(chainC query bank balances "$RECEIVER" -o json | jq -r '.balances[0].amount')

# send ibc transaction to execute the pfm 
MEMO='{"forward":{"receiver":"'"$RECEIVER"'","port":"transfer", "channel":"channel-0" }}'
chainA tx ibc-transfer transfer transfer channel-0 $RECEIVER 100uluna \
       --from validator --keyring-backend test -y  \
       --memo "$MEMO"

# wait for the ibc round trip
sleep 16

# query new bank balances
new_balance=$(chainC query bank balances "$RECEIVER" -o json | jq -r '.balances[0].amount')
# export ADDR_IN_CHAIN_A=$(chainC q ibchooks wasm-sender channel-0 "$VALIDATOR")

echo "denom: $denom, old balance: $balance, new balance: $new_balance"
