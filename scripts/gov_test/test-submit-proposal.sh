HOME_DIR=mytestnet
./build/terrad tx gov submit-proposal ./scripts/gov_test/draft_proposal.json --from test0 --home mytestnet --keyring-backend test -y

./build/terrad tx gov deposit 6 "682221290668uluna" --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov deposit 6 "1uluna" --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test0 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test1 --home mytestnet --keyring-backend test -y
./build/terrad tx gov vote 6 yes --from test2 --home mytestnet --keyring-backend test -y
./build/terrad q gov proposal 6
./build/terrad q gov min-deposit 6

564467876133
582221290668
56568876134