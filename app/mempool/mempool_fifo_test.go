package mempool_test

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/classic-terra/core/v3/custom/auth/ante"
	oracleexported "github.com/classic-terra/core/v3/x/oracle/exported"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	appmempool "github.com/classic-terra/core/v3/app/mempool"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func (s *MempoolTestSuite) TestTxOrder() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 5)
	sa := accounts[0].Address
	sb := accounts[1].Address

	tests := []struct {
		txs   []txSpec
		order []int
		fail  bool
		seed  int64
	}{
		{
			txs: []txSpec{
				{p: 21, n: 4, a: sa},
				{p: 8, n: 3, a: sa},
				{p: 6, n: 2, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 20, n: 1, a: sa},
			},
			order: []int{0, 1, 2, 3, 4},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 3, n: 0, a: sa},
				{p: 5, n: 1, a: sa},
				{p: 9, n: 2, a: sa},
				{p: 6, n: 0, a: sb},
				{p: 5, n: 1, a: sb},
				{p: 8, n: 2, a: sb},
			},
			order: []int{0, 1, 2, 3, 4, 5},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 21, n: 4, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 20, n: 1, a: sa},
			},
			order: []int{0, 1, 2},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 50, n: 3, a: sa},
				{p: 30, n: 2, a: sa},
				{p: 10, n: 1, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 21, n: 2, a: sb},
			},
			order: []int{0, 1, 2, 3, 4},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 50, n: 3, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 99, n: 1, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 8, n: 2, a: sb},
			},
			order: []int{0, 1, 2, 3, 4},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 30, a: sa, n: 2},
				{p: 20, a: sb, n: 1},
				{p: 15, a: sa, n: 1},
				{p: 10, a: sa, n: 0},
				{p: 8, a: sb, n: 0},
				{p: 6, a: sa, n: 3},
				{p: 4, a: sb, n: 3},
			},
			order: []int{0, 1, 2, 3, 4, 5, 6},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 6, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sb},
			},
			order: []int{0, 1, 2, 3},
			seed:  0,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := appmempool.NewFifoMempool()
			// create test txs and insert into mempool
			for i, ts := range tt.txs {
				tx := testTx{id: i, priority: int64(ts.p), nonce: uint64(ts.n), address: ts.a}
				c := ctx.WithPriority(tx.priority)
				err := pool.Insert(c, tx)
				require.NoError(t, err)
			}

			itr := pool.Select(ctx, nil)
			orderedTxs := fetchTxs(itr, 1000)
			var txOrder []int
			for _, tx := range orderedTxs {
				txOrder = append(txOrder, tx.(testTx).id)
			}
			for _, tx := range orderedTxs {
				require.NoError(t, pool.Remove(tx))
			}
			require.Equal(t, tt.order, txOrder)
			require.Equal(t, 0, pool.CountTx())
		})
	}
}

