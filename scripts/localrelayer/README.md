# Localrelayer

Localrelayer is a local testing environment composed of two LocalTerra instances connected by a relayer.

![Architecture](./assets/architecture.png)

## Endpoints

| Chain ID         | Component  | Endpoint                 |
|------------------|------------|--------------------------|
| `localterra-a` | `RPC`      | <http://localhost:26657> |
| `localterra-a` | `REST/LCD` | <http://localhost:1317>  |
| `localterra-a` | `gRPC`     | <http://localhost:9090>  |
| `localterra-b` | `RPC`      | <http://localhost:36657> |
| `localterra-b` | `REST/LCD` | <http://localhost:31317> |
| `localterra-b` | `gRPC`     | <http://localhost:39090> |
| `-`            | `hermes`   | <http://localhost:3000>  |

## Accounts

By default the following mnemonics are used:

| Chain ID         | Account       | Mnemonic                                                                                                                                                          |
|------------------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `localterra-a` | `validator-a` | *family album bird seek tilt color pill danger message abuse manual tent almost ridge boost blast high comic core quantum spoon coconut oyster remove*            |
| `localterra-a` | `relayer`     | *black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken* |
| `localterra-b` | `validator-b` | *family album bird seek tilt color pill danger message abuse manual tent almost ridge boost blast high comic core quantum spoon coconut oyster remove*            |
| `localterra-b` | `relayer`     | *black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken* |


## Deploy

Build a local docker image with current changes

```bash
make build
```

Start the testing environment:

```bash
make start
```

The command will:

1. create a local docker network:

```bash
 ⠿ Network localrelayer_localterra        Created
```

2. run the following containers:

```bash
 ⠿ Container localrelayer-localterra-b-1  Created
 ⠿ Container localrelayer-localterra-a-1  Created
 ⠿ Container localrelayer-hermes-1          Created
```

> If you don't want the logs, you can start in detached mode with the following command:
> 
> `make startd`

Check that everything is running:

```bash
docker ps
```

Expected output:

```bash
❯ docker ps
CONTAINER ID   IMAGE                          COMMAND                  CREATED              STATUS         PORTS                                                                                   NAMES
318c89d3015f   informalsystems/hermes:1.1.0   "/home/hermes/setup.…"   About a minute ago   Up 2 seconds   0.0.0.0:3000->3000/tcp                                                                  localrelayer-hermes-1
d90ec29c7a6f   local:terra                  "/terra/setup.sh"      About a minute ago   Up 3 seconds   26656/tcp, 0.0.0.0:31317->1317/tcp, 0.0.0.0:39090->9090/tcp, 0.0.0.0:36657->26657/tcp   localrelayer-localterra-b-1
e36cead49a07   local:terra                  "/terra/setup.sh"      About a minute ago   Up 3 seconds   0.0.0.0:1317->1317/tcp, 0.0.0.0:9090->9090/tcp, 0.0.0.0:26657->26657/tcp, 26656/tcp     localrelayer-localterra-a-1
```

## Usage

### Interact with chain

Check `localterra-a` status:

```bash
curl -s http://localhost:26657/status
```

Check `localterra-b` status:

```bash
curl -s http://localhost:36657/status
```

### Hermes

You can test that hermes is working by sending a test IBC transaction.

Make sure `hermes` is running:

```bash
❯ docker ps | grep hermes
```

Expected output:

```bash
318c89d3015f   informalsystems/hermes:1.1.0   "/home/hermes/setup.…"   23 minutes ago   Up 22 minutes   0.0.0.0:3000->3000/tcp  
```

Exec inside the container:

```bash
docker exec -ti localrelayer-hermes-1 sh
```

Send a transaction:

```bash
hermes tx ft-transfer --timeout-seconds 1000 \
    --dst-chain localterra-a \
    --src-chain localterra-b \
    --src-port transfer \
    --src-channel channel-0 \
    --amount 100 \
    --denom uluna
```

Expected output:

```bash
2022-12-01T11:41:22.351909Z  INFO ThreadId(01) using default configuration from '/root/.hermes/config.toml'
SUCCESS [
    IbcEventWithHeight {
        event: SendPacket(
            SendPacket {
                packet: Packet {
                    sequence: Sequence(
                        1,
                    ),
                    source_port: PortId(
                        "transfer",
                    ),
                    source_channel: ChannelId(
                        "channel-0",
                    ),
                    destination_port: PortId(
                        "transfer",
                    ),
                    destination_channel: ChannelId(
                        "channel-0",
                    ),
                    data: [123, 34, 97, 109, 111, 117, 110, 116, 34, 58, 34, 49, 48, 48, 34, 44, 34, 100, 101, 110, 111, 109, 34, 58, 34, 117, 111, 115, 109, 111, 34, 44, 34, 114, 101, 99, 101, 105, 118, 101, 114, 34, 58, 34, 111, 115, 109, 111, 49, 113, 118, 100, 101, 117, 52, 120, 51, 52, 114, 97, 112, 112, 51, 119, 99, 56, 102, 121, 109, 53, 103, 52, 119, 117, 51, 52, 51, 109, 115, 119, 120, 50, 101, 120, 107, 117, 103, 34, 44, 34, 115, 101, 110, 100, 101, 114, 34, 58, 34, 111, 115, 109, 111, 49, 113, 118, 100, 101, 117, 52, 120, 51, 52, 114, 97, 112, 112, 51, 119, 99, 56, 102, 121, 109, 53, 103, 52, 119, 117, 51, 52, 51, 109, 115, 119, 120, 50, 101, 120, 107, 117, 103, 34, 125],
                    timeout_height: Never,
                    timeout_timestamp: Timestamp {
                        time: Some(
                            Time(
                                2022-12-01 11:57:59.365129852,
                            ),
                        ),
                    },
                },
            },
        ),
        height: Height {
            revision: 0,
            height: 1607,
        },
    },
]
```
