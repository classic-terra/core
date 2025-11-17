source scripts/wasm/env-test-pre.sh

# Function to upload a contract and return code_id
upload_contract() {
    local contract_path=$1
    
    out=$($BINARY tx wasm store "$contract_path" \
        --from $KEY \
        --chain-id $CHAIN_ID \
        --gas 20000000 \
        --fees 575529204uluna \
        --keyring-backend $KEYRING \
        --home $HOME \
        --output json \
        -y)

    sleep $SLEEP_TIME
    txhash=$(echo $out | jq -r '.txhash')
    code_id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')
    
    # Only return the code_id to stdout
    printf "%s" "$code_id"
}


# Function to instantiate a contract
instantiate_contract() {
    local contract_path=$1
    local code_id=$2
    local msg=$3
    local label=$4

    
    out=$($BINARY tx wasm instantiate $code_id "$msg" \
        --label "$label" \
        --from $KEY \
        --chain-id $CHAIN_ID \
        --gas 20000000 \
        --fees 575529204uluna \
        --no-admin \
        --keyring-backend $KEYRING \
        --home $HOME \
        --output json \
        -y)

    sleep $SLEEP_TIME  # Wait for transaction to be processed
    txhash=$(echo $out | jq -r '.txhash')
    
    # Query the tx and extract contract address from events
    tx_response=$($BINARY q tx $txhash --output json)
    contract_address=$(echo "$tx_response" | jq -r '.logs[0].events[] | select(.type=="instantiate").attributes[] | select(.key=="_contract_address").value')
    
    printf "%s" "$contract_address"
}

# Function to execute a contract
execute_contract() {
    local contract_addr=$1
    local msg=$2
    local key=$3

    out=$($BINARY tx wasm execute $contract_addr "$msg" --from $key \
        --chain-id $CHAIN_ID \
        --gas 20000000 \
        --fees 575529204uluna \
        --keyring-backend $KEYRING \
        --home $HOME \
        --output json -y)

    echo $out
}

# Function to query a contract
query_contract() {
    local contract_addr=$1
    local msg=$2

    out=$($BINARY query wasm contract-state smart $contract_addr "$msg" --output json)

    echo $out
}

