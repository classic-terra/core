#!/bin/bash

set -ex

# initialize Hermes relayer configuration
mkdir -p /root/.hermes/
touch /root/.hermes/config.toml

# setup Hermes relayer configuration
tee /root/.hermes/config.toml <<EOF
[global]
log_level = 'info'
[mode]
[mode.clients]
enabled = true
refresh = true
misbehaviour = true
[mode.connections]
enabled = true
[mode.channels]
enabled = true
[mode.packets]
enabled = true
clear_interval = 100
clear_on_start = true
tx_confirmation = true
[rest]
enabled = true
host = '0.0.0.0'
port = 3031
[telemetry]
enabled = true
host = '127.0.0.1'
port = 3001
[[chains]]
id = '$TERRA_A_E2E_CHAIN_ID'
rpc_addr = 'http://$TERRA_A_E2E_VAL_HOST:26657'
grpc_addr = 'http://$TERRA_A_E2E_VAL_HOST:9090'
websocket_addr = 'ws://$TERRA_A_E2E_VAL_HOST:26657/websocket'
rpc_timeout = '10s'
account_prefix = 'terra'
key_name = 'val01-terra-a'
store_prefix = 'ibc'
max_gas = 6000000
gas_price = { price = 0.000, denom = 'luna' }
gas_multiplier = 1.1
max_msg_num = 30
max_tx_size = 2097152
clock_drift = '1m' # to accomdate docker containers
trusting_period = '239seconds'
[chains.trust_threshold]
numerator = '1'
denominator = '3'
[[chains]]
id = '$TERRA_B_E2E_CHAIN_ID'
rpc_addr = 'http://$TERRA_B_E2E_VAL_HOST:26657'
grpc_addr = 'http://$TERRA_B_E2E_VAL_HOST:9090'
websocket_addr = 'ws://$TERRA_B_E2E_VAL_HOST:26657/websocket'
rpc_timeout = '10s'
account_prefix = 'terra'
key_name = 'val01-terra-b'
store_prefix = 'ibc'
max_gas = 6000000
gas_price = { price = 0.000, denom = 'luna' }
gas_multiplier = 1.1
max_msg_num = 30
max_tx_size = 2097152
clock_drift = '1m' # to accomdate docker containers
trusting_period = '239seconds'
[chains.trust_threshold]
numerator = '1'
denominator = '3'
EOF

# import keys

hermes keys add --hd-path "m/44'/330'/0'/0/0" --chain ${TERRA_A_E2E_CHAIN_ID} --key-name "val01-terra-a" --mnemonic-file "${TERRA_A_E2E_VAL_MNEMONIC}" --overwrite
hermes keys add --hd-path "m/44'/330'/0'/0/0" --chain ${TERRA_B_E2E_CHAIN_ID} --key-name "val01-terra-a" --mnemonic-file "${TERRA_B_E2E_VAL_MNEMONIC}" --overwrite
# hermes keys restore ${TERRA_B_E2E_CHAIN_ID} -n "val01-terra-b" -m "${TERRA_B_E2E_VAL_MNEMONIC}"
# hermes keys restore ${TERRA_A_E2E_CHAIN_ID} -n "val01-terra-a" -m "${TERRA_A_E2E_VAL_MNEMONIC}"

# start Hermes relayer
hermes start
