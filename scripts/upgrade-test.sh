#!/bin/bash

# the upgrade is a fork, "true" otherwise
FORK=${FORK:-"false"}

# $(curl --silent "https://api.github.com/repos/classic-terra/core/releases/latest" | jq -r '.tag_name')

OLD_VERSION=${OLD_VERSION:-v4.0.0}
HOME=mytestnet
ROOT=$(pwd)
DENOM=uluna
CHAIN_ID=localterra
SOFTWARE_UPGRADE_NAME=${SOFTWARE_UPGRADE_NAME:-"v14_2"}
ADDITIONAL_PRE_SCRIPTS=${ADDITIONAL_PRE_SCRIPTS:-""}
ADDITIONAL_AFTER_SCRIPTS=${ADDITIONAL_AFTER_SCRIPTS:-""}
GAS_PRICE=${GAS_PRICE:-"30uluna"}
FORK_HALT_HEIGHT=${FORK_HALT_HEIGHT:-""}

if [[ "$FORK" == "true" ]]; then
    if [ -n "$FORK_HALT_HEIGHT" ]; then
        export TERRAD_HALT_HEIGHT="$FORK_HALT_HEIGHT"
    elif [ -n "$ADDITIONAL_PRE_SCRIPTS" ]; then
        # Allow pre-upgrade scripts (e.g. wasm deploy) enough blocks before halting.
        export TERRAD_HALT_HEIGHT=200
    else
        export TERRAD_HALT_HEIGHT=20
    fi
    echo "FORK mode: TERRAD_HALT_HEIGHT=$TERRAD_HALT_HEIGHT"
fi

# underscore so that go tool will not take gocache into account
mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

# install old source if not exist
if [ ! -f "_build/$OLD_VERSION.zip" ]; then
    wget -c "https://github.com/classic-terra/core/archive/refs/tags/${OLD_VERSION}.zip" -O _build/${OLD_VERSION}.zip
fi

if [ ! -d "_build/core-${OLD_VERSION:1}" ]; then
    unzip -o _build/${OLD_VERSION}.zip -d _build
fi