func (s *MempoolTestSuite) TestTxOrderWithOracle() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 5)
	sa := accounts[0].Address
	sb := accounts[1].Address

	tests := []struct {
		txs            []testTx
		numberTxOracle int
		fail           bool
	}{
		{
			txs: []testTx{
				{
					id:      1,
					nonce:   1,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sa,
				},
				{
					id:      2,
					nonce:   1,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address: sb,
				},
			},
			numberTxOracle: 1,
		},
		{
			// Test multiple oracle txs vs regular txs
			txs: []testTx{
				{
					id:      0,
					nonce:   1,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sa,
				},
				{
					id:      1,
					nonce:   1,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address: sb,
				},
				{
					id:      2,
					nonce:   2,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address: sa,
				},
				{
					id:      3,
					nonce:   2,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sb,
				},
			},
			numberTxOracle: 2,
		},
		{
			// Test oracle tx priority with different nonces
			txs: []testTx{
				{
					id:      0,
					nonce:   5,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sa,
				},
				{
					id:      1,
					nonce:   10,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address: sa,
				},
				{
					id:      2,
					nonce:   1,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sa,
				},
			},
			numberTxOracle: 1,
		},
		{
			// Test multiple oracle txs from different senders
			txs: []testTx{
				{
					id:      0,
					nonce:   2,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sb,
				},
				{
					id:      1,
					nonce:   2,
					msgs:    []sdk.Msg{&banktypes.MsgSend{}},
					address: sa,
				},
				{
					id:      2,
					nonce:   1,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address: sa,
				},
				{
					id:      3,
					nonce:   1,
					msgs:    []sdk.Msg{&oracleexported.MsgAggregateExchangeRatePrevote{}},
					address: sb,
				},
			},
			numberTxOracle: 2,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := appmempool.NewFifoMempool()
			// create test txs and insert into mempool
			for i, ts := range tt.txs {
				tx := testTx{id: i, nonce: ts.nonce, address: ts.address, msgs: ts.msgs}
				err := pool.Insert(ctx, tx)
				require.NoError(t, err)
			}

			itr := pool.Select(ctx, nil)
			orderedTxs := fetchTxs(itr, 1000)
			numberTxOracle := 0
			for _, tx := range orderedTxs {
				if ante.IsOracleTx(tx.GetMsgs()) {
					numberTxOracle += +1
				} else {
					break
				}
			}
			for _, tx := range orderedTxs {
				require.NoError(t, pool.Remove(tx))
			}
			require.Equal(t, tt.numberTxOracle, numberTxOracle)
			require.Equal(t, 0, pool.CountTx())
		})
	}
}

func (s *MempoolTestSuite) TestOracleTx() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 3)
	mp := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(3))

	tx := testTx{
		id:       0,
		nonce:    2,
		address:  accounts[0].Address,
		priority: rand.Int63(),
	}
	tx1 := testTx{
		id:       1,
		nonce:    1,
		address:  accounts[0].Address,
		priority: rand.Int63(),
		msgs: []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{
			Salt: "1",
		}},
	}
	tx2 := testTx{
		id:       2,
		nonce:    1,
		address:  accounts[2].Address,
		priority: rand.Int63(),
		msgs: []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{
			Salt: "2",
		}},
	}

	// empty mempool behavior
	require.Equal(t, 0, s.mempool.CountTx())
	itr := mp.Select(ctx, nil)
	require.Nil(t, itr)
	err := mp.Insert(ctx, tx)
	require.NoError(t, err)
	err = mp.Insert(ctx, tx1)
	require.NoError(t, err)
	err = mp.Insert(ctx, tx2)
	require.NoError(t, err)

	itr = mp.Select(ctx, nil)
	orderedTxs := fetchTxs(itr, 1000)
	for _, tmpTx := range orderedTxs {
		fmt.Println(tmpTx.GetMsgs())
		err = mp.Remove(tmpTx)
		require.NoError(t, err)
	}
	require.Equal(t, 3, len(orderedTxs))

	require.True(t, ante.IsOracleTx(orderedTxs[0].GetMsgs()))
	require.True(t, ante.IsOracleTx(orderedTxs[1].GetMsgs()))
	require.False(t, ante.IsOracleTx(orderedTxs[2].GetMsgs()))

	require.Equal(t, 0, mp.CountTx())
}

func (s *MempoolTestSuite) TestMaxTx() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 1)
	mp := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(1))

	tx := testTx{
		nonce:    0,
		address:  accounts[0].Address,
		priority: rand.Int63(),
	}
	tx2 := testTx{
		nonce:    1,
		address:  accounts[0].Address,
		priority: rand.Int63(),
	}

	// empty mempool behavior
	require.Equal(t, 0, s.mempool.CountTx())
	itr := mp.Select(ctx, nil)
	require.Nil(t, itr)

	ctx = ctx.WithPriority(tx.priority)
	err := mp.Insert(ctx, tx)
	require.NoError(t, err)
	ctx = ctx.WithPriority(tx.priority)
	err = mp.Insert(ctx, tx2)
	require.Equal(t, mempool.ErrMempoolTxMaxCapacity, err)
}

