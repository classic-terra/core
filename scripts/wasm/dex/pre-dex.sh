source scripts/wasm/dex/fixture.sh
source scripts/wasm/dex/dex-utils.sh
source scripts/wasm/utils.sh

echo "PRE-DEX: Adding pair"

PAIR_ADDRESS=$(create_pair $FACTORY_CONTRACT_ADDRESS $NATIVE_TOKEN $TOKEN_CONTRACT_ADDRESS)
echo "PAIR_ADDRESS: $PAIR_ADDRESS"

echo "PRE-DEX: Adding liquidity"
provide_liquidity $PAIR_ADDRESS $NATIVE_TOKEN '10000000000' $TOKEN_CONTRACT_ADDRESS "1000000000000"

echo "PRE-DEX: Executing swap"
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


