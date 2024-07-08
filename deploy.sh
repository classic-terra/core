#!/bin/bash

terrad tx wasm store forwarder.wasm --from test0 --keyring-backend test --home mytestnet --gas 2000000 -y

sleep 5

terrad tx wasm instantiate 1 {}  --label test --no-admin --from test0 --keyring-backend test --home mytestnet -y

sleep 5

contract_address=$(terrad q wasm list-contract-by-code 1 --output json | jq -r '.contracts[0]')

echo "Contract_addr:" $contract_address

recipient=$(terrad keys show test1 --keyring-backend test --home mytestnet --output json | jq -r '.address')

echo "Recipient:" $recipient

sleep 5

terrad tx wasm execute $contract_address '{"forward":{"recipient":"'$recipient'","amount":"100000"}}' --from test0 --keyring-backend test --home mytestnet  --amount 100000uluna

# terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au