# reinstall old binary
OLD_BUILD_MARKER="_build/old/.built-from-version"
REINSTALL_OLD="false"
if [ $# -eq 1 ] && [ "$1" == "--reinstall-old" ]; then
    REINSTALL_OLD="true"
fi

if [[ "$REINSTALL_OLD" == "true" ]] || [ ! -x "_build/old/terrad" ] || [ ! -f "$OLD_BUILD_MARKER" ] || [ "$(cat "$OLD_BUILD_MARKER" 2>/dev/null)" != "$OLD_VERSION" ]; then
    mkdir -p _build/old
    cd ./_build/core-${OLD_VERSION:1}
    GOBIN="$ROOT/_build/old" go install -mod=readonly ./...
    cd ../..
    echo "$OLD_VERSION" > "$OLD_BUILD_MARKER"
fi

# install new binary
if [ ! -x "_build/new/terrad" ]; then
    mkdir -p ./_build/new
    GOBIN="$ROOT/_build/new" go install -mod=readonly ./...
    cd ../..
fi

if [[ "$FORK" != "true" ]]; then
    if grep -Rqs "const UpgradeName = \"$SOFTWARE_UPGRADE_NAME\"" "./_build/core-${OLD_VERSION:1}/app/upgrades"; then
        echo "OLD_VERSION=$OLD_VERSION already includes upgrade handler $SOFTWARE_UPGRADE_NAME"
        echo "choose an older OLD_VERSION (without this handler) or a different SOFTWARE_UPGRADE_NAME"
        exit 1
    fi
fi

# run old node
if [[ "$OSTYPE" == "darwin"* ]]; then
    screen -L -dmS node1 bash scripts/run-node.sh _build/old/terrad $DENOM
else
    screen -L -Logfile $HOME/log-screen.txt -dmS node1 bash scripts/run-node.sh _build/old/terrad $DENOM
fi

sleep 20

# execute additional pre scripts
if [ ! -z "$ADDITIONAL_PRE_SCRIPTS" ]; then
    # slice ADDITIONAL_SCRIPTS by ,
    SCRIPTS=($(echo "$ADDITIONAL_PRE_SCRIPTS" | tr ',' ' '))
    for SCRIPT in "${SCRIPTS[@]}"; do
         # check if SCRIPT is a file
        if [ -f "$SCRIPT" ]; then
            echo "executing additional pre scripts from $SCRIPT"
            source $SCRIPT
            sleep 5
        else
            echo "$SCRIPT is not a file"
        fi
    done
fi

run_fork () {
    echo "forking"

    LAST_BLOCK_HEIGHT=""
    STALLED_ROUNDS=0
    while true; do
        BLOCK_HEIGHT=$(./_build/old/terrad status --home $HOME | jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height')
        if [ ! -z "$BLOCK_HEIGHT" ] && [ "$BLOCK_HEIGHT" != "null" ]; then
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            if [ "$BLOCK_HEIGHT" == "$LAST_BLOCK_HEIGHT" ]; then
                STALLED_ROUNDS=$((STALLED_ROUNDS + 1))
            else
                STALLED_ROUNDS=0
                LAST_BLOCK_HEIGHT="$BLOCK_HEIGHT"
            fi
            if [ "$STALLED_ROUNDS" -ge 3 ]; then
                echo "Block height stalled at $BLOCK_HEIGHT, node halted. Forking."
                pkill terrad
                break
            fi
            sleep 10
        else
            echo "BLOCK_HEIGHT is empty, forking"
            break
        fi
    done
}

run_upgrade () {
    echo "upgrading"

    CURRENT_HEIGHT=$(./_build/old/terrad status --home $HOME | jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height')
    if [ -z "$CURRENT_HEIGHT" ] || [ "$CURRENT_HEIGHT" == "null" ]; then
        echo "could not fetch latest block height from terrad status"
        exit 1
    fi
    UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 100))
    echo "UPGRADE_HEIGHT = $UPGRADE_HEIGHT"

    tar -cf ./_build/new/terrad.tar -C ./_build/new terrad
    SUM=$(shasum -a 256 ./_build/new/terrad.tar | cut -d ' ' -f1)
    UPGRADE_INFO=$(jq -n '
    {
        "binaries": {
            "linux/amd64": "file://'$(pwd)'/_build/new/terrad.tar?checksum=sha256:'"$SUM"'",
        }
    }')

    ./_build/old/terrad keys list --home $HOME --keyring-backend test

    # Get the gov module authority address
    GOV_AUTHORITY=$(./_build/old/terrad q auth module-account gov --home $HOME --output json | jq -r '.account.value.address // .account.base_account.address // .account.address')

    # Create the upgrade proposal JSON file
    cat > $HOME/upgrade_proposal.json <<EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "$GOV_AUTHORITY",
      "plan": {
        "name": "$SOFTWARE_UPGRADE_NAME",
        "height": "$UPGRADE_HEIGHT",
        "info": $(echo "$UPGRADE_INFO" | jq -c '. | tostring')
      }
    }
  ],
  "deposit": "20000000${DENOM}",
  "title": "upgrade to $SOFTWARE_UPGRADE_NAME",
  "summary": "upgrade to $SOFTWARE_UPGRADE_NAME",
  "metadata": ""
}
EOF

    ./_build/old/terrad tx gov submit-proposal $HOME/upgrade_proposal.json --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE --gas auto --gas-adjustment 1.5 -y

    sleep 2

    # Deposit is included in the proposal JSON, but add more if needed
    ./_build/old/terrad tx gov deposit 1 "20000000${DENOM}" --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    ./_build/old/terrad tx gov vote 1 yes --from test0 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    ./_build/old/terrad tx gov vote 1 yes --from test1 --keyring-backend test --chain-id $CHAIN_ID --home $HOME --gas-prices $GAS_PRICE -y

    sleep 2

    # determine block_height to halt
    LAST_BLOCK_HEIGHT=""
    STALLED_ROUNDS=0
    while true; do 
        BLOCK_HEIGHT=$(./_build/old/terrad status --home $HOME | jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height')

        if [ -z "$BLOCK_HEIGHT" ] || [ "$BLOCK_HEIGHT" == "null" ]; then
            echo "failed to fetch block height from old node"
            exit 1
        fi

        if [ "$BLOCK_HEIGHT" == "$LAST_BLOCK_HEIGHT" ]; then
            STALLED_ROUNDS=$((STALLED_ROUNDS + 1))
        else
            STALLED_ROUNDS=0
            LAST_BLOCK_HEIGHT="$BLOCK_HEIGHT"
        fi

        if [ "$STALLED_ROUNDS" -ge 6 ]; then
            echo "block height stalled at $BLOCK_HEIGHT; old node may have halted or crashed"
            exit 1
        fi

        if [ "$BLOCK_HEIGHT" = "$UPGRADE_HEIGHT" ]; then
            # assuming running only 1 terrad
            echo "BLOCK HEIGHT = $UPGRADE_HEIGHT REACHED, KILLING OLD ONE"
            pkill terrad
            break
        else
            PROPOSAL_STATUS=$(./_build/old/terrad q gov proposal 1 --output=json | jq -r '.status // .proposal.status // "UNKNOWN"')
            echo "$PROPOSAL_STATUS"
            if [ "$PROPOSAL_STATUS" = "PROPOSAL_STATUS_FAILED" ] || [ "$PROPOSAL_STATUS" = "PROPOSAL_STATUS_REJECTED" ]; then
                echo "upgrade proposal is not passable in this run"
                exit 1
            fi
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            sleep 10
        fi
    done
}

# if FORK = true
if [[ "$FORK" == "true" ]]; then
    run_fork
    unset TERRAD_HALT_HEIGHT
else
    run_upgrade
fi

sleep 5

# run new node
if [[ "$OSTYPE" == "darwin"* ]]; then
    CONTINUE="true" screen -L -dmS node1 bash scripts/run-node.sh _build/new/terrad $DENOM
else
    CONTINUE="true" screen -L -Logfile $HOME/log-screen.txt -dmS node1 bash scripts/run-node.sh _build/new/terrad $DENOM
fi

echo "Waiting for new node to be ready..."
for i in $(seq 1 60); do
    if ./_build/new/terrad status --home $HOME > /dev/null 2>&1; then
        echo "New node is ready (attempt $i)"
        break
    fi
    if [ $i -eq 60 ]; then
        echo "New node failed to start within 60 attempts. Check $HOME/log-screen.txt"
        exit 1
    fi
    sleep 5
done

# execute additional after scripts
if [ ! -z "$ADDITIONAL_AFTER_SCRIPTS" ]; then
    # slice ADDITIONAL_SCRIPTS by ,
    SCRIPTS=($(echo "$ADDITIONAL_AFTER_SCRIPTS" | tr ',' ' '))
    for SCRIPT in "${SCRIPTS[@]}"; do
         # check if SCRIPT is a file
        if [ -f "$SCRIPT" ]; then
            echo "executing additional after scripts from $SCRIPT"
            source $SCRIPT
            sleep 5
        else
            echo "$SCRIPT is not a file"
        fi
    done
fi