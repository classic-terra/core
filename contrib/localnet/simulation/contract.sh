#!/bin/sh

SCRIPT=$(realpath "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
CONTRACTPATH=${SCRIPTPATH}/misc/cw721_base.wasm

for j in $(seq 0 4); do

	echo "node $j ..."

	# val stores contract
	echo "... stores a wasm"
	folder="${TESTNET_FOLDER}"/node$j/
	val_addr_name=$(basename $folder)
	val_addr=$(terrad keys show $val_addr_name -a --home ${folder}terrad --keyring-backend $KEYRING_BACKEND)
	out=$(terrad tx wasm store ${CONTRACTPATH} --from ${val_addr_name} --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not store NFT binary" >&2
		exit $code
	fi
	id=$(echo $out | jq -r '.logs[0].events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

	# val instantiates contract
	echo "... instantiates contract"
	msg='{"name":"BaseNFT","symbol":"BASE","minter":"'$val_addr'"}'
	out=$(terrad tx wasm instantiate $id "$msg" --from ${val_addr_name} --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not instantiate NFT contract" >&2
		exit $code
	fi
	contract_addr=$(echo $out | jq -r '.logs[0].events[0].attributes[] | select(.key=="contract_address").value')

	# val mints some tokens
	echo "... mints tokens"
	for i in $(seq 0 4); do
		echo "	- token id: "$i
		msg='{"mint":{"token_id":"'$i'","owner":"'$val_addr'"}}'
		out=$(terrad tx wasm execute $contract_addr "$msg" --from ${val_addr_name} --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
		code=$(echo $out | jq -r '.code')
		if [ "$code" != "0" ]; then
			echo "... Could not mint tokens from contract" $contract_addr >&2
			exit $code
		fi
	done

	# val sends token to other nodes
	echo "... send tokens"
	for i in $(seq 0 4); do
		peer_folder="${TESTNET_FOLDER}"/node$i/
		peer_val_addr_name=$(basename $peer_folder)
		peer_val_addr=$(terrad keys show $peer_val_addr_name -a --home ${peer_folder}terrad --keyring-backend $KEYRING_BACKEND)
		if [ "$peer_val_addr" = "$val_addr" ]; then
			continue
		fi
		msg='{"transfer_nft":{"recipient":"'$peer_val_addr'","token_id":"'$i'"}}'
		out=$(terrad tx wasm execute $contract_addr "$msg" --from $val_addr_name --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
		code=$(echo $out | jq -r '.code')
		if [ "$code" != "0" ]; then
			echo "... Could not transfer NFT id $i from $val_addr to $peer_val_addr (contract: $contract_addr)" >&2
			exit $code
		fi
	done

done
