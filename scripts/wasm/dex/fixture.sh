#!/bin/bash
source scripts/wasm/env-test-pre.sh

echo "Uploading contracts..."

# Configuration
FACTORY_CONTRACT="scripts/wasm/dex/artifacts/terraswap_factory.wasm"
PAIR_CONTRACT="scripts/wasm/dex/artifacts/terraswap_pair.wasm"
ROUTER_CONTRACT="scripts/wasm/dex/artifacts/terraswap_router.wasm"

# Initialize variables to store code IDs
FACTORY_CODE_ID=""
PAIR_CODE_ID=""
ROUTER_CODE_ID=""

# Function to upload a contract and return code_id
upload_contract() {
    local contract_path=$1
    echo "Uploading contract: $contract_path..."
    
    out=$($BINARY tx wasm store "$contract_path" \
        --from $KEY \
        --chain-id $CHAIN_ID \
        --gas 20000000 \
        --fees 575529204uluna \
        --keyring-backend $KEYRING \
        --home $HOME \
        --output json \
        -y)

    sleep 1
    txhash=$(echo $out | jq -r '.txhash')
    code_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
    echo "Contract uploaded successfully with code_id: $code_id"
    echo $code_id  # Return the code_id
}

# Check if files exist before uploading
for contract in "$FACTORY_CONTRACT" "$PAIR_CONTRACT" "$ROUTER_CONTRACT"; do
    if [ ! -f "$contract" ]; then
        echo "Error: Contract file $contract not found!"
        exit 1
    fi
done

# uploading the contracts
FACTORY_CODE_ID=$(upload_contract "$FACTORY_CONTRACT")
PAIR_CODE_ID=$(upload_contract "$PAIR_CONTRACT")
ROUTER_CODE_ID=$(upload_contract "$ROUTER_CONTRACT")
