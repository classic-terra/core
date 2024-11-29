#!/bin/bash

# Step 1: Run old chain --> Sub Proposal Upgrade (soft-upgrade)
# Step 2: Vote --> Proposal Passed
# Step 3: Stop old chain --> Switch new code --> Build --> Run new chain 
# Step 4: Test Submit Proposal New Flow with MinUstc Deposit


HOME_DIR=mytestnet

./build/terrad tx gov submit-legacy-proposal software-upgrade v10_1 --upgrade-height 20 --upgrade-info v10_1 --title "upgrade" --description "upgrade" --no-validate --deposit "100000000uluna" --from test0 --keyring-backend test --home mytestnet -y

./build/terrad tx gov vote 1 yes --from test0 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 1 yes --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 1 yes --from test2 --home mytestnet --keyring-backend test -y

./build/terrad q gov proposal 1

./build/terrad tx gov submit-proposal ./scripts/gov_test/draft_proposal.json --from test0 --home mytestnet --keyring-backend test -y

./build/terrad tx gov deposit 6 "682221290668uluna" --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov deposit 6 "1uluna" --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test0 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test2 --home mytestnet --keyring-backend test -y
./build/terrad q gov proposal 6
./build/terrad q gov min-deposit 6
