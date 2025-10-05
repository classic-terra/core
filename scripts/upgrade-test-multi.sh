#!/bin/bash

# the upgrade is a fork, "true" otherwise
FORK=${FORK:-"false"}

# Support for multiple versions and upgrades
# OLD_VERSIONS and UPGRADE_NAMES must have the same length.
# Each element in OLD_VERSIONS represents a version to upgrade from,
# and the corresponding element in UPGRADE_NAMES is the upgrade name applied to that version.
# For example, OLD_VERSIONS[0] is upgraded using UPGRADE_NAMES[0], and so on.
OLD_VERSIONS_STRING=${OLD_VERSIONS:-"v2.4.2,v3.0.4,v3.1.3,v3.1.5,v3.1.6,v3.3.0,v3.4.0,v3.4.3,v3.5.0,v3.6.0-rc.0"}
UPGRADE_NAMES_STRING=${UPGRADE_NAMES:-"v8,v8_1,v8_2,v8_3,v10_1,v11_1,v11_2,v12,v13,v14"}

# Parse comma-separated lists into arrays
IFS=',' read -r -a OLD_VERSIONS <<< "$OLD_VERSIONS_STRING"
IFS=',' read -r -a UPGRADE_NAMES <<< "$UPGRADE_NAMES_STRING"

