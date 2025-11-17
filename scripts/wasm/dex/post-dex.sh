#!/bin/bash
source scripts/wasm/env-test-pre.sh
source scripts/wasm/dex/dex-utils.sh
source scripts/wasm/utils.sh

echo "--------------------------------"
# Load deployment info
if [ -f "scripts/wasm/dex/deployment_info.sh" ]; then
    source scripts/wasm/dex/deployment_info.sh
else
    echo "Error: deployment_info.sh not found. Please run fixture.sh first"
    exit 1
fi


echo "POST-DEX: Asserting token balance"

# Get user address
USER_ADDRESS=$(get_address_from_key $KEY)
echo "USER_ADDRESS: $USER_ADDRESS"

TOKEN_BALANCE_BEFORE=$(get_token_balance $USER_ADDRESS $TOKEN_CONTRACT_ADDRESS)
echo "TOKEN_BALANCE_BEFORE: $TOKEN_BALANCE_BEFORE"


echo "POST-DEX: Executing swap"
# Swap parameters
SWAP_AMOUNT="100000000"
MIN_RECEIVE="0"
DEADLINE=$(($(date +%s) + 120))  # 2 minutes from now


# Balance before 
echo "TOKEN_BALANCE_BEFORE: $(get_token_balance $(get_address_from_key $KEY) $TOKEN_CONTRACT_ADDRESS)"

# Execute the swap
RECEIVED_AMOUNT=$(execute_swap "$ROUTER_CONTRACT_ADDRESS" "$NATIVE_TOKEN" "$SWAP_AMOUNT" "$TOKEN_CONTRACT_ADDRESS" "$MIN_RECEIVE" "$DEADLINE")

# Balance after
echo "TOKEN_BALANCE_AFTER: $(get_token_balance $(get_address_from_key $KEY) $TOKEN_CONTRACT_ADDRESS)"
