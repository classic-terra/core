SCRIPT=$(realpath "$0")
SCRIPTPATH=$(dirname "$SCRIPT")

echo "Submitting text prop from 0"
folder="${TESTNET_FOLDER}"/node0/
val_addr_name=$(basename $folder)
val_addr=$(terrad keys show $val_addr_name -a --home ${folder}terrad --keyring-backend $KEYRING_BACKEND)

DEPOSIT_DENOM=$(cat ${folder}terrad/config/genesis.json | jq -r '.app_state.gov.deposit_params.min_deposit[0].denom')
DEPOSIT_AMNT=$(cat ${folder}terrad/config/genesis.json | jq -r '.app_state.gov.deposit_params.min_deposit[0].amount')

out=$(terrad tx gov submit-proposal --from ${val_addr_name} --type text --title "Proposal" --description "This is a proposal" --deposit ${DEPOSIT_AMNT}${DEPOSIT_DENOM} --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
	echo "... Could not submit prop" >&2
	exit $code
fi
id=$(echo $out | jq -r '.logs[0].events[] | select(.type == "submit_proposal") | .attributes[] | select(.key == "proposal_id") | .value')

for j in $(seq 0 4); do

	echo "submitting vote from node $j for prop $id..."

	folder="${TESTNET_FOLDER}"/node$j/
	val_addr_name=$(basename $folder)
	val_addr=$(terrad keys show $val_addr_name -a --home ${folder}terrad --keyring-backend $KEYRING_BACKEND)
	out=$(terrad tx gov vote $id yes --from ${val_addr_name} --broadcast-mode block --output json --gas auto --gas-adjustment 2.3 --chain-id $CHAIN_ID --home ${folder}terrad --keyring-backend $KEYRING_BACKEND -y)
	code=$(echo $out | jq -r '.code')
	if [ "$code" != "0" ]; then
		echo "... Could not vote"
		exit $code
	fi

done
