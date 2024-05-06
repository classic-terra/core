#!/bin/bash
set -xeu

# always returns true so set -e doesn't exit if it is not running.
killall terrad || true
rm -rf $HOME/.terrad/

# make four terra directories
mkdir $HOME/.terrad
cd $HOME/.terrad/
mkdir $HOME/.terrad/validator1
mkdir $HOME/.terrad/validator2
mkdir $HOME/.terrad/validator3

# init all three validators
terrad init --chain-id=testing validator1 --home=$HOME/.terrad/validator1
terrad init --chain-id=testing validator2 --home=$HOME/.terrad/validator2
terrad init --chain-id=testing validator3 --home=$HOME/.terrad/validator3

# create keys for all three validators
terrad keys add validator1 --keyring-backend=test --home=$HOME/.terrad/validator1
terrad keys add validator2 --keyring-backend=test --home=$HOME/.terrad/validator2
terrad keys add validator3 --keyring-backend=test --home=$HOME/.terrad/validator3

# create validator node with tokens to transfer to the three other nodes
terrad add-genesis-account $(terrad keys show validator1 -a --keyring-backend=test --home=$HOME/.terrad/validator1) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator1
terrad add-genesis-account $(terrad keys show validator2 -a --keyring-backend=test --home=$HOME/.terrad/validator2) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator1
terrad add-genesis-account $(terrad keys show validator3 -a --keyring-backend=test --home=$HOME/.terrad/validator3) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator1
terrad add-genesis-account $(terrad keys show validator1 -a --keyring-backend=test --home=$HOME/.terrad/validator1) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator2
terrad add-genesis-account $(terrad keys show validator2 -a --keyring-backend=test --home=$HOME/.terrad/validator2) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator2
terrad add-genesis-account $(terrad keys show validator3 -a --keyring-backend=test --home=$HOME/.terrad/validator3) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator2
terrad add-genesis-account $(terrad keys show validator1 -a --keyring-backend=test --home=$HOME/.terrad/validator1) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator3
terrad add-genesis-account $(terrad keys show validator2 -a --keyring-backend=test --home=$HOME/.terrad/validator2) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator3
terrad add-genesis-account $(terrad keys show validator3 -a --keyring-backend=test --home=$HOME/.terrad/validator3) 10000000000000000000000000000000uluna --home=$HOME/.terrad/validator3
terrad gentx validator1 100000000000000000uluna --keyring-backend=test --home=$HOME/.terrad/validator1 --chain-id=testing
terrad gentx validator2 100000000000000000uluna --keyring-backend=test --home=$HOME/.terrad/validator2 --chain-id=testing
terrad gentx validator3 100000000000000000uluna --keyring-backend=test --home=$HOME/.terrad/validator3 --chain-id=testing

cp validator2/config/gentx/*.json $HOME/.terrad/validator1/config/gentx/
cp validator3/config/gentx/*.json $HOME/.terrad/validator1/config/gentx/
terrad collect-gentxs --home=$HOME/.terrad/validator1 

cp validator1/config/genesis.json $HOME/.terrad/validator2/config/genesis.json
cp validator1/config/genesis.json $HOME/.terrad/validator3/config/genesis.json


# change app.toml values
VALIDATOR1_APP_TOML=$HOME/.terrad/validator1/config/app.toml
VALIDATOR2_APP_TOML=$HOME/.terrad/validator2/config/app.toml
VALIDATOR3_APP_TOML=$HOME/.terrad/validator3/config/app.toml

# validator1
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9050|g' $VALIDATOR1_APP_TOML

# validator2
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1316|g' $VALIDATOR2_APP_TOML
# sed -i -E 's|0.0.0.0:9090|0.0.0.0:9088|g' $VALIDATOR2_APP_TOML
sed -i -E 's|localhost:9090|localhost:9088|g' $VALIDATOR2_APP_TOML
# sed -i -E 's|0.0.0.0:9091|0.0.0.0:9089|g' $VALIDATOR2_APP_TOML
sed -i -E 's|localhost:9091|localhost:9089|g' $VALIDATOR2_APP_TOML
sed -i -E 's|tcp://0.0.0.0:10337|tcp://0.0.0.0:10347|g' $VALIDATOR2_APP_TOML

# validator3
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1315|g' $VALIDATOR3_APP_TOML
# sed -i -E 's|0.0.0.0:9090|0.0.0.0:9086|g' $VALIDATOR3_APP_TOML
sed -i -E 's|localhost:9090|localhost:9086|g' $VALIDATOR3_APP_TOML
# sed -i -E 's|0.0.0.0:9091|0.0.0.0:9087|g' $VALIDATOR3_APP_TOML
sed -i -E 's|localhost:9091|localhost:9087|g' $VALIDATOR3_APP_TOML
sed -i -E 's|tcp://0.0.0.0:10337|tcp://0.0.0.0:10357|g' $VALIDATOR3_APP_TOML

# change config.toml values
VALIDATOR1_CONFIG=$HOME/.terrad/validator1/config/config.toml
VALIDATOR2_CONFIG=$HOME/.terrad/validator2/config/config.toml
VALIDATOR3_CONFIG=$HOME/.terrad/validator3/config/config.toml


# validator1
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR1_CONFIG
sed -i -E 's|prometheus = false|prometheus = true|g' $VALIDATOR1_CONFIG


# validator2
sed -i -E 's|tcp://127.0.0.1:26658|tcp://127.0.0.1:26655|g' $VALIDATOR2_CONFIG
sed -i -E 's|tcp://127.0.0.1:26657|tcp://127.0.0.1:26654|g' $VALIDATOR2_CONFIG
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26653|g' $VALIDATOR2_CONFIG
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR2_CONFIG
sed -i -E 's|prometheus = false|prometheus = true|g' $VALIDATOR2_CONFIG
sed -i -E 's|prometheus_listen_addr = ":26660"|prometheus_listen_addr = ":26630"|g' $VALIDATOR2_CONFIG

# validator3
sed -i -E 's|tcp://127.0.0.1:26658|tcp://127.0.0.1:26652|g' $VALIDATOR3_CONFIG
sed -i -E 's|tcp://127.0.0.1:26657|tcp://127.0.0.1:26651|g' $VALIDATOR3_CONFIG
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26650|g' $VALIDATOR3_CONFIG
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR3_CONFIG
sed -i -E 's|prometheus = false|prometheus = true|g' $VALIDATOR3_CONFIG
sed -i -E 's|prometheus_listen_addr = ":26660"|prometheus_listen_addr = ":26620"|g' $VALIDATOR3_CONFIG


# copy validator1 genesis file to validator2-3
cp $HOME/.terrad/validator1/config/genesis.json $HOME/.terrad/validator2/config/genesis.json
cp $HOME/.terrad/validator1/config/genesis.json $HOME/.terrad/validator3/config/genesis.json

# copy tendermint node id of validator1 to persistent peers of validator2-3
node1=$(terrad tendermint show-node-id --home=$HOME/.terrad/validator1)
node2=$(terrad tendermint show-node-id --home=$HOME/.terrad/validator2)
node3=$(terrad tendermint show-node-id --home=$HOME/.terrad/validator3)
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$node1@localhost:26656,$node2@localhost:26656,$node3@localhost:26656\"|g" $HOME/.terrad/validator1/config/config.toml
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$node1@localhost:26656,$node2@localhost:26656,$node3@localhost:26656\"|g" $HOME/.terrad/validator2/config/config.toml
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$node1@localhost:26656,$node2@localhost:26656,$node3@localhost:26656\"|g" $HOME/.terrad/validator3/config/config.toml


# # start all three validators
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3