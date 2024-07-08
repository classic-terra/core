#!/bin/bash

terrad tx wasm store forwarder4.wasm --from test0 --keyring-backend test --home mytestnet --gas 2000000 -y

sleep 5

terrad tx wasm instantiate 1 {}  --label test --no-admin --from test0 --keyring-backend test --home mytestnet -y

sleep 5

terrad tx wasm instantiate 1 {}  --label test --no-admin --from test0 --keyring-backend test --home mytestnet -y

sleep 5

contract_address0=$(terrad q wasm list-contract-by-code 1 --output json | jq -r '.contracts[0]')
contract_address1=$(terrad q wasm list-contract-by-code 1 --output json | jq -r '.contracts[1]')

echo "Contract_addr0:" $contract_address0
echo "Contract_addr1:" $contract_address1

recipient=$(terrad keys show test1 --keyring-backend test --home mytestnet --output json | jq -r '.address')

echo "Recipient:" $recipient

sleep 5

terrad tx wasm execute $contract_address0 '{"forward_to_contract":{"contract":"'$contract_address1'","recipient":"'$recipient'","amount":"10000"}}' --from test0 --keyring-backend test --home mytestnet  --amount 10000uluna --gas 500000

# terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au