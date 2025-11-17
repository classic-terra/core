#!/bin/sh

BINARY=_build/old/terrad
WASMFILE="cw721_base.wasm"
CONTRACTPATH="scripts/wasm/contracts/$WASMFILE"
KEYRING_BACKEND="test"
HOME=mytestnet
CHAIN_ID=localterra

TXHASH=()
CONTRACT_ADDRESSES=()

echo "SETTING UP SMART CONTRACT INTERACTION"

# create two contracts only
for j in $(seq 0 1); do

	echo "key test$j ..."

	# stores contract
	echo "... stores a wasm"
	addr=$($BINARY keys show test$j -a --home $HOME --keyring-backend $KEYRING_BACKEND)
	out=$($BINARY tx wasm store ${CONTRACTPATH} --from test$j --output json --gas auto --gas-adjustment 2.3 --fees 1000000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y )
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not store NFT binary" >&2
		echo $out >&2
		exit $code
	fi
	sleep 5
	txhash=$(echo $out | jq -r '.txhash')
	TXHASH+=($txhash)
	id=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[1].value')

	# instantiates contract
	echo "... instantiates contract"
	msg='{"name":"BaseNFT","symbol":"BASE","minter":"'$addr'"}'
	out=$($BINARY tx wasm instantiate $id "$msg" --from test$j --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y --label mynft --admin $addr)
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not instantiate NFT contract" >&2
		echo $out >&2
		exit $code
	fi
	sleep 5
	txhash=$(echo $out | jq -r '.txhash')
	TXHASH+=("$txhash")
	contract_addr=$($BINARY q tx $txhash -o json | jq -r '.raw_log' | jq -r '.[0].events[1].attributes[0].value')
	CONTRACT_ADDRESSES+=("$contract_addr")
	echo $($BINARY q tx $txhash -o json | jq -r '.raw_log')
	echo "contract_addr = $contract_addr"

	# mints some tokens
	echo "... mints tokens"
	for i in $(seq 0 2); do
		echo "	- token id: "$i
		msg='{"mint":{"token_id":"'$i'","owner":"'$addr'"}}'
		out=$($BINARY tx wasm execute $contract_addr "$msg" --from test$j --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
		code=$(echo $out | jq -r '.code')
		if [ "$code" != "0" ]; then
			echo "... Could not mint tokens from contract" $contract_addr >&2
			echo $out >&2
			exit $code
		fi
		txhash=$(echo $out | jq -r '.txhash')
		TXHASH+=("$txhash")

		sleep 5
	done

	# sends token to other nodes
	echo "... send tokens"
	for i in $(seq 0 2); do
		peer_addr=$($BINARY keys show test$i -a --home $HOME --keyring-backend $KEYRING_BACKEND)
		if [ "$peer_addr" = "$addr" ]; then
			continue
		fi
		msg='{"transfer_nft":{"recipient":"'$peer_addr'","token_id":"'$i'"}}'
		out=$($BINARY tx wasm execute $contract_addr "$msg" --from test$j --output json --gas auto --gas-adjustment 2.3 --fees 20000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
		code=$(echo $out | jq -r '.code')
		if [ "$code" != "0" ]; then
			echo "... Could not transfer NFT id $i from $addr to $peer_addr (contract: $contract_addr)" >&2
			echo $out >&2
			exit $code
		fi
		txhash=$(echo $out | jq -r '.txhash')
		TXHASH+=("$txhash")

		sleep 5
	done

		# write the contract state to a file
	echo "Writing contract state to file"
	mkdir -p scripts/wasm/contract_states/
	$BINARY q wasm contract-state all $contract_addr --output json --home $HOME > scripts/wasm/contract_states/old_$contract_addr.json


done

TXHASH_STRING="${TXHASH[*]}"
CONTRACT_ADDRESSES_STRING="${CONTRACT_ADDRESSES[*]}"

echo "TXHASH = $TXHASH_STRING"
echo "CONTRACT_ADDRESSES = $CONTRACT_ADDRESSES_STRING"

export TXHASH_STRING
export CONTRACT_ADDRESSES_STRING