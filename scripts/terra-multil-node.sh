#!/bin/bash
set -xeu

# always returns true so set -e doesn't exit if it is not running.
killall terrad || true
rm -rf $HOME/.terrad/

# start version 2.0.0
git clone https://github.com/classic-terra/core
cd core
git checkout v2.0.0
make install
cd ..
rm -rf core

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

# cp validator1/config/genesis.json $HOME/.terrad/validator2/config/genesis.json
# cp validator1/config/genesis.json $HOME/.terrad/validator3/config/genesis.json


# change app.toml values
VALIDATOR1_APP_TOML=$HOME/.terrad/validator1/config/app.toml
VALIDATOR2_APP_TOML=$HOME/.terrad/validator2/config/app.toml
VALIDATOR3_APP_TOML=$HOME/.terrad/validator3/config/app.toml

# validator1
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9050|g' $VALIDATOR1_APP_TOML

# validator2
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1316|g' $VALIDATOR2_APP_TOML
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9088|g' $VALIDATOR2_APP_TOML
# sed -i -E 's|localhost:9090|localhost:9088|g' $VALIDATOR2_APP_TOML
sed -i -E 's|0.0.0.0:9091|0.0.0.0:9089|g' $VALIDATOR2_APP_TOML
# sed -i -E 's|localhost:9091|localhost:9089|g' $VALIDATOR2_APP_TOML
sed -i -E 's|tcp://0.0.0.0:10337|tcp://0.0.0.0:10347|g' $VALIDATOR2_APP_TOML

# validator3
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1315|g' $VALIDATOR3_APP_TOML
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9086|g' $VALIDATOR3_APP_TOML
# sed -i -E 's|localhost:9090|localhost:9086|g' $VALIDATOR3_APP_TOML
sed -i -E 's|0.0.0.0:9091|0.0.0.0:9087|g' $VALIDATOR3_APP_TOML
# sed -i -E 's|localhost:9091|localhost:9087|g' $VALIDATOR3_APP_TOML
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

# update
update_test_genesis () {
    # EX: update_test_genesis '.consensus_params["block"]["max_gas"]="100000000"'
    cat $HOME/.terrad/validator1/config/genesis.json | jq "$1" > tmp.json && mv tmp.json $HOME/.terrad/validator1/config/genesis.json
    cat $HOME/.terrad/validator2/config/genesis.json | jq "$1" > tmp.json && mv tmp.json $HOME/.terrad/validator2/config/genesis.json
    cat $HOME/.terrad/validator3/config/genesis.json | jq "$1" > tmp.json && mv tmp.json $HOME/.terrad/validator3/config/genesis.json
}

update_test_genesis '.app_state["gov"]["voting_params"]["voting_period"] = "30s"'
update_test_genesis '.app_state["mint"]["params"]["mint_denom"]= "uluna"'
update_test_genesis '.app_state["gov"]["deposit_params"]["min_deposit"]=[{"denom": "uluna","amount": "1000000"}]'
update_test_genesis '.app_state["crisis"]["constant_fee"]={"denom": "uluna","amount": "1000"}'
update_test_genesis '.app_state["staking"]["params"]["bond_denom"]="uluna"'


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


# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

# ===========

sleep 7

terrad tx wasm store /Users/donglieu/52024/core/scripts/hackatom.wasm --from validator1 --output json --gas auto --gas-adjustment 2.4 --fees 100000000uluna --chain-id testing --home ~/.terrad/validator1 --keyring-backend test -y

sleep 7

terrad tx wasm instantiate 1 '{"verifier":"terra1au3y48s4ezv3j0n77m53h0lf9vnju59vevqvqx","beneficiary":"terra1hxxtn2lhuv9qf2s77llsemtayer5exhmkwh4le"}' --from validator1 --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id testing --home ~/.terrad/validator1 --keyring-backend  test --admin "" -y

sleep 7

terrad tx gov submit-proposal  software-upgrade v4  --upgrade-info v4 --upgrade-height 10  --title upgrade --description upgrade --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing --deposit 10000000uluna -y

sleep 7

terrad tx gov vote 1 yes  --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing -y 