# Validate that both arrays have the same length
if [ ${#OLD_VERSIONS[@]} -ne ${#UPGRADE_NAMES[@]} ]; then
    echo "Error: The number of OLD_VERSIONS (${#OLD_VERSIONS[@]}) must match the number of UPGRADE_NAMES (${#UPGRADE_NAMES[@]})"
    exit 1
fi

# First version is the starting point
CURRENT_VERSION=${OLD_VERSIONS[0]}

UPGRADE_WAIT=${UPGRADE_WAIT:-10}
HOME=mytestnet
ROOT=$(pwd)
DENOM=uluna
CHAIN_ID=localterra-legacy
ADDITIONAL_PRE_SCRIPTS=${ADDITIONAL_PRE_SCRIPTS:-""}
ADDITIONAL_AFTER_SCRIPTS=${ADDITIONAL_AFTER_SCRIPTS:-""}
GAS_PRICE=${GAS_PRICE:-"30uluna"}
CW20_TOKEN_WASM=${CW20_TOKEN_WASM:-"./scripts/cw20_token.wasm"}

if [[ "$FORK" == "true" ]]; then
    export TERRAD_HALT_HEIGHT=20
fi

# underscore so that go tool will not take gocache into account
mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

# Function to install a specific version
install_version() {
    local version=$1
    local target_dir=$2
    local reinstall_flag=$3
    
    # Download and extract if not exist
    if [ ! -f "_build/$version.zip" ]; then
        mkdir -p _build/$target_dir
        wget -c "https://github.com/classic-terra/core/archive/refs/tags/${version}.zip" -O _build/${version}.zip
        unzip _build/${version}.zip -d _build
    fi
    
    # Install the binary
    if [ "$reinstall_flag" == "--reinstall" ] || ! command -v _build/$target_dir/terrad &> /dev/null; then
        cd ./_build/core-${version:1}
        GOBIN="$ROOT/_build/$target_dir" go install -mod=readonly ./...
        cd ../..
    fi
}

# Install all required versions
for ((i=0; i<${#OLD_VERSIONS[@]}; i++)); do
    # For the first version, install as "old"
    if [ $i -eq 0 ]; then
        install_version "${OLD_VERSIONS[$i]}" "old" $1
    else
        # For intermediate versions, install in version-specific directories
        install_version "${OLD_VERSIONS[$i]}" "v$i" $1
    fi
done

# Install the current version as "new"
if ! command -v _build/new/terrad &> /dev/null; then
    mkdir -p ./_build/new
    GOBIN="$ROOT/_build/new" go install -mod=readonly ./...
fi

# Function to run a node with a specific binary
run_node() {
    local binary_path=$1
    local continue_flag=$2
    
    echo "Starting node with binary: $binary_path"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        CONTINUE="$continue_flag" screen -L -dmS node1 bash scripts/run-node-legacy.sh $binary_path $DENOM
    else
        CONTINUE="$continue_flag" screen -L -Logfile $HOME/log-screen.txt -dmS node1 bash scripts/run-node-legacy.sh $binary_path $DENOM
    fi
    
    sleep 10
}

# Function to execute additional scripts
execute_scripts() {
    local scripts_list=$1
    
    if [ ! -z "$scripts_list" ]; then
        # slice scripts by ,
        SCRIPTS=($(echo "$scripts_list" | tr ',' ' '))
        for SCRIPT in "${SCRIPTS[@]}"; do
             # check if SCRIPT is a file
            if [ -f "$SCRIPT" ]; then
                echo "executing scripts from $SCRIPT"
                source $SCRIPT
                sleep 2
            else
                echo "$SCRIPT is not a file"
            fi
        done
    fi
}

run_fork () {
    echo "forking"

    while true; do 
        BLOCK_HEIGHT=$(./_build/old/terrad status | jq '.SyncInfo.latest_block_height' -r)
        # if BLOCK_HEIGHT is not empty
        if [ ! -z "$BLOCK_HEIGHT" ]; then
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            sleep 2
        else
            echo "BLOCK_HEIGHT is empty, forking"
            break
        fi
    done
}

run_upgrade () {
    local current_binary=$1
    local next_binary=$2
    local upgrade_name=$3
    local proposal_id=$4
    
    echo "Upgrading from $current_binary to $next_binary with upgrade name $upgrade_name"

    STATUS_INFO=($(./_build/$current_binary/terrad status --home $HOME | jq -r '.NodeInfo.network,.SyncInfo.latest_block_height'))
    UPGRADE_HEIGHT=$((STATUS_INFO[1] + 20))
    if [ $UPGRADE_HEIGHT -lt 35 ]; then
        UPGRADE_HEIGHT=35
    fi

    # Create the upgrade package for the next binary
    tar -cf ./_build/$next_binary/terrad.tar -C ./_build/$next_binary terrad
    SUM=$(shasum -a 256 ./_build/$next_binary/terrad.tar | cut -d ' ' -f1)
    UPGRADE_INFO=$(jq -n '
    {
        "binaries": {
            "linux/amd64": "file://'$(pwd)'/_build/'$next_binary'/terrad.tar?checksum=sha256:'"$SUM"'",
        }
    }')

    ./_build/$current_binary/terrad keys list --home $HOME --keyring-backend test

    # Submit the upgrade proposal
    ./_build/$current_binary/terrad tx gov submit-legacy-proposal software-upgrade "$upgrade_name" --upgrade-height $UPGRADE_HEIGHT --upgrade-info "$UPGRADE_INFO" --title "upgrade to $upgrade_name" --description "upgrade to $upgrade_name"  --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    # Deposit tokens for the proposal
    ./_build/$current_binary/terrad tx gov deposit $proposal_id "20000000${DENOM}" --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    # Vote yes on the proposal
    ./_build/$current_binary/terrad tx gov vote $proposal_id yes --from test0 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    ./_build/$current_binary/terrad tx gov vote $proposal_id yes --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    # Wait for the upgrade height
    while true; do 
        BLOCK_HEIGHT=$(./_build/$current_binary/terrad status | jq '.SyncInfo.latest_block_height' -r)
        if [ $BLOCK_HEIGHT = "$UPGRADE_HEIGHT" ]; then
            # assuming running only 1 terrad
            echo "BLOCK HEIGHT = $UPGRADE_HEIGHT REACHED, KILLING CURRENT NODE"
            pkill terrad
            sleep 5
            break
        else
            ./_build/$current_binary/terrad q gov proposal $proposal_id --output=json | jq ".status"
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            sleep 2
        fi
    done
}

# Run the first node with the old binary
run_node "_build/old/terrad" ""

# Function to upload and instantiate CW20 token contract
upload_and_instantiate_contract() {
    local binary_path=$1
    local wasm_file=$2
    
    echo "Uploading and instantiating CW20 token contract"
    
    # Upload the contract
    STORE_OUTPUT=$(${binary_path} tx wasm store "${wasm_file}" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode block \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json)
    
    # Extract code ID
    CODE_ID=$(echo $STORE_OUTPUT | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    echo "Contract uploaded with code ID: $CODE_ID"
    
    # Prepare instantiate message
    INIT_MSG='{"name":"Test Token","symbol":"TEST","decimals":6,"initial_balances":[{"address":"'$(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME})'","amount":"1000000000"}],"mint":{"minter":"'$(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME})'"},"marketing":null}'
    
    # Instantiate the contract
    INIT_OUTPUT=$(${binary_path} tx wasm instantiate $CODE_ID "$INIT_MSG" \
        --from test1 \
        --label "Test CW20 Token" \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode block \
        --keyring-backend test \
        --home ${HOME} \
        --admin $(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME}) \
        -y \
        --output json)
    
    # Extract contract address
    CONTRACT_ADDR=$(echo $INIT_OUTPUT | jq -r '.logs[0].events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value')
    echo "Contract instantiated at address: $CONTRACT_ADDR"
    
    # Save contract address to a file for later use
    echo "$CONTRACT_ADDR" > ${HOME}/cw20_contract_address.txt
    
    sleep 2
}

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

    echo -e "\n======== TESTS COMPLETED ========\n"
}

# Function to execute a CW20 transfer
execute_cw20_transfer() {
    local binary_path=$1
    
    echo "Executing CW20 token transfer after upgrade"
    
    # Read the contract address from the file
    CONTRACT_ADDR=$(cat ${HOME}/cw20_contract_address.txt)
    
    # Create test2 account if it doesn't exist
    if ! ${binary_path} keys show test2 --keyring-backend test --home ${HOME} &> /dev/null; then
        ${binary_path} keys add test2 --keyring-backend test --home ${HOME}
    fi
    
    # Get test2 address
    TEST2_ADDR=$(${binary_path} keys show test2 -a --keyring-backend test --home ${HOME})
    
    # Prepare transfer message
    TRANSFER_MSG='{"transfer":{"recipient":"'$TEST2_ADDR'","amount":"500000"}}'
    
    # Execute the transfer
    TRANSFER_OUTPUT=$(${binary_path} tx wasm execute $CONTRACT_ADDR "$TRANSFER_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json)
    
    # Check if transfer was successful
    TX_HASH=$(echo $TRANSFER_OUTPUT | jq -r '.txhash')
    echo "CW20 transfer executed with txhash: $TX_HASH"
    
    # Wait for transaction to be included in a block
    echo "Waiting for transaction to be included in a block..."
    sleep 2
    
    # Query the balance of test2 to verify the transfer
    BALANCE_MSG='{"balance":{"address":"'$TEST2_ADDR'"}}'
    BALANCE_QUERY=$(${binary_path} query wasm contract-state smart $CONTRACT_ADDR "$BALANCE_MSG" --output json)
    BALANCE=$(echo $BALANCE_QUERY | jq -r '.data.balance')
    
    echo "Test2 account balance after transfer: $BALANCE"
    sleep 2
}

# Execute pre-upgrade scripts
execute_scripts "$ADDITIONAL_PRE_SCRIPTS"

# Upload and instantiate CW20 token contract before the first upgrade
upload_and_instantiate_contract "_build/old/terrad" "${CW20_TOKEN_WASM}"

# Main upgrade sequence
if [[ "$FORK" == "true" ]]; then
    run_fork
    unset TERRAD_HALT_HEIGHT
else
    # Loop through all versions and upgrades
    for ((i=0; i<${#OLD_VERSIONS[@]}; i++)); do
        # Skip the first version as it's already running
        if [ $i -gt 0 ]; then
            echo "Proceeding to upgrade ${i} of ${#UPGRADE_NAMES[@]}"
            sleep 2
        fi
        
        # Determine current and next binary paths
        if [ $i -eq 0 ]; then
            CURRENT_BINARY="old"
        else
            # For intermediate versions, use v1, v2, etc. (not v0)
            CURRENT_BINARY="v$i"
        fi
        
        # Determine the next binary
        if [ $i -eq $((${#OLD_VERSIONS[@]}-1)) ]; then
            # Last upgrade uses the "new" binary (current codebase)
            NEXT_BINARY="new"
        else
            # Next binary is the next version in the sequence (i+1)
            NEXT_BINARY="v$((i+1))"
        fi
        
        # Run the upgrade with the appropriate proposal ID
        # Each upgrade gets a new proposal ID (i+1)
        run_upgrade "$CURRENT_BINARY" "$NEXT_BINARY" "${UPGRADE_NAMES[$i]}" "$((i+1))"
        
        # Start the next node after upgrade
        if [ $i -eq $((${#OLD_VERSIONS[@]}-1)) ]; then
            # For the final upgrade, run with the new binary
            run_node "_build/new/terrad" "true"
        else
            # For intermediate upgrades, run with the next version
            run_node "_build/$NEXT_BINARY/terrad" "true"
            
            # After the first upgrade, execute a CW20 transfer and run tests
            if [ $i -eq 0 ]; then
                echo "First upgrade completed, executing CW20 transfer..."
                execute_cw20_transfer "_build/$NEXT_BINARY/terrad"
                
                # Run tests after first upgrade to show historic height query issues
                echo -e "\n======== RUNNING TESTS AFTER FIRST UPGRADE (EXPECT SOME ERRORS) ========\n"
                echo "These tests should show errors with historic height queries that will be fixed in the final upgrade"
                run_final_tests "_build/$NEXT_BINARY/terrad" "35"
            fi
        fi
    done
fi

# Execute post-upgrade scripts
execute_scripts "$ADDITIONAL_AFTER_SCRIPTS"

# Run final tests after all upgrades
run_final_tests "_build/new/terrad" "35"
