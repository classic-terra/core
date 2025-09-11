# Return 0 if address is a Pair contract (responds to {"pair":{}}), non-zero otherwise
is_pair_address() {
    local binary_path=$1
    local addr=$2
    local ok=1
    local out=$(query_smart "$binary_path" "$addr" '{"pair":{}}' 2>/tmp/pair_detect_err || echo '{}')
    local lt=$(echo "$out" | jq -r '.query_result.liquidity_token // .data.liquidity_token // empty')
    if [ -n "$lt" ] && [ "$lt" != "null" ]; then
        ok=0
    fi
    return $ok
}

# If addr is an LP token, try to resolve its pair via {"minter":{}}
resolve_pair_from_lp() {
    local binary_path=$1
    local lp_addr=$2
    local out=$(query_smart "$binary_path" "$lp_addr" '{"minter":{}}' 2>/tmp/minter_err || echo '{}')
    local pa=$(echo "$out" | jq -r '.query_result.minter // .data.minter // empty')
    echo -n "$pa"
}

# Extract pair address from a factory pair query response in multiple shapes
extract_pair_from_factory_response() {
    local json=$1
    # Try common shapes
    echo "$json" | jq -r '(
        .query_result.contract_addr //
        .query_result.pair_address //
        .query_result.pair_info.contract_addr //
        .data.contract_addr //
        .data.pair_address //
        .data.pair_info.contract_addr //
        .contract_addr //
        .pair_address //
        ""
    )'
}
#!/bin/bash

SLEEP_SHORT=${SLEEP_SHORT:-3}
sleep_short() { sleep "$SLEEP_SHORT"; }
WAIT_TIMEOUT=${WAIT_TIMEOUT:-30}
WAIT_INTERVAL=${WAIT_INTERVAL:-1}

# Wait for a tx hash to be included in a block (portable across versions)
wait_for_tx() {
    local binary_path=$1
    local txhash=$2
    local waited=0
    local block_height=0
    local found=0
    while [ $waited -lt $WAIT_TIMEOUT ]; do
        echo "Waiting for tx $txhash to be included in a block" >&2
        local TXQ=$(${binary_path} q tx "$txhash" --output json 2>>/tmp/tx_query_error || echo '{}')
        block_height=$(echo "$TXQ" | jq -r '.height // .tx_response.height // ""')
        local CODE=$(echo "$TXQ" | jq -r '.code // .tx_response.code // empty')
        if [ -n "$block_height" ] && [ "$block_height" != "0" ]; then
            # If code present and non-zero, still return (caller can inspect)
            found=1
            break
        fi
        sleep "$WAIT_INTERVAL"
        waited=$((waited+WAIT_INTERVAL))
    done

    # wait for the next block
    while [ $waited -lt $WAIT_TIMEOUT ]; do
        found=0
        BLOCKQ=$(${binary_path} q block 2>>/tmp/tx_query_error || echo '{}')
        local HEIGHT=$(echo "$BLOCKQ" | jq -r '.block.header.height // ""')
        echo "Waiting for block $block_height, current height $HEIGHT" >&2
        if [ -n "$HEIGHT" ] && [ "$HEIGHT" != "0" ]; then
            if [ "$HEIGHT" -gt "$block_height" ]; then
                found=1
                break
            fi
        fi
        sleep "$WAIT_INTERVAL"
        waited=$((waited+WAIT_INTERVAL))
    done

    if [ $found -eq 0 ]; then
        return 1
    fi
    return 0
}

USE_FACTORY_CREATE=${USE_FACTORY_CREATE:-"true"}

