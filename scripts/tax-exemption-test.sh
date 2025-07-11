#!/bin/bash

# Default values
DEFAULT_HOME="mytestnet"
DEFAULT_BINARY="_build/new/terrad"
DEFAULT_CHAIN_ID="localterra"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --binary=*)
      BINARY="${1#*=}"
      shift
      ;;
    --home=*)
      HOME="${1#*=}"
      shift
      ;;
    --chain-id=*)
      CHAIN_ID="${1#*=}"
      shift
      ;;
    *)
      echo "Unknown parameter: $1"
      echo "Usage: $0 [--binary=BINARY_PATH] [--home=HOME_DIR] [--chain-id=CHAIN_ID]"
      exit 1
      ;;
  esac
done

# Set defaults if parameters were not provided
HOME=${HOME:-$DEFAULT_HOME}
BINARY=${BINARY:-$DEFAULT_BINARY}
CHAIN_ID=${CHAIN_ID:-$DEFAULT_CHAIN_ID}

echo "Using binary: $BINARY"
echo "Using home directory: $HOME"
echo "Using chain ID: $CHAIN_ID"

ROOT=$(pwd)
DENOM=uluna
KEY="test0"
KEY1="test1"
KEY2="test2"
KEYRING="test"
ZONE_NAME="testzone"

# underscore so that go tool will not take gocache into account
mkdir -p _build/gocache
export GOMODCACHE=$ROOT/_build/gocache

# install new binary
if ! command -v $BINARY &> /dev/null
then
    GOBIN="$ROOT/$(dirname $BINARY)" go install -mod=readonly ./...
fi

# spin up mytestnet
if [[ "$OSTYPE" == "darwin"* ]]; then
    screen -L -dmS node1 bash scripts/run-node.sh $BINARY $DENOM
else
    screen -L -Logfile $HOME/log-screen.txt -dmS node1 bash scripts/run-node.sh $BINARY $DENOM
fi

sleep 20

# get test addresses
test1=$($BINARY keys show $KEY1 -a --keyring-backend $KEYRING --home $HOME)
test2=$($BINARY keys show $KEY2 -a --keyring-backend $KEYRING --home $HOME)
echo "Test addresses = $test1,$test2"

echo ""
echo "=========== TEST 1: ADD TAX EXEMPTION ZONE ==========="
echo ""

# Submit proposal to add tax exemption zone
$BINARY tx gov submit-proposal add-tax-exemption-zone $ZONE_NAME $test1,$test2 --exempt-incoming --exempt-outgoing --title "Add tax exemption zone" --description "Add tax exemption zone for testing" --from $KEY --keyring-backend $KEYRING --gas auto --gas-adjustment 2.3 --fees "200000000${DENOM}" --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Deposit tokens for the proposal
$BINARY tx gov deposit 1 "200000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Vote yes on proposal
$BINARY tx gov vote 1 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 1 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

# Wait for proposal to pass
sleep 5
while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 1 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "Checking if zone was created:"
# Query zone details - this will depend on the actual module query structure
# Using a placeholder query - update with actual query command
$BINARY q taxexemption zone $ZONE_NAME -o json | jq "."

echo ""
echo "=========== TEST 2: MODIFY TAX EXEMPTION ZONE ==========="
echo ""

# Submit proposal to modify tax exemption zone
$BINARY tx gov submit-proposal modify-tax-exemption-zone $ZONE_NAME --exempt-cross-zone --title "Modify tax exemption zone" --description "Enable cross zone exemption" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Deposit tokens for the proposal
$BINARY tx gov deposit 2 "200000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Vote yes on proposal
$BINARY tx gov vote 2 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 2 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

# Wait for proposal to pass
sleep 5
while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 2 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "Checking if zone was modified:"
# Query zone details after modification
$BINARY q taxexemption zone $ZONE_NAME -o json | jq "."

echo ""
echo "=========== TEST 3: ADD TAX EXEMPTION ADDRESS ==========="
echo ""

# Add additional test address to the zone
test0=$($BINARY keys show $KEY -a --keyring-backend $KEYRING --home $HOME)

$BINARY tx gov submit-proposal add-tax-exemption-address $ZONE_NAME $test0 --title "Add tax exemption address" --description "Add address to tax exemption list" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Deposit tokens for the proposal
$BINARY tx gov deposit 3 "20000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Vote yes on proposal
$BINARY tx gov vote 3 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 3 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

# Wait for proposal to pass
sleep 5
while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 3 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "Checking if address was added to zone:"
# Query zone addresses
$BINARY q taxexemption zone $ZONE_NAME -o json | jq ".addresses"

echo ""
echo "=========== TEST 4: REMOVE TAX EXEMPTION ADDRESS ==========="
echo ""

# Remove an address from the zone
$BINARY tx gov submit-proposal remove-tax-exemption-address $ZONE_NAME $test1 --title "Remove tax exemption address" --description "Remove address from tax exemption list" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Deposit tokens for the proposal
$BINARY tx gov deposit 4 "20000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Vote yes on proposal
$BINARY tx gov vote 4 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 4 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

# Wait for proposal to pass
sleep 5
while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 4 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "Checking if address was removed from zone:"
# Query zone addresses after removal
$BINARY q taxexemption zone $ZONE_NAME -o json | jq ".addresses"

echo ""
echo "=========== TEST 5: REMOVE TAX EXEMPTION ZONE ==========="
echo ""

# Submit proposal to remove tax exemption zone
$BINARY tx gov submit-proposal remove-tax-exemption-zone $ZONE_NAME --title "Remove tax exemption zone" --description "Remove tax exemption zone completely" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Deposit tokens for the proposal
$BINARY tx gov deposit 5 "20000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

# Vote yes on proposal
$BINARY tx gov vote 5 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 5 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

# Wait for proposal to pass
sleep 5
while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 5 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "Checking if zone was removed:"
# Attempt to query the removed zone - should return an error or empty result
$BINARY q taxexemption zone $ZONE_NAME -o json 2>&1 || echo "Zone successfully removed"

echo ""
echo "=========== TAX EXEMPTION TESTING COMPLETE ==========="

# Optional: Also test burn tax exemption for completeness
echo ""
echo "=========== BONUS TEST: BURN TAX EXEMPTION ==========="
echo ""

# add test1 to burn tax exemption list
$BINARY tx gov submit-legacy-proposal add-burn-tax-exemption-address "$test1,$test2" --title "Burn tax exemption address" --description "Burn tax exemption address" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

$BINARY tx gov deposit 6 "20000000${DENOM}" --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

$BINARY tx gov vote 6 yes --from $KEY --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y
$BINARY tx gov vote 6 yes --from $KEY1 --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME -y

sleep 5

while true; do
    PROPOSAL_STATUS=$($BINARY q gov proposal 6 --output=json | jq ".status" -r)
    echo $PROPOSAL_STATUS
    if [ $PROPOSAL_STATUS = "PROPOSAL_STATUS_PASSED" ]; then
        break
    else
        sleep 10
    fi
done

echo ""
echo "CHECK BURN TAX EXEMPTION LIST"
echo ""

# check burn tax exemption address
$BINARY q treasury burn-tax-exemption-list -o json | jq ".addresses"

echo ""
echo "ALL TESTS COMPLETED" 