#!/bin/sh

set +e

BINARY=_build/new/terrad
STARGATE_TESTER="wasmbinding/testdata/stargate_tester.wasm"
KEYRING_BACKEND="test"
HOME=mytestnet
CHAIN_ID=localterra

# store stargate-tester
echo "... stores a wasm"
addr=$($BINARY keys show test0 -a --home $HOME --keyring-backend $KEYRING_BACKEND)
out=$($BINARY tx wasm store ${STARGATE_TESTER} --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not store contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
echo "$txhash"
tx_result=$($BINARY q tx $txhash -o json)
code_id=$(echo "$tx_result" | jq -r '
    (.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value) //
    (.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value) //
    empty
')
echo "code_id $code_id"

# instantiate stargate-tester
echo "... instantiates contract"
msg='{}'
CMD=($BINARY tx wasm instantiate $code_id "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --no-admin --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
if $BINARY tx wasm instantiate --help 2>&1 | grep -q -- "--label"; then
    CMD+=(--label stargate-tester)
fi
out=$("${CMD[@]}")
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not instantiate contract" >&2
    echo $out >&2
    exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
echo "$txhash"
tx_result=$($BINARY q tx $txhash -o json)
contract_addr=$(echo "$tx_result" | jq -r '
    (.events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value) //
    (.logs[0].events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value) //
    empty
')
echo "contract_addr $contract_addr"

# call stargate-tester
echo "... query tax rate"
msg='{"tax_rate":{}}'
out=$($BINARY query wasm contract-state smart $contract_addr $msg --output json)
echo $out