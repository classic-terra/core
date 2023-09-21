#!/bin/bash

BINARY=${BINARY:-_build/old/terrad}
KEYRING_BACKEND="test"
HOME=mytestnet
CHAIN_ID=localterra

if [ -z "$CONTRACT_ADDRESS_STRING" ]; then
	echo "CONTRACT_ADDRESS_STRING is empty"
	exit 1
fi

read -r -a CONTRACT_ADDRESS <<< ${CONTRACT_ADDRESS_STRING:-""}
echo "CONTRACT_ADDRESS = ${CONTRACT_ADDRESS[@]}"

# specify query function to use
query_res=""
query_wasm_old() {
    contract=$1
    query=$2
    res=$($BINARY query wasm contract-store $contract "$query" --chain-id $CHAIN_ID --home $HOME -o json | jq -r '.query_result')
    query_res=$res
}

query_wasm_new() {
    contract=$1
    query=$2
    res=$($BINARY query wasm contract-state smart $contract "$query" --chain-id $CHAIN_ID --home $HOME -o json | jq -r '.data')
    query_res=$res
}

query_func=query_wasm_old
if [ "$BINARY" = "_build/new/terrad" ]; then
    query_func=query_wasm_new
fi

# submit proposal
echo "... submit proposal"
proposal=$(jq -n '
{
    "submit_proposal": {
        "title": "Proposal X",
        "description": "This is proposal X"
    }
}')

base64_proposal=$(base64 <<< $proposal)

echo $base64_proposal

msg=$(jq -n '
{
   "send": {
		"amount": "2000",
		"contract": "'${CONTRACT_ADDRESS[3]}'",
		"msg": "'$base64_proposal'"
	}
}')
echo $msg
out=$($BINARY tx wasm execute ${CONTRACT_ADDRESS[0]} "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
code=$(echo $out | jq -r '.code')
if [ "$code" != "0" ]; then
    echo "... Could not execute contract" >&2
    echo $out >&2
    exit $code
fi
txhash=$(echo $out | jq -r ."txhash")
echo $txhash

sleep 10

# vote proposal for test1 and test2
query=$(jq -n '
{
    "proposals": {}
}')

$query_func ${CONTRACT_ADDRESS[3]} "$query"
proposal_count=$(echo $query_res | jq -r '.proposal_count')

msg=$(jq -n '
{
   "cast_vote": {
        "proposal_id": '$((proposal_count))',
        "vote": "For"
    }
}')
echo $msg

for i in $(seq 1 2); do
    echo "... test$i vote proposal $proposal_count"
    out=$($BINARY tx wasm execute ${CONTRACT_ADDRESS[3]} "$msg" --from test$i --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
    code=$(echo $out | jq -r '.code')
    if [ "$code" != "0" ]; then
        echo "... Could not execute contract" >&2
        echo $out >&2
        exit $code
    fi
    sleep 10
    txhash=$(echo $out | jq -r '.txhash')
    echo $txhash
done

# end proposal
query=$(jq -n '
{
    "proposal": {
        "proposal_id": '$((proposal_count))'
    }
}')
$query_func ${CONTRACT_ADDRESS[3]} "$query"
end_block=$(echo $query_res | jq -r '.end_block')
while true; do 
    BLOCK_HEIGHT=$($BINARY status | jq '.SyncInfo.latest_block_height' -r)
    echo "BLOCK HEIGHT = $BLOCK_HEIGHT"
    # check if block height is greater than end_block
    if (( $BLOCK_HEIGHT > $end_block )); then
        # ending proposal
        msg=$(jq -n '
        {
            "end_proposal": {
                "proposal_id": '$((proposal_count))'
            }
        }')
        echo $msg
        out=$($BINARY tx wasm execute ${CONTRACT_ADDRESS[3]} "$msg" --from test0 --output json --gas auto --gas-adjustment 2.3 --fees 100000000uluna --chain-id $CHAIN_ID --home $HOME --keyring-backend $KEYRING_BACKEND -y)
        code=$(echo $out | jq -r '.code')
        if [ "$code" != "0" ]; then
            echo "... Could not execute contract" >&2
            echo $out >&2
            exit $code
        fi
        break
    fi
    sleep 10
done

sleep 10

# check result of proposal
echo "... query proposals"
query=$(jq -n '
{
    "proposal": {
        "proposal_id": '$((proposal_count))'
    }
}')
$query_func ${CONTRACT_ADDRESS[3]} "$query"
status=$(echo $query_res | jq -r '.status')
echo $status