#!/bin/bash

source scripts/wasm/env-test-pre.sh

create_asset_info_json() {
    local input=$1
    if [[ $input == terra* ]]; then
        echo "{\"token\":{\"contract_addr\":\"$input\"}}"
    else
        echo "{\"native_token\":{\"denom\":\"$input\"}}"
    fi
}

create_asset_json() {
    local input=$1
    local amount=${2:-"0"}
    if [[ $input == terra* ]]; then
        echo "{\"info\":{\"token\":{\"contract_addr\":\"$input\"}},\"amount\":\"$amount\"}"
    else
        echo "{\"info\":{\"native_token\":{\"denom\":\"$input\"}},\"amount\":\"$amount\"}"
    fi
}

create_pair() {
    sleep $SLEEP_TIME 

    local factory_address=$1
    local token1=$2
    local token2=$3
    
    if [ -z "$token1" ] || [ -z "$token2" ]; then
        >&2 echo "Error: Both token addresses/denoms are required"
        return 1
    fi

    >&2 echo "Creating pair for tokens:"

    local asset1=$(create_asset_json "$token1")
    local asset2=$(create_asset_json "$token2")
    
    >&2 echo "Asset 1: $asset1"
    >&2 echo "Asset 2: $asset2"

    local msg=$(cat << EOF
{
    "create_pair": {
        "assets": [$asset1,$asset2]
    }
}
EOF
)

    >&2 echo "Creating pair..."
    out=$($BINARY tx wasm execute "$factory_address" "$msg" \
        --from "$KEY" \
        --chain-id "$CHAIN_ID" \
        --gas 20000000 \
        --fees 1124975000uluna \
        --keyring-backend "$KEYRING" \
        --home "$HOME" \
        --output json \
        -y)
    
    sleep $SLEEP_TIME
    txhash=$(echo $out | jq -r '.txhash')
    
    sleep $SLEEP_TIME
    tx_response=$($BINARY q tx $txhash --output json)
    pair_address=$(echo "$tx_response" | jq -r '.logs[0].events[] | select(.type=="wasm").attributes[] | select(.key=="pair_contract_addr").value')
    
    printf "%s" "$pair_address"
}

query_pair_address() {
    local factory_address=$1
    local token1=$2
    local token2=$3

    local pair_query="{\"pair\":{\"asset_infos\":[$(create_asset_info_json $token1),$(create_asset_info_json $token2)]}}"
    local pair_info=$($BINARY query wasm contract-state smart $factory_address "$pair_query" --output json)
    echo $(echo $pair_info | jq -r '.data.contract_addr')
}

increase_allowance() {
    local token_address=$1
    local spender=$2
    local amount=$3

    >&2 echo "Increasing allowance for token $token_address..."
    out=$($BINARY tx wasm execute $token_address \
        "{\"increase_allowance\":{\"spender\":\"$spender\",\"amount\":\"$amount\"}}" \
        --from "$KEY" \
        --chain-id "$CHAIN_ID" \
        --gas 20000000 \
        --fees 11124975000uluna \
        --keyring-backend "$KEYRING" \
        --home "$HOME" \
        --output json \
        -y)
    
    txhash=$(echo $out | jq -r '.txhash')
    sleep $SLEEP_TIME
    tx_response=$($BINARY q tx $txhash --output json)
    sleep $SLEEP_TIME
}

provide_liquidity() {
    local factory_address=$1
    local token1=$2
    local amount1=$3
    local token2=$4
    local amount2=$5

    >&2 echo "Providing liquidity..."

    local pair_address=$(query_pair_address "$factory_address" "$token1" "$token2")

    local asset1=$(create_asset_json "$token1" "$amount1")
    local asset2=$(create_asset_json "$token2" "$amount2")

    if [[ $token1 == terra* ]]; then
        increase_allowance "$token1" "$pair_address" "$amount1"
    fi
    if [[ $token2 == terra* ]]; then
        increase_allowance "$token2" "$pair_address" "$amount2"
    fi

    local funds=""
    if [[ $token1 != terra* ]]; then
        funds="$funds--amount $amount1$token1 "
    fi
    if [[ $token2 != terra* ]]; then
        funds="$funds--amount $amount2$token2 "
    fi

    local msg=$(cat << EOF
{
    "provide_liquidity": {
        "assets": [$asset1,$asset2]
    }
}
EOF
)

    out=$($BINARY tx wasm execute "$pair_address" "$msg" \
        --from "$KEY" \
        --chain-id "$CHAIN_ID" \
        --gas 20000000 \
        --fees 1124975000uluna \
        $funds \
        --keyring-backend "$KEYRING" \
        --home "$HOME" \
        --output json \
        -y)

    sleep $SLEEP_TIME
    txhash=$(echo $out | jq -r '.txhash')
    
    sleep $SLEEP_TIME
    tx_response=$($BINARY q tx $txhash --output json)
}

create_base64_msg() {
    local msg=$1
    echo "$msg" | base64
}

execute_swap() {
    local router_address=$1
    local token1=$2
    local amount=$3
    local token2=$4
    local min_receive=${5:-"0"}
    local deadline=${6:-$(($(date +%s) + 120))}

    local offer_asset_info=$(create_asset_info_json "$token1")
    local ask_asset_info=$(create_asset_info_json "$token2")

    local swap_msg=$(cat << EOF
{
  "execute_swap_operations": {
    "operations": [
      {
        "terra_swap": {
          "offer_asset_info": $offer_asset_info,
          "ask_asset_info": $ask_asset_info
        }
      }
    ],
    "minimum_receive": "$min_receive",
    "deadline": $deadline
  }
}
EOF
)

    if [[ $token1 == terra* ]]; then
        >&2 echo "Sending CW20 tokens to router..."
        local send_msg=$(cat << EOF
{
  "send": {
    "contract": "$router_address",
    "amount": "$amount",
    "msg": "$(create_base64_msg "$swap_msg")"
  }
}
EOF
)
        out=$($BINARY tx wasm execute "$token1" "$send_msg" \
            --from "$KEY" \
            --chain-id "$CHAIN_ID" \
            --gas 20000000 \
            --fees 1124975000uluna \
            --keyring-backend "$KEYRING" \
            --home "$HOME" \
            --output json \
            -y)

    else
        >&2 echo "Executing swap through router..."
        local funds="--amount $amount$token1"
        
        out=$($BINARY tx wasm execute "$router_address" "$swap_msg" \
            --from "$KEY" \
            --chain-id "$CHAIN_ID" \
            --gas 20000000 \
            --fees 1124975000uluna \
            $funds \
            --keyring-backend "$KEYRING" \
            --home "$HOME" \
            --output json \
            -y)
    fi

    sleep $SLEEP_TIME
    txhash=$(echo $out | jq -r '.txhash')
    
    sleep $SLEEP_TIME
    tx_response=$($BINARY q tx $txhash --output json)
}

