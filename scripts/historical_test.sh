#!/bin/bash
# Contract address: terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au
# Test1 address: terra1dea2kw98rq3lce6tp8e9ae0rh893cj0hdkja29
# Test2 address: terra14eg2hvlmxt4d4w99n58ctrkcfkzm5p4dvrdgmz
CONTRACT_ADDR=terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au
TEST1_ADDR=terra1dea2kw98rq3lce6tp8e9ae0rh893cj0hdkja29
TEST2_ADDR=terra14eg2hvlmxt4d4w99n58ctrkcfkzm5p4dvrdgmz
HOME=mytestnet
# Function to run final tests after all upgrades
run_final_tests() {
    local binary_path=$1
    local historic_height=$2
    
    echo -e "\n======== RUNNING FINAL TESTS ========\n"
    
    # Get validator address
    VALIDATOR_ADDR=$(${binary_path} q staking validators -o json | jq -r '.validators[0].operator_address')
    echo "Validator address: $VALIDATOR_ADDR"
    
    echo -e "\n======== STAKING PARAMS TESTS ========\n"
    echo -e "\n--- Current height staking params ---"
    ${binary_path} q staking params --output json | jq
    
    echo -e "\n--- Historic height staking params (height $historic_height) ---"
    ${binary_path} q staking params --height $historic_height --output json | jq
    
    echo -e "\n======== STAKING DELEGATIONS TESTS ========\n"
    echo -e "\n--- Current height delegations to validator ---"
    ${binary_path} q staking delegations-to $VALIDATOR_ADDR --output json | jq
    
    echo -e "\n--- Historic height delegations to validator (height $historic_height) ---"
    ${binary_path} q staking delegations-to $VALIDATOR_ADDR --height $historic_height --output json | jq
    
    # Read the contract address from the file
    CONTRACT_ADDR=$(cat ${HOME}/cw20_contract_address.txt)
    
    # Get test1 and test2 addresses
    TEST1_ADDR=$(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME})
    TEST2_ADDR=$(${binary_path} keys show test2 -a --keyring-backend test --home ${HOME})
    
    echo -e "\n======== WASM CONTRACT STATE TESTS ========\n"
    echo "Contract address: $CONTRACT_ADDR"
    echo "Test1 address: $TEST1_ADDR"
    echo "Test2 address: $TEST2_ADDR"
    
    echo -e "\n--- Current height test1 balance ---"
    BALANCE_MSG='{"balance":{"address":"'$TEST1_ADDR'"}}'
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$BALANCE_MSG" --output json | jq
    
    echo -e "\n--- Historic height test1 balance (height $historic_height) ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$BALANCE_MSG" --height $historic_height --output json | jq
    
    echo -e "\n--- Current height test2 balance ---"
    BALANCE_MSG='{"balance":{"address":"'$TEST2_ADDR'"}}'
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$BALANCE_MSG" --output json | jq
    
    echo -e "\n--- Historic height test2 balance (height $historic_height) ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$BALANCE_MSG" --height $historic_height --output json | jq

    echo -e "\n--- Current height contract info ---"
    ${binary_path} q wasm contract $CONTRACT_ADDR --output json | jq

    echo -e "\n--- Historic height contract info (height $historic_height) ---"
    ${binary_path} q wasm contract $CONTRACT_ADDR --height $historic_height --output json | jq

    echo -e "\n--- Current height full contract state dump ---"
    ${binary_path} q wasm contract-state all $CONTRACT_ADDR --output json | jq

    echo -e "\n--- Historic height full contract state dump (height $historic_height) ---"
    ${binary_path} q wasm contract-state all $CONTRACT_ADDR --height $historic_height --output json | jq


    echo -e "\n======== WASM CODE QUERIES ========\n"
    CODE_ID=$(${binary_path} q wasm contract $CONTRACT_ADDR -o json | jq -r '.contract_info.code_id')
    echo "Code ID: $CODE_ID"

    echo -e "\n--- Current height code info ---"
    ${binary_path} q wasm code-info $CODE_ID --output json | jq

    echo -e "\n--- Historic height code info (height $historic_height) ---"
    ${binary_path} q wasm code-info $CODE_ID --height $historic_height --output json | jq

    echo -e "\n--- Current height contract info ---"
    ${binary_path} q wasm contract $CONTRACT_ADDR --output json | jq

    echo -e "\n--- Historic height contract info (height $historic_height) ---"
    ${binary_path} q wasm contract $CONTRACT_ADDR --height $historic_height --output json | jq


    echo -e "\n--- List all codes on chain ---"
    ${binary_path} q wasm list-code --output json | jq

    echo -e "\n--- List all contracts by this code ID ---"
    ${binary_path} q wasm list-contract-by-code $CODE_ID --output json | jq

    echo -e "\n--- Pinned codes in VM cache ---"
    ${binary_path} q wasm pinned --output json | jq

    echo -e "\n======== CW20-SPECIFIC QUERIES ========\n"

    TOKEN_INFO_MSG='{"token_info":{}}'
    echo -e "\n--- Token info ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$TOKEN_INFO_MSG" --output json | jq

    MINTER_MSG='{"minter":{}}'
    echo -e "\n--- Minter ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$MINTER_MSG" --output json | jq

    ALL_ACCOUNTS_MSG='{"all_accounts":{}}'
    echo -e "\n--- All accounts (current height) ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$ALL_ACCOUNTS_MSG" --output json | jq

    echo -e "\n--- All accounts (historic height $historic_height) ---"
    ${binary_path} q wasm contract-state smart $CONTRACT_ADDR "$ALL_ACCOUNTS_MSG" --height $historic_height --output json | jq

    echo -e "\n======== TESTS COMPLETED ========\n"
}


run_final_tests "_build/new/terrad" "10"
