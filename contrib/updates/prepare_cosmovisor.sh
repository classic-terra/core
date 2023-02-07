#!/bin/bash

# this bash will prepare cosmosvisor to the build folder so that it can perform upgrade
# this script is supposed to be run by Makefile

OLD_VERSION=1.0.5
SOFTWARE_UPGRADE_NAME=v2
BUILDDIR=$1
CURDIR=$2

echo $CURDIR

# check if BUILDDIR is set
if [ -z "$BUILDDIR" ]; then
    echo "BUILDDIR is not set"
    exit 1
fi

# check if CURDIR is set
if [ -z "$CURDIR" ]; then
    echo "CURDIR is not set"
    exit 1
fi

# install old version of terrad

## check if _build/classic-${OLD_VERSION} exists
if [ ! -d "$CURDIR/_build/core-${OLD_VERSION}" ]; then
    mkdir $CURDIR/_build
    wget -c "https://github.com/classic-terra/core/archive/refs/tags/v${OLD_VERSION}.zip" -O $CURDIR/_build/v${OLD_VERSION}.zip
    unzip $CURDIR/_build/v${OLD_VERSION}.zip -d $CURDIR/_build
    ls -l $CURDIR/_build
fi

## check if $BUILDDIR/old/terrad exists
if [ ! -f "$BUILDDIR/old/terrad" ]; then
    mkdir -p $BUILDDIR/old
    docker build --platform linux/amd64 --no-cache --build-arg source=/$CURDIR/_build/core-${OLD_VERSION} --tag classic-terra/terraclassic.terrad-binary.old $CURDIR
    docker create --platform linux/amd64 --name old-temp classic-terra/terraclassic.terrad-binary.old:latest
    docker cp old-temp:/usr/local/bin/terrad $BUILDDIR/old/
    docker rm old-temp
fi

# prepare cosmovisor config in four nodes
if [ ! -f "$BUILDDIR/node0/terrad/config/genesis.json" ]; then docker run --rm \
    --user $(id -u):$(id -g) \
    -v $BUILDDIR:/terrad:Z \
    -v /etc/group:/etc/group:ro \
    -v /etc/passwd:/etc/passwd:ro \
    -v /etc/shadow:/etc/shadow:ro \
    --entrypoint /terrad/old/terrad \
    classic-terra/terrad-upgrade-env testnet --v 4 -o . --starting-ip-address 192.168.10.2 --keyring-backend=test --home /terrad; 
fi

for i in {0..3}; do
    CURRENT=$BUILDDIR/node$i/terrad

    # change gov params voting_period
    jq '.app_state.gov.voting_params.voting_period = "50s"' $CURRENT/config/genesis.json > $CURRENT/config/genesis.json.tmp && mv $CURRENT/config/genesis.json.tmp $CURRENT/config/genesis.json

    docker run --rm \
        --user $(id -u):$(id -g) \
        -v $BUILDDIR:/terrad:Z \
        -v /etc/group:/etc/group:ro \
        -v /etc/passwd:/etc/passwd:ro \
        -v /etc/shadow:/etc/shadow:ro \
        -e DAEMON_HOME=/terrad/node$i/terrad \
        -e DAEMON_NAME=terrad \
        -e DAEMON_RESTART_AFTER_UPGRADE=true \
        --entrypoint /terrad/cosmovisor \
        classic-terra/terrad-upgrade-env init /terrad/old/terrad
    mkdir -p $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin
    cp $BUILDDIR/terrad $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin
    touch $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/upgrade-info.json
done