func (s *MempoolTestSuite) TestTxNotFoundOnSender() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 1)
	mp := appmempool.NewFifoMempool()
	txSender := testTx{
		nonce:    0,
		address:  accounts[0].Address,
		priority: rand.Int63(),
	}

	tx := testTx{
		nonce:    1,
		address:  accounts[0].Address,
		priority: rand.Int63(),
	}

	ctx = ctx.WithPriority(tx.priority)
	err := mp.Insert(ctx, txSender)
	require.NoError(t, err)
	err = mp.Remove(tx)
	require.Equal(t, mempool.ErrTxNotFound, err)
}

func (s *MempoolTestSuite) TestBatchTx_WhenEnoughMemPool() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 10)
	mp := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(150))

	// Create 150 transactions (50 each of oracle, send, and staking)
	var allTxs []testTx

	// Create 50 oracle transactions (25 votes and 25 prevotes)
	for i := 0; i < 50; i++ {
		var msg sdk.Msg
		if i%2 == 0 {
			msg = &oracleexported.MsgAggregateExchangeRateVote{
				Salt: fmt.Sprintf("salt_vote_%d", i),
			}
		} else {
			msg = &oracleexported.MsgAggregateExchangeRatePrevote{
				Hash: fmt.Sprintf("hash_prevote_%d", i),
			}
		}

		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{msg},
		}
		allTxs = append(allTxs, tx)
	}

	// Create 50 send transactions
	for i := 50; i < 100; i++ {
		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{&banktypes.MsgSend{}},
		}
		allTxs = append(allTxs, tx)
	}

	// Create 50 staking transactions
	for i := 100; i < 150; i++ {
		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{&stakingtypes.MsgDelegate{}},
		}
		allTxs = append(allTxs, tx)
	}

	// Shuffle the transactions
	rand.Shuffle(len(allTxs), func(i, j int) {
		allTxs[i], allTxs[j] = allTxs[j], allTxs[i]
	})

	// Insert transactions into mempool
	for _, tx := range allTxs {
		err := mp.Insert(ctx, tx)
		require.NoError(t, err)
	}

	// Verify mempool size
	require.Equal(t, 150, mp.CountTx())

	// Get ordered transactions
	itr := mp.Select(ctx, nil)
	orderedTxs := fetchTxs(itr, 1000)
	require.Equal(t, 150, len(orderedTxs))

	// Count oracle transactions in first batch
	oracleCount := 0
	for _, tx := range orderedTxs[:50] {
		if ante.IsOracleTx(tx.GetMsgs()) {
			oracleCount++
		}
	}

	// Verify oracle transactions are prioritized
	require.True(t, oracleCount == 50, "Expected majority of first 50 transactions to be oracle transactions")

	// Cleanup
	for _, tx := range orderedTxs {
		require.NoError(t, mp.Remove(tx))
	}
	require.Equal(t, 0, mp.CountTx())
}

