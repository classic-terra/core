#!/bin/sh
set -e pipefail

DEFAULT_CHAIN_ID="localterra"
DEFAULT_VALIDATOR_MONIKER="validator"
DEFAULT_VALIDATOR_MNEMONIC="bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort"
DEFAULT_FAUCET_MNEMONIC="increase bread alpha rigid glide amused approve oblige print asset idea enact lawn proof unfold jeans rabbit audit return chuckle valve rather cactus great"
DEFAULT_RELAYER_MNEMONIC="black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken"

# Override default values with environment variables
CHAIN_ID=${CHAIN_ID:-$DEFAULT_CHAIN_ID}
VALIDATOR_MONIKER=${VALIDATOR_MONIKER:-$DEFAULT_VALIDATOR_MONIKER}
VALIDATOR_MNEMONIC=${VALIDATOR_MNEMONIC:-$DEFAULT_VALIDATOR_MNEMONIC}
FAUCET_MNEMONIC=${FAUCET_MNEMONIC:-$DEFAULT_FAUCET_MNEMONIC}
RELAYER_MNEMONIC=${RELAYER_MNEMONIC:-$DEFAULT_RELAYER_MNEMONIC}

TERRA_HOME=$HOME/.terrad
CONFIG_FOLDER=$TERRA_HOME/config

install_prerequisites () {
    wget -qO /usr/local/bin/dasel https://github.com/TomWright/dasel/releases/latest/download/dasel_linux_amd64
    chmod a+x /usr/local/bin/dasel
    dasel --version
}

edit_genesis () {

    GENESIS=$CONFIG_FOLDER/genesis.json

    # Update staking module
    dasel put -t string -f $GENESIS -v 'uluna' '.app_state.staking.params.bond_denom' 

    # Update crisis module
    dasel put -t string -f $GENESIS -v 'uluna' '.app_state.crisis.constant_fee.denom'

    # Udpate gov module
    dasel put -t string -f $GENESIS -v '60s' '.app_state.gov.voting_params.voting_period'
    dasel put -t string -f $GENESIS -v 'uluna' '.app_state.gov.deposit_params.min_deposit.[0].denom'

    # Update mint module
    dasel put -t string -f $GENESIS -v "uluna" '.app_state.mint.params.mint_denom'

    # Update txfee basedenom
    dasel put -t string -f $GENESIS -v "uluna" '.app_state.txfees.basedenom'

    # Update wasm permission (Nobody or Everybody)
    dasel put -t string -f $GENESIS -v "Everybody" '.app_state.wasm.params.code_upload_access.permission'
}

add_genesis_accounts () {
    
    # Validator
    echo "⚖️ Add validator account"
    echo $VALIDATOR_MNEMONIC | terrad keys add $VALIDATOR_MONIKER --recover --keyring-backend=test --home $TERRA_HOME
    VALIDATOR_ACCOUNT=$(terrad keys show -a $VALIDATOR_MONIKER --keyring-backend test --home $TERRA_HOME)
    terrad add-genesis-account $VALIDATOR_ACCOUNT 100000000000uluna,100000000000stake --home $TERRA_HOME
    
    # Faucet
    echo "🚰 Add faucet account"
    echo $FAUCET_MNEMONIC | terrad keys add faucet --recover --keyring-backend=test --home $TERRA_HOME
    FAUCET_ACCOUNT=$(terrad keys show -a faucet --keyring-backend test --home $TERRA_HOME)
    terrad add-genesis-account $FAUCET_ACCOUNT 100000000000uluna,100000000000stake --home $TERRA_HOME

    # Relayer
    echo "🔗 Add relayer account"
    echo $RELAYER_MNEMONIC | terrad keys add relayer --recover --keyring-backend=test --home $TERRA_HOME
    RELAYER_ACCOUNT=$(terrad keys show -a relayer --keyring-backend test --home $TERRA_HOME)
    terrad add-genesis-account $RELAYER_ACCOUNT 1000000000uluna,1000000000stake --home $TERRA_HOME
    
    terrad gentx $VALIDATOR_MONIKER 500000000uluna --keyring-backend=test --chain-id=$CHAIN_ID --home $TERRA_HOME
    terrad collect-gentxs --home $TERRA_HOME
}

edit_config () {
    # Remove seeds
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v '' '.p2p.seeds' 

    # Expose the rpc
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "tcp://0.0.0.0:26657" '.rpc.laddr' 
}

install_prerequisites
echo "Creating Terra home for $VALIDATOR_MONIKER"
echo $VALIDATOR_MNEMONIC | terrad init -o --chain-id=$CHAIN_ID --home $TERRA_HOME --recover $VALIDATOR_MONIKER
edit_genesis
add_genesis_accounts
edit_config


echo "🏁 Starting $CHAIN_ID..."
terrad start --home $TERRA_HOME