# Create a pair via factory (modern form) and return pair address
# Uses init_params as an empty string (required by modern pairs 4007/4156)
create_pair_via_factory_modern() {
    local binary_path=$1
    local factory_addr=$2
    local denom_a=${3:-"uusd"}
    local denom_b=${4:-"uluna"}
    local wasm_file=${5:-"${XYK_PAIR_WASM}"}

    local CREATE_MSG
    CREATE_MSG='{"create_pair":{"pair_type":{"xyk":{}},"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"init_params":"eyJhbXAiOjIwMH0="}}'
    echo "Creating modern pair via factory with msg: $CREATE_MSG" >&2
    local TXH
    TXH=$(${binary_path} tx wasm execute "$factory_addr" "$CREATE_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y -o json 2>>/tmp/tx_exec_error | jq -r '.txhash // empty')
    echo "Factory create_pair txhash: $TXH" >&2
    if [ -n "$TXH" ]; then
        wait_for_tx "$binary_path" "$TXH"
    fi
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Factory create_pair failed." >&2
        exit 1
    fi
    # Try to resolve pair address by querying factory
    local Q_MSG='{"pair":{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}]}}'
    local ATT=0
    local PA=""
    while [ $ATT -lt 10 ] && { [ -z "$PA" ] || [ "$PA" = "null" ]; }; do
        local RES=$(query_smart "$binary_path" "$factory_addr" "$Q_MSG" 2>/dev/null || echo '{}')
        PA=$(extract_pair_from_factory_response "$RES")
        if [ -n "$PA" ] && [ "$PA" != "null" ]; then
            break
        fi
        ATT=$((ATT+1))
        sleep_short
    done
    PA=$(echo -n "$PA" | tr -d '\n\r\t ')
    # Validate that PA is a pair; if we accidentally got LP, derive pair via minter
    if [ -n "$PA" ]; then
        if ! is_pair_address "$binary_path" "$PA"; then
            # Attempt LP->pair using minter
            local PA2=$(resolve_pair_from_lp "$binary_path" "$PA")
            if [ -n "$PA2" ] && is_pair_address "$binary_path" "$PA2"; then
                PA="$PA2"
            fi
        fi
        echo "$PA" > ${HOME}/astroport_pair_address_$(basename "${wasm_file}").txt
    fi

    printf "%s" "$PA"
}

# Portable smart query wrapper supporting legacy and modern CLI syntaxes
query_smart() {
    local binary_path=$1
    local addr=$2
    local msg=$3
    # Prefer modern syntax if available
    if ${binary_path} q wasm --help 2>&1 | grep -q "contract-state"; then
        ${binary_path} q wasm contract-state smart "$addr" "$msg" --output json
    else
        # Legacy: terrad query wasm contract-store [bech32-address] [msg]
        ${binary_path} query wasm contract-store "$addr" "$msg" --output json
    fi
}

# Helper: store a wasm and return its code_id
store_wasm_and_get_code_id() {
    local binary_path=$1
    local wasm_file=$2
    TXH=$(${binary_path} tx wasm store "${wasm_file}" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json 2>>/tmp/tx_exec_error | jq -r '.txhash // empty')
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Store wasm failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
    STORE_CODE_ID=$(echo "$TXQ" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    echo "$STORE_CODE_ID"
}

# Upload and instantiate Astroport factory; returns factory address
upload_and_instantiate_astroport_factory() {
    local binary_path=$1
    local wasm_file=$2
    local lp_code_id=$3
    local xyk_pair_code_id=$4
    local owner_addr=$(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME})

    echo "Uploading Astroport factory wasm: ${wasm_file}" >&2
    TXH=$(${binary_path} tx wasm store "${wasm_file}" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json 2>>/tmp/tx_exec_error | jq -r '.txhash // empty')
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Store wasm failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
    FACT_CODE_ID=$(echo "$TXQ" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    echo "Factory code uploaded with code ID: $FACT_CODE_ID" >&2

    FACT_INIT_MSG=$(jq -n --arg owner "$owner_addr" --argjson token_code_id "$lp_code_id" --argjson xyk_code_id "$xyk_pair_code_id" '
      {
        owner: $owner,
        pair_configs: [
          { code_id: $xyk_code_id, pair_type: { xyk: {} }, total_fee_bps: 30, maker_fee_bps: 3333 }
        ],
        token_code_id: $token_code_id
      }
    ')

    echo "Instantiating factory with INIT_MSG: $FACT_INIT_MSG" >&2
    FACT_CMD=("${binary_path}" tx wasm instantiate "$FACT_CODE_ID" "$FACT_INIT_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        --admin $(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME}) \
        -y \
        --output json)
    if ${binary_path} tx wasm instantiate --help 2>&1 | grep -q -- "--label"; then
        FACT_CMD+=(--label "Astroport Factory")
    fi
    TXH=$("${FACT_CMD[@]}" 2>>/tmp/tx_exec_error | jq -r '.txhash // empty')
    echo "Factory instantiate txhash: ${TXH}" >&2

    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Factory instantiate failed: ${FACT_INIT_OUTPUT}"
        exit 1
    fi

    wait_for_tx "$binary_path" "$TXH"

    # Try multiple schemas to extract address
    TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
    FACTORY_ADDR=$(echo "$TXQ" | jq -r '[.logs[]? | .events[]? | .attributes[]? | select(.key=="_contract_address" or .key=="contract_address") | .value] | last // ""')
    if [ -z "$FACTORY_ADDR" ] || [ "$FACTORY_ADDR" = "null" ]; then
        if [ -n "$TXH" ] && [ "$TXH" != "null" ]; then
            sleep_short
            TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
            FACTORY_ADDR=$(echo "$TXQ" | jq -r '[.logs[]? | .events[]? | .attributes[]? | select(.key=="_contract_address" or .key=="contract_address") | .value] | last // ""')
            if [ -z "$FACTORY_ADDR" ] || [ "$FACTORY_ADDR" = "null" ]; then
                RAW=$(echo "$TXQ" | jq -r '.raw_log // empty')
                if [ -n "$RAW" ]; then
                    # Try to grep address pattern terra1...
                    FACTORY_ADDR=$(echo "$RAW" | grep -oE 'terra1[0-9a-z]{38,}' | head -n1)
                fi
            fi
        fi
    fi
    echo "Factory instantiated at: $FACTORY_ADDR" >&2
    echo "$FACTORY_ADDR" > ${HOME}/astroport_factory_address.txt
    sleep 3
    # Only echo the address (no other lines) so callers can capture cleanly
    printf "%s" "$FACTORY_ADDR"
}

# Support for multiple versions and upgrades
# OLD_VERSIONS and UPGRADE_NAMES must have the same length.
# Each element in OLD_VERSIONS represents a version to upgrade from,
# and the corresponding element in UPGRADE_NAMES is the upgrade name applied to that version.
# For example, OLD_VERSIONS[0] is upgraded using UPGRADE_NAMES[0], and so on.
#OLD_VERSIONS_STRING=${OLD_VERSIONS:-"v2.4.2,v3.0.4,v3.1.3,v3.1.5,v3.1.6,v3.3.0,v3.4.0,v3.4.3,v3.5.0"}
#UPGRADE_NAMES_STRING=${UPGRADE_NAMES:-"v8,v8_1,v8_2,v8_3,v10_1,v11_1,v11_2,v12,v13"}
OLD_VERSIONS_STRING=${OLD_VERSIONS:-"v1.1.0,v2.0.1,v2.1.1,v2.2.1,v2.3.0,v2.3.3,v2.4.2,v3.0.4,v3.1.3,v3.1.5,v3.1.6,v3.3.0,v3.4.0,v3.4.3,v3.5.0"}
UPGRADE_NAMES_STRING=${UPGRADE_NAMES:-"v3,v4,v5,v6,v6_1,v7,v8,v8_1,v8_2,v8_3,v10_1,v11_1,v11_2,v12,v13"}

# Parse comma-separated lists into arrays
IFS=',' read -r -a OLD_VERSIONS <<< "$OLD_VERSIONS_STRING"
IFS=',' read -r -a UPGRADE_NAMES <<< "$UPGRADE_NAMES_STRING"

# Map a NEXT_BINARY name (e.g., v5, new) to its numeric index for comparison
get_binary_index() {
    local name=$1
    if [[ "$name" == v* ]]; then
        echo "${name#v}"
    else
        # treat "new" (or others) as very high
        echo 9999
    fi
}

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
ASTROPORT_LP_WASM=${ASTROPORT_LP_WASM:-"./scripts/wasm/contracts/astroport-440-lp.wasm"}
ASTROPORT_POOL_WASM_GLOB=${ASTROPORT_POOL_WASM_GLOB:-"./scripts/wasm/contracts/astroport-*.wasm"}
ASTROPORT_FACTORY_WASM=${ASTROPORT_FACTORY_WASM:-"./scripts/wasm/contracts/astroport-4006-factory.wasm"}

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
        make build && cp build/terrad _build/$target_dir/terrad
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
    make build && cp build/terrad _build/new/terrad
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
                sleep_short
            else
                echo "$SCRIPT is not a file"
            fi
        done
    fi
}

run_upgrade () {
    local current_binary=$1
    local next_binary=$2
    local upgrade_name=$3
    local proposal_id=$4
    
    echo "Upgrading from $current_binary to $next_binary with upgrade name $upgrade_name"

    STATUS_INFO=($(./_build/$current_binary/terrad status --home $HOME | jq -r '.NodeInfo.network,.SyncInfo.latest_block_height'))
    UPGRADE_HEIGHT=$((STATUS_INFO[1] + 20))

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

    while [ true ] ; do
        # Submit the upgrade proposal
        if ./_build/$current_binary/terrad tx gov --help 2>&1 | grep -q "submit-legacy-proposal"; then
            CMD=( ./_build/$current_binary/terrad tx gov submit-legacy-proposal software-upgrade "$upgrade_name" \
                --upgrade-height "$UPGRADE_HEIGHT" \
                --upgrade-info "$UPGRADE_INFO" \
                --title "upgrade to $upgrade_name" \
                --description "upgrade to $upgrade_name" \
                --from test1 --keyring-backend test --chain-id "$CHAIN_ID" --home "$HOME" \
                --broadcast-mode sync \
                --gas-prices "$GAS_PRICE" -y )
        else
            CMD=( ./_build/$current_binary/terrad tx gov submit-proposal software-upgrade "$upgrade_name" \
                --upgrade-height "$UPGRADE_HEIGHT" \
                --upgrade-info "$UPGRADE_INFO" \
                --title "upgrade to $upgrade_name" \
                --description "upgrade to $upgrade_name" \
                --from test1 --keyring-backend test --chain-id "$CHAIN_ID" --home "$HOME" \
                --broadcast-mode sync \
                --gas-prices "$GAS_PRICE" -y )
        fi

        OUT=$("${CMD[@]}" --output json 2>>/tmp/tx_exec_error || echo '{}')
        TX_HASH=$(echo "$OUT" | jq -r '.txhash // empty')
        if [ -z "$TX_HASH" ] || [ "$TX_HASH" = "null" ]; then
            echo "Failed to submit proposal" >&2
            return 1
        fi
        wait_for_tx "./_build/$current_binary/terrad" "$TX_HASH"

        res=$(./_build/$current_binary/terrad q gov proposals --home $HOME --output json | jq -r '.proposals[] | select(.title == "upgrade to '$upgrade_name'") | .id')
        if [ -n "$res" ]; then
            echo "Found proposal id: $res"
            proposal_id=$res
            break
        fi

        if ./_build/$current_binary/terrad q gov proposal $proposal_id --home $HOME --output json >/dev/null 2>&1; then 
            echo "Proposal $proposal_id found"
            break
        fi

        sleep_short
        echo "CMD failed: ${CMD[@]}"
    done

    sleep_short

    # Deposit tokens for the proposal
    OUT=$(./_build/$current_binary/terrad tx gov deposit $proposal_id "20000000${DENOM}" --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE --output json -y 2>>/tmp/tx_exec_error)
    TX_HASH=$(echo "$OUT" | jq -r '.txhash // empty')
    if [ -z "$TX_HASH" ] || [ "$TX_HASH" = "null" ]; then
        echo "Deposit failed" >&2
        return 1
    fi
    wait_for_tx "./_build/$current_binary/terrad" "$TX_HASH"

    # Vote yes on the proposal
    OUT=$(./_build/$current_binary/terrad tx gov vote $proposal_id yes --from test0 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE --output json -y 2>>/tmp/tx_exec_error)
    TX_HASH=$(echo "$OUT" | jq -r '.txhash // empty')
    if [ -z "$TX_HASH" ] || [ "$TX_HASH" = "null" ]; then
        echo "Vote failed" >&2
        return 1
    fi
    wait_for_tx "./_build/$current_binary/terrad" "$TX_HASH"

    OUT=$(./_build/$current_binary/terrad tx gov vote $proposal_id yes --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE --output json -y 2>>/tmp/tx_exec_error)
    TX_HASH=$(echo "$OUT" | jq -r '.txhash // empty')
    if [ -z "$TX_HASH" ] || [ "$TX_HASH" = "null" ]; then
        echo "Vote failed" >&2
        return 1
    fi
    wait_for_tx "./_build/$current_binary/terrad" "$TX_HASH"

    # Wait for the upgrade height
    while true; do 
        BLOCK_HEIGHT=$(./_build/$current_binary/terrad status | jq '.SyncInfo.latest_block_height' -r)
        if [ $BLOCK_HEIGHT = "$UPGRADE_HEIGHT" ]; then
            # assuming running only 1 terrad
            echo "BLOCK HEIGHT = $UPGRADE_HEIGHT REACHED, KILLING CURRENT NODE"
            sleep 3
            pkill terrad
            sleep 3
            break
        else
            ./_build/$current_binary/terrad q gov proposal $proposal_id --output=json | jq ".status"
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            sleep_short
        fi
    done
}

# Run the first node with the old binary
run_node "_build/old/terrad" ""

# Function to upload LP code
upload_astroport_lp_code() {
    local binary_path=$1
    local wasm_file=$2

    echo "Uploading Astroport LP CW20 contract code"
    TXH=$(${binary_path} tx wasm store "${wasm_file}" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json 2>>/tmp/tx_exec_error | jq -r '.txhash')
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Astroport LP code upload failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    local TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
    LP_CODE_ID=$(echo "$TXQ" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    echo "Astroport LP code uploaded with code ID: $LP_CODE_ID"
    echo "$LP_CODE_ID" > ${HOME}/astroport_lp_code_id.txt
    sleep_short
}

upload_and_instantiate_astroport_pool() {
    local binary_path=$1
    local wasm_file=$2
    local lp_code_id=$3
    local denom_a=${4:-"uluna"}
    local denom_b=${5:-"uusd"}
    local factory_addr=${6:-""}

    echo "Uploading pool wasm: ${wasm_file}" >&2
    TXH=$(${binary_path} tx wasm store "${wasm_file}" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y \
        --output json 2>>/tmp/tx_exec_error | jq -r '.txhash')
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Pool code upload failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    local TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
    POOL_CODE_ID=$(echo "$TXQ" | jq -r '.logs[0].events[]? | select(.type=="store_code") | .attributes[]? | select(.key=="code_id") | .value')
    echo "Pool code uploaded with code ID: $POOL_CODE_ID" >&2

    # Determine if this wasm requires init_params (only 4007 and 4156)
    base_file=$(basename "$wasm_file")
    NEEDS_INIT_PARAMS=false
    if [[ "$base_file" == *"4007"* || "$base_file" == *"4156"* ]]; then
        NEEDS_INIT_PARAMS=true
    fi

    # Build a minimal set of candidates
    CANDIDATES=()
    if [ "$NEEDS_INIT_PARAMS" = true ]; then
        # Only for 4007 and 4156: pair_type xyk with init_params required by legacy schemas
        if [ -n "$factory_addr" ]; then
            CANDIDATES+=(
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"factory_addr":"'"${factory_addr}"'","pair_type":{"xyk":{}},"init_params":"eyJhbXAiOjIwMH0="}'
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"factory_addr":"'"${factory_addr}"'","pair_type":{"xyk":{}}}'
            )
        else
            CANDIDATES+=(
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"pair_type":{"xyk":{}},"init_params":"eyJhbXAiOjIwMH0="}'
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"pair_type":{"xyk":{}}}'
            )
        fi
    else
        # All other pools: xyk only, no init_params
        if [ -n "$factory_addr" ]; then
            CANDIDATES+=(
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"factory_addr":"'"${factory_addr}"'","pair_type":{"xyk":{}}}'
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"factory_addr":"'"${factory_addr}"'"}'
            )
        else
            CANDIDATES+=(
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"',"pair_type":{"xyk":{}}}'
                '{"asset_infos":[{"native_token":{"denom":"'"${denom_a}"'"}},{"native_token":{"denom":"'"${denom_b}"'"}}],"token_code_id":'"${lp_code_id}"'}'
            )
        fi
    fi
    INIT_OUTPUT=""
    for raw in "${CANDIDATES[@]}"; do
        INIT_MSG_C=$(echo "$raw" | jq -c . 2>/dev/null || echo "$raw")
        echo "Instantiating pool with INIT_MSG: $INIT_MSG_C" >&2
        CMD=("${binary_path}" tx wasm instantiate "$POOL_CODE_ID" "$INIT_MSG_C" \
            --from test1 \
            --chain-id ${CHAIN_ID} \
            --gas auto \
            --gas-adjustment 1.3 \
            --gas-prices ${GAS_PRICE} \
            --broadcast-mode sync \
            --keyring-backend test \
            --home ${HOME} \
            --admin $(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME}) \
            -y \
            --output json)
        if ${binary_path} tx wasm instantiate --help 2>&1 | grep -q -- "--label"; then
            CMD+=(--label "Astroport Pool ${denom_a}-${denom_b}")
        fi
        if INIT_OUTPUT=$("${CMD[@]}"); then
            # success
            sleep 2
            break
        else
            echo "Instantiate attempt failed, trying next schema..." >&2
        fi
    done
    # Robust address extraction with retries (logs -> q tx -> list-contract-by-code)
    # Ensure INIT_OUTPUT looks like JSON before parsing
    if ! echo "$INIT_OUTPUT" | jq -e . >/dev/null 2>&1; then
        echo "Instantiate did not return JSON output; skipping address extraction." >&2
        printf "%s" ""
        return 0
    fi
    TXH=$(echo "$INIT_OUTPUT" | jq -r '.txhash // empty')
    echo "Pair instantiate txhash: ${TXH} for wasm $(basename "$wasm_file")" >&2
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Pair instantiate failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    PAIR_ADDR=""
    ATTEMPTS=0
    while [ $ATTEMPTS -lt 8 ] && { [ -z "$PAIR_ADDR" ] || [ "$PAIR_ADDR" = "null" ]; }; do
        TXQ=$(${binary_path} q tx "$TXH" -o json 2>/dev/null || echo '{}')
        CAND=$(echo "$TXQ" | jq -r '[.logs[]? | .events[]? | .attributes[]? | select(.key=="_contract_address" or .key=="contract_address") | .value] | last // ""')
        if [ -n "$CAND" ] && [ "$CAND" != "null" ]; then
            PAIR_ADDR="$CAND"
            break
        fi
        if [ -n "$TXH" ] && [ "$TXH" != "null" ]; then
            RAW=$(echo "$TXQ" | jq -r '.raw_log // empty')
            if [ -n "$RAW" ]; then
                CAND=$(echo "$RAW" | grep -oE 'terra1[0-9a-z]{38,}' | head -n1)
            fi
            if [ -n "$CAND" ] && [ "$CAND" != "null" ]; then
                PAIR_ADDR="$CAND"
                break
            fi
        fi
        ATTEMPTS=$((ATTEMPTS+1))
        sleep_short
    done
    PAIR_ADDR=$(echo -n "$PAIR_ADDR" | tr -d '\n\r\t ')
    echo "Pool instantiated at: $PAIR_ADDR" >&2
    # Return only the address on stdout for callers using command substitution
    printf "%s" "$PAIR_ADDR"
    if [ -n "$PAIR_ADDR" ]; then
        echo "$PAIR_ADDR" > ${HOME}/astroport_pair_address_$(basename "${wasm_file}").txt
    fi
    sleep_short
}