func (s *MempoolTestSuite) TestBatchTx_WhenNotEnoughMemPool() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 10)
	maxMempoolSize := 100
	mp := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(maxMempoolSize))

	// Create 150 transactions (50 each of oracle, send, and staking)
	var allTxs []testTx

	// Create 50 oracle transactions (25 votes and 25 prevotes)
	for i := 0; i < 50; i++ {
		var msg sdk.Msg
		if i%2 == 0 {
			msg = &oracleexported.MsgAggregateExchangeRateVote{
				Salt: fmt.Sprintf("salt_vote_%d", i),
			}
		} else {
			msg = &oracleexported.MsgAggregateExchangeRatePrevote{
				Hash: fmt.Sprintf("hash_prevote_%d", i),
			}
		}

		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{msg},
		}
		allTxs = append(allTxs, tx)
	}

	// Create 50 send transactions
	for i := 50; i < 100; i++ {
		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{&banktypes.MsgSend{}},
		}
		allTxs = append(allTxs, tx)
	}

	// Create 50 staking transactions
	for i := 100; i < 150; i++ {
		tx := testTx{
			id:       i,
			nonce:    uint64(i),
			address:  accounts[rand.Intn(len(accounts))].Address,
			priority: rand.Int63(),
			msgs:     []sdk.Msg{&stakingtypes.MsgDelegate{}},
		}
		allTxs = append(allTxs, tx)
	}

	// Shuffle the transactions
	rand.Shuffle(len(allTxs), func(i, j int) {
		allTxs[i], allTxs[j] = allTxs[j], allTxs[i]
	})

	// Insert transactions into mempool
	i := 0
	for _, tx := range allTxs {
		err := mp.Insert(ctx, tx)
		if i < maxMempoolSize {
			require.NoError(t, err)
		} else {
			require.Equal(t, mempool.ErrMempoolTxMaxCapacity, err)
		}
		i += 1
	}

	// Verify mempool size
	require.Equal(t, 100, mp.CountTx())

	// Get ordered transactions
	itr := mp.Select(ctx, nil)
	orderedTxs := fetchTxs(itr, 1000)
	require.Equal(t, 100, len(orderedTxs))

	// Verify oracle transactions come first, followed by regular transactions
	var lastOracleIndex = -1
	var firstRegularIndex = -1

	for i, tx := range orderedTxs {
		if ante.IsOracleTx(tx.GetMsgs()) {
			lastOracleIndex = i
			// If we've already seen a regular transaction, this is an error
			require.Equal(t, -1, firstRegularIndex,
				"Found oracle tx after regular tx at index %d", i)
		} else {
			if firstRegularIndex == -1 {
				firstRegularIndex = i
			}
		}
	}

	// Verify oracle transactions are prioritized
	require.True(t, lastOracleIndex < firstRegularIndex, "Expected majority oracle transactions come first")

	// Cleanup
	for _, tx := range orderedTxs {
		require.NoError(t, mp.Remove(tx))
	}
	require.Equal(t, 0, mp.CountTx())
}

func BenchmarkMempool(b *testing.B) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 10)
	maxMempoolSize := 1000

	benchmarks := []struct {
		name string
		size int
	}{
		{"Small-100", 100},
		{"Medium-500", 500},
		{"Large-1000", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Reset timer for setup
			b.StopTimer()

			var allTxs []testTx
			// Create mixed transaction types (oracle, send, staking)
			for i := 0; i < bm.size; i++ {
				var msg sdk.Msg
				switch i % 3 {
				case 0:
					msg = &oracleexported.MsgAggregateExchangeRateVote{
						Salt: fmt.Sprintf("salt_vote_%d", i),
					}
				case 1:
					msg = &banktypes.MsgSend{}
				case 2:
					msg = &stakingtypes.MsgDelegate{}
				}

				tx := testTx{
					id:       i,
					nonce:    uint64(i),
					address:  accounts[rand.Intn(len(accounts))].Address,
					priority: rand.Int63(),
					msgs:     []sdk.Msg{msg},
				}
				allTxs = append(allTxs, tx)
			}

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				mp := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(maxMempoolSize))

				// Benchmark insertion
				for _, tx := range allTxs {
					err := mp.Insert(ctx, tx)
					if err != nil && !errors.Is(err, mempool.ErrMempoolTxMaxCapacity) {
						b.Fatal(err)
					}
				}

				// Benchmark selection
				itr := mp.Select(ctx, nil)
				orderedTxs := fetchTxs(itr, int64(bm.size))

				// Benchmark removal
				for _, tx := range orderedTxs {
					if err := mp.Remove(tx); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
