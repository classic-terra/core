#!/bin/bash

set -e

# This script is used to deploy the contract to the network.
BINARY=_build/old/terrad
ASSEMBLY="scripts/wasm/contracts/old_astroport_assembly.wasm"
BUILDERUNLOCK="scripts/wasm/contracts/old_astroport_builder_unlock.wasm"
XASTROC="scripts/wasm/contracts/old_xastroc.wasm"
ASTROC="scripts/wasm/contracts/old_astroc_token.wasm"
KEYRING_BACKEND="test"
HOME=mytestnet
CHAIN_ID=localterra

# CONTRACT ADDRESS array
# xastroc, astroc, builder unlock, assembly
CONTRACT_ADDRESS=()

# ====== STORE OLD CONTRACTS ======
echo "... stores builder unlock"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${BUILDERUNLOCK} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store binary" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
builder_unlock_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
echo "BUILDERUNLOCK CODE = $builder_unlock_id"
echo ""

echo "... stores assembly"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${ASSEMBLY} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store binary" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
assembly_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
echo "ASSEMBLY CODE = $assembly_id"
echo ""

echo "... stores xastroc"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${XASTROC} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store binary" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
xastroc_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
echo "XASTROC CODE = $xastroc_id"
echo ""

echo "... stores astroc"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${ASTROC} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store binary" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
astroc_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
echo "ASTROC CODE = $astroc_id"
echo ""

echo "... stores dummy"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${XASTROC} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store binary" >&2
    echo $out >&2
    exit $code
fi
sleep 10

# ====== INSTATIATE OLD CONTRACTS ======
echo "... instantiates xastroc"
addr1=$($BINARY keys show test1 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
addr2=$($BINARY keys show test2 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
msg=$(jq -n '
{
    "name": "Staked Astroport",
    "symbol": "xASTRO",
    "decimals": 6,
    "initial_balances": [
        {
            "address": "'$addr'",
            "amount": "1000000000000000"
        },
        {
            "address": "'$addr1'",
            "amount": "1000000000000000"
        },
        {
            "address": "'$addr2'",
            "amount": "1000000000000000"
        }
    ],
}')
echo $msg
out=$($BINARY tx wasm instantiate $xastroc_id "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not instantiate contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
XASTROC_ADDR=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[0].attributes[3].value')
CONTRACT_ADDRESS+=($XASTROC_ADDR)

echo "... instantiates astroc"
msg=$(jq -n '
{
    "name": "Astroport",
    "symbol": "ASTRO",
    "decimals": 6,
    "initial_balances": [
        {
            "address": "'$addr'",
            "amount": "1000000000000000"
        }
    ]
}')
echo $msg
out=$($BINARY tx wasm instantiate $astroc_id "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not instantiate contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
ASTROC_ADDR=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[0].attributes[3].value')
CONTRACT_ADDRESS+=($ASTROC_ADDR)

echo "... instantiates builder unlock"
msg=$(jq -n '
{
    "owner": "'$addr'",
    "max_allocations_amount": "100000000",
    "astro_token": "'$ASTROC_ADDR'"
}')
echo $msg
out=$($BINARY tx wasm instantiate $builder_unlock_id "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not instantiate contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
BUILDER_UNLOCK_ADDR=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[0].attributes[3].value')
CONTRACT_ADDRESS+=($BUILDER_UNLOCK_ADDR)

# 50 block voting period
# 13000 block effective delay
# 87000 block expiration period
# 1000xastroc required deposit
echo "... instantiates assembly"
msg=$(jq -n '
{
    "xastro_token_addr": "'$XASTROC_ADDR'",
    "builder_unlock_addr": "'$BUILDER_UNLOCK_ADDR'",
    "proposal_voting_period": 20,
    "proposal_effective_delay": 13000,
    "proposal_expiration_period": 87000,
    "proposal_required_deposit": "1000",
    "proposal_required_quorum": "0.1",
    "proposal_required_threshold": "0.50",
    "whitelisted_links": [
        "https://forum.astroport.fi/",
        "http://forum.astroport.fi/",
        "https://astroport.fi/",
        "http://astroport.fi/"
    ]
}')
echo $msg
out=$($BINARY tx wasm instantiate $assembly_id "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not instantiate contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
ASSEMBLY_ADDR=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[0].attributes[3].value')
CONTRACT_ADDRESS+=($ASSEMBLY_ADDR)

CONTRACT_ADDRESS_STRING="${CONTRACT_ADDRESS[*]}"
echo "CONTRACT_ADDRESS = $CONTRACT_ADDRESS_STRING"
export CONTRACT_ADDRESS_STRING