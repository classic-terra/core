#!/bin/bash

# Expecting CONTRACT_ADDRESSES_STRING exported from wasm-deploy.sh.
# Queries old contracts with new binary after upgrade:
#   1. contract-state all   — dump full state and save to file
#   2. contract-state raw   — raw key lookup (contract_info)
#   3. contract-state smart — smart query (num_tokens for cw721)

set +e

BINARY=_build/new/terrad
HOME=mytestnet
FAILURES=0
SUMMARY=()

read -r -a CONTRACTS <<< "${CONTRACT_ADDRESSES_STRING:-""}"

if [ ${#CONTRACTS[@]} -eq 0 ]; then
    echo "ERROR: No contract addresses found. Was wasm-deploy.sh run as a pre-script?"
    exit 1
fi

echo "=== QUERYING OLD CONTRACTS WITH NEW BINARY ==="
echo "CONTRACTS = ${CONTRACTS[@]}"

for contract_addr in "${CONTRACTS[@]}"; do
    echo ""
    echo "--- Contract: $contract_addr ---"

    # 1. contract-state all
    echo "  [1/3] contract-state all"
    out=$($BINARY q wasm contract-state all "$contract_addr" --output json --home $HOME 2>&1)
    if [ $? -ne 0 ]; then
        echo "  FAIL: contract-state all for $contract_addr"
        echo "  Error: $out"
        FAILURES=$((FAILURES + 1))
        SUMMARY+=("FAIL | $contract_addr | contract-state all | $out")
    else
        mkdir -p scripts/wasm/contract_states/
        echo "$out" > "scripts/wasm/contract_states/new_${contract_addr}.json"
        num_models=$(echo "$out" | jq '.models | length')
        echo "  OK: $num_models state entries (saved to contract_states/new_${contract_addr}.json)"
        SUMMARY+=("OK   | $contract_addr | contract-state all | $num_models state entries")
    fi

    # 2. contract-state raw (key "contract_info" = hex 636F6E74726163745F696E666F)
    echo "  [2/3] contract-state raw (contract_info)"
    out=$($BINARY q wasm contract-state raw "$contract_addr" 636F6E74726163745F696E666F --output json --home $HOME 2>&1)
    if [ $? -ne 0 ]; then
        echo "  FAIL: contract-state raw (contract_info) for $contract_addr"
        echo "  Error: $out"
        FAILURES=$((FAILURES + 1))
        SUMMARY+=("FAIL | $contract_addr | contract-state raw  | $out")
    else
        data=$(echo "$out" | jq -r '.data // empty')
        if [ -z "$data" ]; then
            echo "  FAIL: contract-state raw (contract_info) returned empty data for $contract_addr"
            FAILURES=$((FAILURES + 1))
            SUMMARY+=("FAIL | $contract_addr | contract-state raw  | empty data")
        else
            echo "  OK: $(echo "$out" | jq -c '.')"
            SUMMARY+=("OK   | $contract_addr | contract-state raw  | data returned")
        fi
    fi

    # 3. contract-state smart (num_tokens query for cw721)
    echo "  [3/3] contract-state smart (num_tokens)"
    out=$($BINARY q wasm contract-state smart "$contract_addr" '{"num_tokens":{}}' --output json --home $HOME 2>&1)
    if [ $? -ne 0 ]; then
        echo "  FAIL: contract-state smart {\"num_tokens\":{}} for $contract_addr"
        echo "  Error: $out"
        FAILURES=$((FAILURES + 1))
        SUMMARY+=("FAIL | $contract_addr | contract-state smart | $out")
    else
        data=$(echo "$out" | jq -r '.data // empty')
        if [ -z "$data" ]; then
            echo "  FAIL: contract-state smart {\"num_tokens\":{}} returned empty data for $contract_addr"
            FAILURES=$((FAILURES + 1))
            SUMMARY+=("FAIL | $contract_addr | contract-state smart | empty data")
        else
            echo "  OK: $(echo "$out" | jq -c '.data')"
            SUMMARY+=("OK   | $contract_addr | contract-state smart | $(echo "$out" | jq -c '.data')")
        fi
    fi
done

echo ""
echo "=== SUMMARY ==="
echo "-----+------------------------------------------------------------------+----------------------+-------------------"
printf "%-4s | %-64s | %-20s | %s\n" "STAT" "CONTRACT" "QUERY" "DETAIL"
echo "-----+------------------------------------------------------------------+----------------------+-------------------"
for line in "${SUMMARY[@]}"; do
    IFS='|' read -r stat contract query detail <<< "$line"
    printf "%-4s | %-64s | %-20s | %s\n" "$(echo $stat)" "$(echo $contract)" "$(echo $query)" "$(echo $detail)"
done
echo "-----+------------------------------------------------------------------+----------------------+-------------------"
echo "Contracts tested: ${#CONTRACTS[@]} | Total queries: ${#SUMMARY[@]} | Passed: $((${#SUMMARY[@]} - FAILURES)) | Failed: $FAILURES"

if [ $FAILURES -gt 0 ]; then
    echo "SOME QUERIES FAILED"
    exit 1
fi

echo "ALL QUERIES PASSED"