terrad tx gov vote 1 yes  --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing -y 

terrad tx gov vote 1 yes  --from validator3 --keyring-backend test --home ~/.terrad/validator3 --chain-id testing -y 

sleep 50

echo "=========================================================upgrade to 2.1.0================================================================="

# upgrade 2.1.0
killall terrad || true
git clone https://github.com/classic-terra/core
cd core
git checkout v2.1.0
make install
cd ..
rm -rf core

# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

sleep 7

terrad tx gov submit-proposal  software-upgrade v5  --upgrade-info v5 --upgrade-height 20  --title upgrade --description upgrade --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing --deposit 10000000uluna -y

sleep 7

terrad tx gov vote 2 yes  --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing -y 

terrad tx gov vote 2 yes  --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing -y 

terrad tx gov vote 2 yes  --from validator3 --keyring-backend test --home ~/.terrad/validator3 --chain-id testing -y 

sleep 50

echo "=========================================================upgrade to 2.2.0================================================================="
# upgrade 2.2.0
killall terrad || true
git clone https://github.com/classic-terra/core
cd core
git checkout v2.2.0
make install
cd ..
rm -rf core

# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

sleep 7

terrad tx gov submit-legacy-proposal  software-upgrade v6    --upgrade-info v6 --upgrade-height 30 --title upgrade --description upgrade --from validator2 --keyring-backend test --home  ~/.terrad/validator2 --chain-id testing --deposit 10000000uluna -y --no-validate 

sleep 7

terrad tx gov vote 3 yes  --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing -y 

terrad tx gov vote 3 yes  --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing -y 

terrad tx gov vote 3 yes  --from validator3 --keyring-backend test --home ~/.terrad/validator3 --chain-id testing -y 

sleep 50

echo "=========================================================upgrade to 2.3.0================================================================="
# upgrade 2.3.0
killall terrad || true
git clone https://github.com/classic-terra/core
cd core
git checkout v2.3.0
make install
cd ..
rm -rf core

# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

sleep 7

terrad tx gov submit-legacy-proposal  software-upgrade v7    --upgrade-info v7 --upgrade-height 40 --title upgrade --description upgrade --from validator2 --keyring-backend test --home  ~/.terrad/validator2 --chain-id testing --deposit 10000000uluna -y --no-validate 

sleep 7

terrad tx gov vote 4 yes  --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing -y 

terrad tx gov vote 4 yes  --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing -y 

terrad tx gov vote 4 yes  --from validator3 --keyring-backend test --home ~/.terrad/validator3 --chain-id testing -y 

sleep 50


echo "=========================================================upgrade to 2.4.0================================================================="
# upgrade 2.4.0
killall terrad || true
git clone https://github.com/classic-terra/core
cd core
git checkout v2.4.0
make install
cd ..
rm -rf core

# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

sleep 7

terrad tx gov submit-legacy-proposal  software-upgrade v8 --upgrade-info v8 --upgrade-height 50 --title upgrade --description upgrade --from validator2 --keyring-backend test --home  ~/.terrad/validator2 --chain-id testing --deposit 10000000uluna -y --no-validate 

sleep 7

terrad tx gov vote 5 yes  --from validator1 --keyring-backend test --home ~/.terrad/validator1 --chain-id testing -y 

terrad tx gov vote 5 yes  --from validator2 --keyring-backend test --home ~/.terrad/validator2 --chain-id testing -y 

terrad tx gov vote 5 yes  --from validator3 --keyring-backend test --home ~/.terrad/validator3 --chain-id testing -y 

sleep 50


echo "=========================================================upgrade to v3.0.1-rc.2================================================================="
# upgrade v3.0.1-rc.2
killall terrad || true
git clone https://github.com/classic-terra/core
cd core
git checkout v3.0.1-rc.2
make install
cd ..
rm -rf core

# # start all three validators/
screen -S terra1 -t terra1 -d -m terrad start --home=$HOME/.terrad/validator1
screen -S terra2 -t terra2 -d -m terrad start --home=$HOME/.terrad/validator2
screen -S terra3 -t terra3 -d -m terrad start --home=$HOME/.terrad/validator3

sleep 7

