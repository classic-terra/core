#!/bin/bash

# expecting that TXHASH from wasm-deploy.sh will be exported
# querying TXHASH after upgrade to see if it still works

set +e

read -r -a CONTRACTS <<< ${CONTRACT_ADDRESSES_STRING:-""}

echo "CONTRACTS = ${CONTRACTS[@]}"

# loop through OLD_TXHASH
for i in "${CONTRACTS[@]}"; do
    echo "getting new state of contract $i"
    ./_build/new/terrad q wasm contract-state all $i --output json --home $HOME > scripts/wasm/contract_states/new_$i.json
done