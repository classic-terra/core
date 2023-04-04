# Mainnet validator image

This image bases on the following assumption:
* validator already has a config folder on their machine (so that Dockerfile will not init another config folder)

Based on that asumption, the image will do the following:
* setup validator production env
* contain mainnet genesis.json
* contain addrbook.json

The container will do the following:
* Download newest snapshot from quicksync.io to /terra/data volume if not already have
* Copy addrbook.json to /terra/.terra volume
* Copy genesis.json to /terra/.terra volume
* Remove old data and copy snapshot data to /terra/.terra

## What you can do as an user?
1. To force download snapshot again, delete snapshot file in [data folder](data/README.md)
2. To configure tendermint and cosmos, define ENV in docker-compose.yaml. This is a feature provided by https://github.com/spf13/viper

```
TERRAD_P2P_LADDR=tcp://0.0.0.0:26656
```

the ENV above will configure --p2p.laddr

If you want to enable API in app.toml, add this to docker-compose.yml

```
environment:
    - TERRAD_API_ENABLE=true
```

some common env:
* TERRAD_RPC_LADDR: change address of rpc
* will supply more if people ask

3. To continue running node after setting up something, change this line in docker-compose.yml

```
environment:
    - CONTINUE=true
```

Setting CONTINUE=false will start node again