#!/bin/bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$SCRIPT_DIR/old_terrad_cache.sh"

# this bash will prepare cosmosvisor to the build folder so that it can perform upgrade
# this script is supposed to be run by Makefile

# Every rehearsal must identify both the immutable predecessor and registered plan.
: "${OLD_VERSION:?Set OLD_VERSION explicitly}"
: "${OLD_REVISION:?Set OLD_REVISION to an immutable 40-character commit}"
: "${SOFTWARE_UPGRADE_NAME:?Set SOFTWARE_UPGRADE_NAME explicitly}"
[[ "$OLD_REVISION" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid OLD_REVISION' >&2; exit 1; }
[[ "$SOFTWARE_UPGRADE_NAME" =~ ^[a-zA-Z0-9_-]+$ ]] || exit 1
source "$SCRIPT_DIR/upgrade_assertions.sh"
BUILDDIR=${1:-}
TESTNET_NVAL=${2:-}
TESTNET_CHAINID=${3:-}

# check if BUILDDIR is set
if [ -z "$BUILDDIR" ]; then
    echo "BUILDDIR is not set"
    exit 1
fi

[[ "$TESTNET_NVAL" == 7 && -n "$TESTNET_CHAINID" ]] || { echo 'Seven validators and chain ID required' >&2; exit 1; }
[[ ! -f "$BUILDDIR/node0/terrad/config/genesis.json" ]] || { echo 'Upgrade rehearsal requires fresh node genesis' >&2; exit 1; }

# install old version of terrad

OLD_SOURCE="_build/core-${OLD_REVISION}"
if [ ! -d "$OLD_SOURCE" ]; then
    mkdir -p _build
    curl --fail --location --retry 3 "https://github.com/classic-terra/core/archive/${OLD_REVISION}.zip" -o "_build/${OLD_REVISION}.zip"
    unzip "_build/${OLD_REVISION}.zip" -d _build
fi

## Reuse the cached binary only if its requested source version and contents
## match the metadata written after a successful build. Legacy/unverifiable
## cache entries are rebuilt rather than trusted.
OLD_BINARY="$BUILDDIR/old/terrad"
OLD_BINARY_STAMP="$BUILDDIR/old/terrad.version"
if ! old_terrad_cache_valid "$OLD_BINARY" "$OLD_BINARY_STAMP" "$OLD_VERSION" "$OLD_REVISION"; then
    mkdir -p "$BUILDDIR/old"
    rm -f "$OLD_BINARY" "$OLD_BINARY_STAMP"
    OLD_VERSION_NUMBER=${OLD_VERSION#v}
    docker build --platform linux/amd64 --no-cache \
        --build-arg "source=./${OLD_SOURCE}/" \
        --build-arg "GIT_VERSION=$OLD_VERSION_NUMBER" \
        --build-arg "GIT_COMMIT=$OLD_REVISION" \
        --tag classic-terra/terraclassic.terrad-binary.old \
        -f contrib/updates/Dockerfile.old .
    docker create --platform linux/amd64 --name old-temp classic-terra/terraclassic.terrad-binary.old:latest
    docker cp old-temp:/usr/local/bin/terrad "$OLD_BINARY"
    docker rm old-temp
    OLD_BINARY_SHA256=$(sha256sum "$OLD_BINARY" | awk '{print $1}')
    printf 'version=%s\nrevision=%s\nsha256=%s\n' "$OLD_VERSION" "$OLD_REVISION" "$OLD_BINARY_SHA256" > "$OLD_BINARY_STAMP"
fi

# prepare cosmovisor config in TESTNET_NVAL nodes
if [ ! -f "$BUILDDIR/node0/terrad/config/genesis.json" ]; then docker run --rm \
    --user $(id -u):$(id -g) \
    -v "$BUILDDIR:/terrad:Z" \
    -v /etc/group:/etc/group:ro \
    -v /etc/passwd:/etc/passwd:ro \
    -v /etc/shadow:/etc/shadow:ro \
    --entrypoint /terrad/old/terrad \
    --platform linux/amd64 \
    classic-terra/terrad-upgrade-env testnet --v "$TESTNET_NVAL" --chain-id "$TESTNET_CHAINID" -o . --starting-ip-address 192.168.10.2 --keyring-backend=test --home=temp; \
fi

# Seed one funded validator with USTC for pre-upgrade donation transactions.
NODE1_ADDRESS=$(docker run --rm --user "$(id -u):$(id -g)" -v "$BUILDDIR:/terrad:Z" --entrypoint /terrad/old/terrad --platform linux/amd64 classic-terra/terrad-upgrade-env keys show node1 -a --keyring-backend test --home /terrad/node1/terrad)
for (( i=0; i<$TESTNET_NVAL; i++ )); do
    CURRENT="$BUILDDIR/node$i/terrad"

    prepare_upgrade_genesis "$CURRENT/config/genesis.json" "$NODE1_ADDRESS"

    docker run --rm \
        --user $(id -u):$(id -g) \
        -v "$BUILDDIR:/terrad:Z" \
        -v /etc/group:/etc/group:ro \
        -v /etc/passwd:/etc/passwd:ro \
        -v /etc/shadow:/etc/shadow:ro \
        -e DAEMON_HOME=/terrad/node$i/terrad \
        -e DAEMON_NAME=terrad \
        -e DAEMON_RESTART_AFTER_UPGRADE=true \
        --entrypoint /terrad/cosmovisor \
        --platform linux/amd64 \
        classic-terra/terrad-upgrade-env init /terrad/old/terrad
    mkdir -p "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin"
    cp "$BUILDDIR/terrad" "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin"

done
