#!/bin/sh

echo "Submitting text prop from key test0"

DEPOSIT_DENOM=$(cat $HOME/config/genesis.json | jq -r '.app_state.gov.deposit_params.min_deposit[0].denom')
DEPOSIT_AMNT=$(cat $HOME/config/genesis.json | jq -r '.app_state.gov.deposit_params.min_deposit[0].amount')

out=$(terrad tx gov submit-proposal --from test0 --type text --title "Proposal" --description "This is a proposal" --deposit ${DEPOSIT_AMNT}${DEPOSIT_DENOM} --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y --node $(sh $SIMULATION_FOLDER/next_node.sh))
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
	echo "... Could not submit prop" >&2
	exit $code
fi
sleep 10
txhash=$(echo $out | jq -r '.txhash')
id=$(terrad q tx $txhash -o json --node $(sh $SIMULATION_FOLDER/next_node.sh) | jq -r '.raw_log' | jq -r '.[0].events[4].attributes[0].value')

sleep 10

for j in $(seq 0 3); do

	echo "submitting vote from test$j for prop $id..."

	out=$(terrad tx gov vote $id yes --from test$j --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y --node $(sh $SIMULATION_FOLDER/next_node.sh))
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not vote"
		exit $code
	fi

	sleep 10
done
