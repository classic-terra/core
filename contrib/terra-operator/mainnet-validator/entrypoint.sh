#!/bin/sh

CONTINUE=${CONTINUE:-false}

if [ "$CONTINUE" = "true" ]; then
    echo "Continuing from previous run"
    terrad start --home ~/.terra
    exit 0
fi

# Copy genesis.json and addrbook.json to config
cp /terra/columbus-5-genesis.json /terra/.terra/config/genesis.json
cp /terra/addrbook.json /terra/.terra/config/addrbook.json

# Download snapshot from quicksync.io
URL=$(curl -L https://quicksync.io/terra.json|jq -r '.[] |select(.file=="columbus-5-pruned")|select (.mirror=="Netherlands")|.url')

# TODO: need a better way to check for columbus-5-pruned* else it will always download newest snapshot
if [ ! -f /terra/data/columbus-5-pruned* ]; then 
    echo "Downloading snapshot from $URL"
    aria2c -x5 $URL -d /terra/data
    echo "Download complete"
fi
DATA_FILE=$(find /terra/data -type f -name 'columbus-5-pruned*' | grep '\.tar\.lz4$')
rm -rf /terra/.terra/data
echo "Extracting snapshot from $DATA_FILE"
lz4 -dc $DATA_FILE | tar -xf - -C /terra/.terra
echo "Extraction complete"

terrad start --home ~/.terra