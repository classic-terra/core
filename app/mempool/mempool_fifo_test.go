package mempool_test

import (
	"fmt"
	"github.com/classic-terra/core/v3/custom/auth/ante"
	oracleexported "github.com/classic-terra/core/v3/x/oracle/exported"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"math/rand"
	"testing"

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
			order: []int{3, 4, 2, 1, 0},
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
			order: []int{3, 4, 5, 0, 1, 2},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 21, n: 4, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 20, n: 1, a: sa},
			},
			order: []int{1, 2, 0},
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
			order: []int{3, 4, 2, 1, 0},
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
			order: []int{3, 4, 2, 1, 0},
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
			order: []int{4, 1, 6, 3, 2, 0, 5},
			seed:  0,
		},
		{
			txs: []txSpec{
				{p: 6, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sb},
			},
			order: []int{2, 3, 0, 1},
			seed:  0,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := appmempool.NewFifoSenderNonceMempool()
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
		txs   []testTx
		order []int
		fail  bool
	}{
		{
			txs: []testTx{
				{
					id:       1,
					priority: 1,
					nonce:    1,
					msgs:     []sdk.Msg{&banktypes.MsgSend{}},
					address:  sa,
				},
				{
					id:       2,
					priority: 1,
					nonce:    1,
					msgs:     []sdk.Msg{&oracleexported.MsgAggregateExchangeRateVote{}},
					address:  sb,
				},
			},
			order: []int{1, 0},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := appmempool.NewFifoSenderNonceMempool()
			// create test txs and insert into mempool
			for i, ts := range tt.txs {
				tx := testTx{id: i, priority: ts.priority, nonce: uint64(ts.nonce), address: ts.address, msgs: ts.msgs}
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

func (s *MempoolTestSuite) TestOracleTx() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 3)
	mp := appmempool.NewFifoSenderNonceMempool(appmempool.SenderNonceMaxTxOpt(3))

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
	}
	require.Equal(t, 3, len(orderedTxs))

	require.True(t, ante.IsOracleTx(orderedTxs[0].GetMsgs()))
	require.True(t, ante.IsOracleTx(orderedTxs[1].GetMsgs()))
	require.False(t, ante.IsOracleTx(orderedTxs[2].GetMsgs()))
}

func (s *MempoolTestSuite) TestMaxTx() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 1)
	mp := appmempool.NewFifoSenderNonceMempool(appmempool.SenderNonceMaxTxOpt(1))

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
	mp := appmempool.NewFifoSenderNonceMempool()

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