test_astroport_pool() {
    local binary_path=$1
    local pair_addr=$2
    local denom_a=${3:-"uusd"}
    local denom_b=${4:-"uluna"}

    TEST1_ADDR=$(${binary_path} keys show test1 -a --keyring-backend test --home ${HOME})

    echo "Providing initial liquidity to $pair_addr"
    PROVIDE_MSG='{"provide_liquidity":{"assets":[{"info":{"native_token":{"denom":"'"${denom_b}"'"}},"amount":"3733100000"},{"info":{"native_token":{"denom":"'"${denom_a}"'"}},"amount":"10222500000"}]}}'
    COINS_PROVIDE="10222500000${denom_a},3733100000${denom_b}"
    CMD=("${binary_path}" tx wasm execute "$pair_addr" "$PROVIDE_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y)
    if ${binary_path} tx wasm execute --help 2>&1 | grep -q -- "--amount"; then
        CMD+=(--amount "$COINS_PROVIDE")
    else
        CMD+=("$COINS_PROVIDE")
    fi
    ATT=0; TXH=""
    while [ $ATT -lt 5 ]; do
        OUT=$("${CMD[@]}" --output json 2>>/tmp/tx_exec_error || true)
        TXH=$(echo "$OUT" | jq -r '.txhash // empty' 2>/dev/null)
        if [ -n "$TXH" ] && [ "$TXH" != "null" ]; then
            break
        fi
        echo "provide_liquidity attempt $ATT failed" >&2
        echo "CMD: ${CMD[@]}" >> /tmp/tx_exec_error
        echo "OUT: $OUT" >> /tmp/tx_exec_error
        sleep_short
        ATT=$((ATT+1))
    done
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Failed to provide liquidity after retries; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    ${binary_path} q tx "$TXH" -o json 2>/dev/null >> /tmp/execution_logs

    echo "Swap ${denom_a} -> ${denom_b}"
    SWAP_AB_MSG='{"swap":{"max_spread":"0.1","offer_asset":{"info":{"native_token":{"denom":"'"${denom_a}"'"}},"amount":"100000"}}}'
    COINS_SWAP_AB="100000${denom_a}"
    CMD=("${binary_path}" tx wasm execute "$pair_addr" "$SWAP_AB_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y)
    if ${binary_path} tx wasm execute --help 2>&1 | grep -q -- "--amount"; then
        CMD+=(--amount "$COINS_SWAP_AB")
    else
        CMD+=("$COINS_SWAP_AB")
    fi
    OUT=$("${CMD[@]}" --output json 2>>/tmp/tx_exec_error || true)
    TXH=$(echo "$OUT" | jq -r '.txhash // empty' 2>/dev/null)
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Swap A->B failed; aborting." >&2
    else
        wait_for_tx "$binary_path" "$TXH"
        ${binary_path} q tx "$TXH" -o json 2>/dev/null >> /tmp/execution_logs
    fi

    echo "Swap ${denom_b} -> ${denom_a}"
    SWAP_BA_MSG='{"swap":{"max_spread":"0.1","offer_asset":{"info":{"native_token":{"denom":"'"${denom_b}"'"}},"amount":"50000"}}}'
    COINS_SWAP_BA="50000${denom_b}"
    CMD=("${binary_path}" tx wasm execute "$pair_addr" "$SWAP_BA_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y)
    if ${binary_path} tx wasm execute --help 2>&1 | grep -q -- "--amount"; then
        CMD+=(--amount "$COINS_SWAP_BA")
    else
        CMD+=("$COINS_SWAP_BA")
    fi
    OUT=$("${CMD[@]}" --output json 2>>/tmp/tx_exec_error || true)
    TXH=$(echo "$OUT" | jq -r '.txhash // empty' 2>/dev/null)
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Swap B->A failed; aborting." >&2
    else
        wait_for_tx "$binary_path" "$TXH"
        ${binary_path} q tx "$TXH" -o json 2>/dev/null > /tmp/execution_logs
    fi

    echo "Discover LP token address"
    echo "Contract info for pair $pair_addr:" >&2
    ${binary_path} q wasm contract "$pair_addr" -o json | jq . || true
    echo "Querying {\"pair\":{}}" >&2
    if ! PAIR_INFO=$(query_smart "$binary_path" "$pair_addr" '{"pair":{}}' 2> /tmp/pair_err.txt); then
        echo "pair query error" >&2
        PAIR_INFO='{}'
    fi


    LP_TOKEN_ADDR=$(echo "$PAIR_INFO" | jq -r '.query_result.liquidity_token // .data.liquidity_token // empty')
    if [ -z "$LP_TOKEN_ADDR" ] || [ "$LP_TOKEN_ADDR" = "null" ]; then
        echo "Unable to discover LP token address via {\"pair\":{}}; querying {\"pool\":{}} for info..."
        if ! POOL_INFO=$(query_smart "$binary_path" "$pair_addr" '{"pool":{}}' 2> /tmp/pool_err.txt); then
            echo "pool query error" >&2
            POOL_INFO='{}'
        fi
        echo "Pool info:"
        echo "$POOL_INFO" | jq .
        echo "Skipping withdrawal test for $pair_addr"
        return 0
    fi
    echo "LP token: $LP_TOKEN_ADDR"

    BAL_Q='{"balance":{"address":"'"$TEST1_ADDR"'"}}'
    BAL_JSON=$(query_smart "${binary_path}" "$LP_TOKEN_ADDR" "$BAL_Q")
    LP_BAL=$(echo "$BAL_JSON" | jq -r '.query_result.balance // .data.balance // "0"')
    echo "LP balance: $LP_BAL"
    if [ "$LP_BAL" = "0" ]; then
        echo "No LP balance to withdraw"
        return 0
    fi
    WITHDRAW_AMT=$((LP_BAL / 2))
    BASE64_WITHDRAW=$(echo -n '{"withdraw_liquidity":{}}' | base64 | tr -d '\n')
    SEND_MSG='{"send":{"msg":"'"$BASE64_WITHDRAW"'","amount":"'"$WITHDRAW_AMT"'","contract":"'"$pair_addr"'"}}'
    echo "Withdrawing liquidity amount: $WITHDRAW_AMT"
    OUT=$(${binary_path} tx wasm execute $LP_TOKEN_ADDR "$SEND_MSG" \
        --from test1 \
        --chain-id ${CHAIN_ID} \
        --gas auto \
        --gas-adjustment 1.3 \
        --gas-prices ${GAS_PRICE} \
        --broadcast-mode sync \
        --keyring-backend test \
        --home ${HOME} \
        -y --output json 2>>/tmp/tx_exec_error || true)
    TXH=$(echo "$OUT" | jq -r '.txhash // empty' 2>/dev/null)
    if [ -z "$TXH" ] || [ "$TXH" = "null" ]; then
        echo "Withdraw liquidity failed; aborting." >&2
        return 1
    fi
    wait_for_tx "$binary_path" "$TXH"
    ${binary_path} q tx "$TXH" -o json 2>/dev/null >> /tmp/execution_logs
}

deploy_and_test_all_pools() {
    local binary_path=$1
    local denom_a=${2:-"uluna"}
    local denom_b=${3:-"uusd"}

    if [ -f ${HOME}/astroport_lp_code_id.txt ]; then
        LP_CODE_ID=$(cat ${HOME}/astroport_lp_code_id.txt)
    else
        upload_astroport_lp_code "$binary_path" "$ASTROPORT_LP_WASM"
        LP_CODE_ID=$(cat ${HOME}/astroport_lp_code_id.txt)
    fi

    # Pre-store a preferred pair code (4007 preferred, else 4156) to feed factory init
    FACTORY_PAIR_CODE_ID=""
    for wasm in ${ASTROPORT_POOL_WASM_GLOB}; do
        base=$(basename "$wasm")
        if [[ "$base" == "$(basename "$ASTROPORT_LP_WASM")" ]]; then
            continue
        fi
        if [[ "$base" == *"4007"* ]]; then
            # Reuse stored code id if available
            if [[ -f "${HOME}/codeid_${base}.txt" ]]; then
                FACTORY_PAIR_CODE_ID=$(cat "${HOME}/codeid_${base}.txt")
            else
                FACTORY_PAIR_CODE_ID=$(store_wasm_and_get_code_id "$binary_path" "$wasm")
                echo -n "$FACTORY_PAIR_CODE_ID" > "${HOME}/codeid_${base}.txt"
            fi
            break
        fi
    done
    if [[ -z "$FACTORY_PAIR_CODE_ID" ]]; then
        for wasm in ${ASTROPORT_POOL_WASM_GLOB}; do
            base=$(basename "$wasm")
            if [[ "$base" == "$(basename "$ASTROPORT_LP_WASM")" ]]; then
                continue
            fi
            if [[ "$base" == *"4156"* ]]; then
                if [[ -f "${HOME}/codeid_${base}.txt" ]]; then
                    FACTORY_PAIR_CODE_ID=$(cat "${HOME}/codeid_${base}.txt")
                else
                    FACTORY_PAIR_CODE_ID=$(store_wasm_and_get_code_id "$binary_path" "$wasm")
                    echo -n "$FACTORY_PAIR_CODE_ID" > "${HOME}/codeid_${base}.txt"
                fi
                break
            fi
        done
    fi

    FACTORY_ADDR=""
    # Reuse existing global factory if present
    if [[ -f "${HOME}/astroport_factory_address.txt" ]]; then
        FACTORY_ADDR=$(cat "${HOME}/astroport_factory_address.txt" | tr -d '\n\r\t ')
        echo "Reusing existing Astroport factory at: $FACTORY_ADDR"
    elif [ -n "$FACTORY_PAIR_CODE_ID" ] && [ -f "$ASTROPORT_FACTORY_WASM" ]; then
        echo "Deploying Astroport factory using pair code id: $FACTORY_PAIR_CODE_ID"
        FACTORY_ADDR=$(upload_and_instantiate_astroport_factory "$binary_path" "$ASTROPORT_FACTORY_WASM" "$LP_CODE_ID" "$FACTORY_PAIR_CODE_ID")
        FACTORY_ADDR=$(echo -n "$FACTORY_ADDR" | sed -e 's/^"//' -e 's/"$//' | tr -d '\n\r\t ')
        echo "Factory deployed at: $FACTORY_ADDR"
    else
        echo "Factory wasm not available or no suitable pair code id found; will instantiate pairs without factory_addr where allowed."
    fi

    for wasm in ${ASTROPORT_POOL_WASM_GLOB}; do
        if [[ "$(basename "$wasm")" == "$(basename "$ASTROPORT_LP_WASM")" ]]; then
            continue
        fi
        if [[ -n "$ASTROPORT_FACTORY_WASM" && "$(basename "$wasm")" == "$(basename "$ASTROPORT_FACTORY_WASM")" ]]; then
            # Skip the factory wasm; it is not a pool to instantiate/test
            continue
        fi
        if [[ ! -f "$wasm" ]]; then
            continue
        fi
        echo "=== Deploying and testing pool: $wasm ==="
        base_file=$(basename "$wasm")
        PAIR_ADDR=""
        # First, reuse previously created pair if present; validate it's a pair (not LP)
        if [[ -f "${HOME}/astroport_pair_address_${base_file}.txt" ]]; then
            CACHED=$(cat "${HOME}/astroport_pair_address_${base_file}.txt" | tr -d '\n\r\t ')
            if [ -n "$CACHED" ] && is_pair_address "$binary_path" "$CACHED"; then
                PAIR_ADDR="$CACHED"
                echo "Reusing existing pair for $base_file at: $PAIR_ADDR"
            else
                # Try to resolve from LP via minter
                PAIR_ADDR=$(resolve_pair_from_lp "$binary_path" "$CACHED")
                if [ -n "$PAIR_ADDR" ] && is_pair_address "$binary_path" "$PAIR_ADDR"; then
                    echo "Corrected cached address for $base_file to pair: $PAIR_ADDR"
                    echo "$PAIR_ADDR" > "${HOME}/astroport_pair_address_${base_file}.txt"
                else
                    PAIR_ADDR=""
                fi
            fi
        elif [[ "$base_file" == *"4007"* || "$base_file" == *"4156"* ]]; then
            # Modern pairs: create a dedicated factory bound to this pair code id (reused if exists)
            if [[ -f "${HOME}/codeid_${base_file}.txt" ]]; then
                PAIR_CODE_ID=$(cat "${HOME}/codeid_${base_file}.txt")
            else
                PAIR_CODE_ID=$(store_wasm_and_get_code_id "$binary_path" "$wasm")
                echo -n "$PAIR_CODE_ID" > "${HOME}/codeid_${base_file}.txt"
            fi
            if [[ ! -f "$ASTROPORT_FACTORY_WASM" ]]; then
                echo "Factory wasm not found but required for modern pair $base_file; skipping." >&2
                continue
            fi
            # Dedicated factory per modern code id
            if [[ -f "${HOME}/astroport_modern_factory_${base_file}.txt" ]]; then
                DEDICATED_FACTORY=$(cat "${HOME}/astroport_modern_factory_${base_file}.txt" | tr -d '\n\r\t ')
                echo "Reusing dedicated factory for $base_file at: $DEDICATED_FACTORY"
            else
                DEDICATED_FACTORY=$(upload_and_instantiate_astroport_factory "$binary_path" "$ASTROPORT_FACTORY_WASM" "$LP_CODE_ID" "$PAIR_CODE_ID")
                DEDICATED_FACTORY=$(echo -n "$DEDICATED_FACTORY" | tr -d '\n\r\t ')
                echo "$DEDICATED_FACTORY" > "${HOME}/astroport_modern_factory_${base_file}.txt"
                echo "Dedicated factory for $base_file at: $DEDICATED_FACTORY"
            fi
            PAIR_ADDR=$(create_pair_via_factory_modern "$binary_path" "$DEDICATED_FACTORY" "$denom_a" "$denom_b" "$wasm" | grep -oE 'terra1[0-9a-z]{38,}' | head -n1)
            # Ensure we didn't capture LP by mistake
            if [ -n "$PAIR_ADDR" ] && ! is_pair_address "$binary_path" "$PAIR_ADDR"; then
                FIX=$(resolve_pair_from_lp "$binary_path" "$PAIR_ADDR")
                if [ -n "$FIX" ] && is_pair_address "$binary_path" "$FIX"; then
                    PAIR_ADDR="$FIX"
                    echo "$PAIR_ADDR" > "${HOME}/astroport_pair_address_${base_file}.txt"
                fi
            fi
        else
            # Legacy pairs: instantiate directly; include global factory_addr if present
            if [[ -n "$FACTORY_ADDR" ]]; then
                PAIR_ADDR=$(upload_and_instantiate_astroport_pool "$binary_path" "$wasm" "$LP_CODE_ID" "$denom_a" "$denom_b" "$FACTORY_ADDR" | grep -oE 'terra1[0-9a-z]{38,}' | head -n1)
            else
                PAIR_ADDR=$(upload_and_instantiate_astroport_pool "$binary_path" "$wasm" "$LP_CODE_ID" "$denom_a" "$denom_b" | grep -oE 'terra1[0-9a-z]{38,}' | head -n1)
            fi
            # Ensure we didn't capture LP by mistake
            if [ -n "$PAIR_ADDR" ] && ! is_pair_address "$binary_path" "$PAIR_ADDR"; then
                FIX=$(resolve_pair_from_lp "$binary_path" "$PAIR_ADDR")
                if [ -n "$FIX" ] && is_pair_address "$binary_path" "$FIX"; then
                    PAIR_ADDR="$FIX"
                    echo "$PAIR_ADDR" > "${HOME}/astroport_pair_address_${base_file}.txt"
                fi
            fi
        fi
        # Verify that PAIR_ADDR is actually a contract by querying contract info
        if [[ -n "$PAIR_ADDR" ]]; then
            CONTRACT_INFO=$(${binary_path} q wasm contract "$PAIR_ADDR" -o json 2>/dev/null || echo '')
            # Support both shapes: {address,code_id} and {contract_info:{address,code_id,...}}
            CODE_ID_OK=$(echo "$CONTRACT_INFO" | jq -r '.code_id // .contract_info.code_id // empty')
            if [[ -n "$CODE_ID_OK" ]]; then
                echo "Pair deployed at: $PAIR_ADDR (code_id=$CODE_ID_OK)"
            else
                echo "Address $PAIR_ADDR did not return valid contract info; skipping tests for $(basename "$wasm"): $CONTRACT_INFO." >&2
                continue
            fi
        else
            echo "Empty address for $(basename "$wasm"); skipping." >&2
            continue
        fi
        # Validate address before testing
        if [[ -z "$PAIR_ADDR" || ! "$PAIR_ADDR" =~ ^terra1[0-9a-z]+$ ]]; then
            echo "Skipping tests for $(basename "$wasm"): invalid/empty pair address ('$PAIR_ADDR')." >&2
            continue
        fi
        sleep_short
        test_astroport_pool "$binary_path" "$PAIR_ADDR" "$denom_a" "$denom_b"
    done
}

# Execute pre-upgrade scripts
execute_scripts "$ADDITIONAL_PRE_SCRIPTS"

# Do NOT deploy CW20 yet; it requires >= v2.3.3. Will deploy after reaching threshold.

# Upload LP CW20 code needed by Astroport pools (store once, usable across upgrades)
upload_astroport_lp_code "_build/old/terrad" "${ASTROPORT_LP_WASM}"

# Deploy and test Astroport pools
echo "Deploying and testing Astroport pools after first upgrade..."
deploy_and_test_all_pools "_build/old/terrad" "uluna" "uusd"

# Main upgrade sequence
# Loop through all versions and upgrades
for ((i=0; i<${#OLD_VERSIONS[@]}; i++)); do
    # Skip the first version as it's already running
    if [ $i -gt 0 ]; then
        echo "Proceeding to upgrade ${i} of ${#UPGRADE_NAMES[@]}"
        sleep_short
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

        deploy_and_test_all_pools "_build/new/terrad" "uluna" "uusd"
    else
        # For intermediate upgrades, run with the next version
        run_node "_build/$NEXT_BINARY/terrad" "true"

        deploy_and_test_all_pools "_build/$NEXT_BINARY/terrad" "uluna" "uusd"
    fi
done

# Execute post-upgrade scripts
execute_scripts "$ADDITIONAL_AFTER_SCRIPTS"
