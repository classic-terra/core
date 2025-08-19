#!/bin/bash

ITER=$1
SLEEP=$2
NODEADDR=$3

if [ -z "$1" ]; then
  echo "Need to input number of iterations to run..."
  exit 1
fi

if [ -z "$2" ]; then
  echo "Need to input number of seconds to sleep between iterations"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Need to input node address to poll..."
  exit 1
fi

CNT=0
FAILED_TESTS=0
# Initialize passed tests tracking (not using associative array for better compatibility)
PASSED_TESTS=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if node is ready
check_node_ready() {
    local max_attempts=30  # Maximum number of attempts (30 * 5 = 150 seconds)
    local attempt=0
    
    echo "Checking REST API..."
    while [ $attempt -lt $max_attempts ]; do
        if curl -s ${NODEADDR}:1317/cosmos/base/tendermint/v1beta1/blocks/latest > /dev/null; then
            echo -e "${GREEN}✓ REST API is ready${NC}"
            return 0
        fi
        
        attempt=$((attempt + 1))
        if [ $((attempt % 6)) -eq 0 ]; then  # Show status every 30 seconds
            echo "Still waiting for REST API... ($(($attempt / 6)) minutes passed)"
        fi
        sleep 5
    done
    
    echo -e "${RED}Timeout waiting for REST API${NC}"
    return 1
}

# Function to run a single test case
run_test_case() {
    local test_name=$1
    local endpoint=$2
    local request_data=$3
    local expected_field=$4
    
    echo -e "${YELLOW}Running test: ${test_name}${NC}"
    
    # Make the API request
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$request_data" \
        ${NODEADDR}:1317${endpoint})

    # Check for error responses first
    if echo "$RESPONSE" | grep -q '"code":[^0]'; then
        echo -e "${RED}✗ Test FAILED: API returned error${NC}"
        echo "Response: $RESPONSE"
        return 1
    fi

    # Check if response contains expected field
    if echo "$RESPONSE" | grep -q "$expected_field"; then
        echo -e "${GREEN}✓ Test PASSED: Response contains expected field${NC}"
        echo "Response: $RESPONSE"
        return 0
    else
        echo -e "${RED}✗ Test FAILED: Response does not contain expected field${NC}"
        echo "Expected to find: $expected_field"
        echo "Actual response: $RESPONSE"
        return 1
    fi
}

# Test Cases
# Format: "test_name|endpoint|request_data|expected_field"
TEST_CASES=(
    "Tax Computation|/terra/tx/v1beta1/compute_tax|\
{\"tx\":{\"body\":{\"messages\":[{\"@type\":\"/cosmos.bank.v1beta1.MsgSend\",\"from_address\":\"terra1xxxx\",\"to_address\":\"terra1xxxx\",\"amount\":[{\"denom\":\"uluna\",\"amount\":\"1000000\"}]}],\"memo\":\"\",\"timeout_height\":\"0\",\"extension_options\":[],\"non_critical_extension_options\":[]},\"auth_info\":{\"signer_infos\":[],\"fee\":{\"amount\":[],\"gas_limit\":\"200000\",\"payer\":\"\",\"granter\":\"\"},\"tip\":null},\"signatures\":[]}}\
|tax_amount"
    # Add more test cases here in the same format
    # "Test Name|/endpoint/path|{request-json}|expected_field"
)

# Wait for node to be ready first
if ! check_node_ready; then
    echo -e "${RED}ERROR: Failed to connect to REST API${NC}"
    exit 1
fi

# Trap any unexpected errors
echo -e "${GREEN}Node is ready. Starting API regression tests...${NC}"

while [ ${CNT} -lt $ITER ]; do
    echo -e "\n${YELLOW}Test iteration ${CNT}...${NC}"
    
    # Run each test case that hasn't passed yet
    ALL_PASSED=true
    for test_case in "${TEST_CASES[@]}"; do
        IFS='|' read -r test_name endpoint request_data expected_field <<< "$test_case"
        
        # Skip if test already passed (check if test name is in PASSED_TESTS string)
        if [[ $PASSED_TESTS == *"$test_name"* ]]; then
            echo -e "${GREEN}Skipping '$test_name' (already passed)${NC}"
            continue
        fi
        
        ALL_PASSED=false
        if run_test_case "$test_name" "$endpoint" "$request_data" "$expected_field"; then
            # Add test name to passed tests list
            PASSED_TESTS="$PASSED_TESTS $test_name"
        else
            let FAILED_TESTS=FAILED_TESTS+1
            echo -e "${RED}Test case '$test_name' failed${NC}"
            exit 1
        fi
    done
    
    # If all tests have passed, exit early
    if [ "$ALL_PASSED" = true ]; then
        echo -e "${GREEN}All tests have passed successfully! No more iterations needed.${NC}"
        exit 0
    fi

    let CNT=CNT+1
    
    if [ $CNT -lt $ITER ]; then
        echo "Sleeping for $SLEEP seconds before next iteration..."
        sleep $SLEEP
    fi
done

# Check final status
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "\n${RED}ERROR: Some tests failed. Failed tests: $FAILED_TESTS${NC}"
    exit 1
elif [ -z "$PASSED_TESTS" ]; then
    echo -e "\n${RED}ERROR: No tests were run successfully${NC}"
    exit 1
else
    echo -e "\n${GREEN}All API regression tests completed successfully!${NC}"
    echo "Passed tests: $PASSED_TESTS"
    exit 0
fi
