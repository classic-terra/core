source scripts/wasm/env-test-pre.sh

# Function to get native token balance
get_native_token_balance() {
    local address=$1
    local denom=$2
    local balance=$($BINARY q bank balances $address --output json | jq -r '.balances[] | select(.denom=="'$denom'").amount')
    printf "%s" "${balance:-0}"  # Return 0 if balance is null/empty
}

# Function to get CW20 token balance
get_cw20_token_balance() {
    local address=$1
    local token_contract=$2
    local query="{\"balance\":{\"address\":\"$address\"}}"
    local balance=$($BINARY query wasm contract-state smart $token_contract "$query" --output json | jq -r '.data.balance')
    printf "%s" "${balance:-0}"  # Return 0 if balance is null/empty
}

# Generic function to get token balance (detects token type)
get_token_balance() {
    local address=$1
    local token=$2
    
    >&2 echo "Token: $token"
    >&2 echo "Address: $address"
    if [[ $token == terra* ]]; then
        get_cw20_token_balance "$address" "$token"
    else
        get_native_token_balance "$address" "$token"
    fi
}


get_address_from_key() {
    local key=$1
    
    # log the query 
    local address=$($BINARY keys show $key --output json --keyring-backend $KEYRING --home $HOME | jq -r '.address')
    printf "%s" "$address"
}