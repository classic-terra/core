HOME_DIR=mytestnet

./build/terrad query oracle params --home mytestnet
./build/terrad query gov params --home mytestnet


./build/terrad tx oracle aggregate-prevote 1234 0.1uusd --from test0 --home mytestnet --keyring-backend test -y

./build/terrad tx oracle aggregate-vote 1234 0.1uusd terravaloper1dymsken93a500fak5f6ye8duw9w23yarlyuj82 --from test0 --home mytestnet --keyring-backend test -y

./build/terrad query oracle exchange-rates uusd --home mytestnet

./build/terrad tx gov submit-proposal ./scripts/gov_test/proposal.json --from test0 --home mytestnet --keyring-backend test -y

./build/terrad tx gov deposit 1 "20000000uluna" --from test0 --home mytestnet --keyring-backend test -y

./build/terrad q gov proposal 1
./build/terrad q gov min-deposit 1

# 1220000000
# 5000000000

./build/terrad query oracle params --node https://terra-classic-rpc.publicnode.com:443
./build/terrad query oracle params --home mytestnet

# Get proposal id
./build/terrad query gov proposal 1 --home mytestnet