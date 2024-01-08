#!/bin/sh

CHAIN_ID=localterra
TERRA_HOME=$HOME/.terrad
CONFIG_FOLDER=$TERRA_HOME/config
MONIKER=val
STATE='false'

MNEMONIC="bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort"
POOLSMNEMONIC="traffic cool olive pottery elegant innocent aisle dial genuine install shy uncle ride federal soon shift flight program cave famous provide cute pole struggle"

while getopts s flag
do
    case "${flag}" in
        s) STATE='true';;
    esac
done

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
    dasel put -t string -f $GENESIS -v 'uluna' '.app_state.mint.params.mint_denom' 

    # Update txfee basedenom
    dasel put -t string -f $GENESIS -v 'uluna' '.app_state.txfees.basedenom' 

    # Update wasm permission (Nobody or Everybody)
    dasel put -t string -f $GENESIS -v 'Everybody' '.app_state.wasm.params.code_upload_access.permission' 
}

add_genesis_accounts () {

    terrad add-genesis-account terra1a7fgca0746t9kjz079s0m63eqkczfjp3luesac 100000000000uluna --home $TERRA_HOME
    # note such large amounts are set for e2e tests on FE 
    terrad add-genesis-account terra192xhdwsnc44zz0dsnhf7sq7rhmtuklf7nupy53 9999999999999999999999999999999999999999999999999uluna --home $TERRA_HOME
    terrad add-genesis-account terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra17lmam6zguazs5q5u6z5mmx76uj63gldnse2pdp 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra1757tkx08n0cqrw7p86ny9lnxsqeth0wgp0em95 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra199vw7724lzkwz6lf2hsx04lrxfkz09tg8dlp6r 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra18wlvftxzj6zt0xugy2lr9nxzu402690ltaf4ss 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra1e8ryd9ezefuucd4mje33zdms9m2s90m57878v9 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra17tv2hvwpg0ukqgd2y5ct2w54fyan7z0zxrm2f9 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra1lkccuqgj6sjwjn8gsa9xlklqv4pmrqg9dx2fxc 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra1333veey879eeqcff8j3gfcgwt8cfrg9mq20v6f 100000000000uluna --home $TERRA_HOME
    terrad add-genesis-account terra1fmcjjt6yc9wqup2r06urnrd928jhrde6gcld6n 1000000000000uluna --home $TERRA_HOME

    echo $MNEMONIC | terrad keys add $MONIKER --recover --keyring-backend=test --home $TERRA_HOME
    echo $POOLSMNEMONIC | terrad keys add pools --recover --keyring-backend=test --home $TERRA_HOME
    terrad gentx $MONIKER 500000000uluna --keyring-backend=test --chain-id=$CHAIN_ID --home $TERRA_HOME

    terrad collect-gentxs --home $TERRA_HOME
}

edit_config () {

    # Remove seeds
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v '' '.p2p.seeds' 

    # Expose the rpc
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "tcp://0.0.0.0:26657" '.rpc.laddr' 
    
    # Expose pprof for debugging
    # To make the change enabled locally, make sure to add 'EXPOSE 6060' to the root Dockerfile
    # and rebuild the image.
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "0.0.0.0:6060" '.rpc.pprof_laddr' 
}

enable_cors () {

    # Enable cors on RPC
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "*" '.rpc.cors_allowed_origins.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "Accept-Encoding" '.rpc.cors_allowed_headers.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "DELETE" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "OPTIONS" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "PATCH" '.rpc.cors_allowed_methods.[]'
    dasel put -t string -f $CONFIG_FOLDER/config.toml -v "PUT" '.rpc.cors_allowed_methods.[]'

    # Enable unsafe cors and swagger on the api
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.swagger'
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.enabled-unsafe-cors'

    # Enable cors on gRPC Web
    dasel put -t bool -f $CONFIG_FOLDER/app.toml -v "true" '.grpc-web.enable-unsafe-cors'
}

run_with_retries() {
  cmd=$1
  success_msg=$2

  substring='code: 0'
  COUNTER=0

  while [ $COUNTER -lt 15 ]; do
    string=$(eval $cmd 2>&1)
    echo $string

    if [ "$string" != "${string%"$substring"*}" ]; then
      echo "$success_msg"
      break
    else
      COUNTER=$((COUNTER+1))
      sleep 0.5
    fi
  done
}

    echo $MNEMONIC | terrad init -o --chain-id=$CHAIN_ID --home $TERRA_HOME --recover $MONIKER
    install_prerequisites
    edit_genesis
    add_genesis_accounts
    edit_config
    enable_cors

terrad start --home $TERRA_HOME &

wait
