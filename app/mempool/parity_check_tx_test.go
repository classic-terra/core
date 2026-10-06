package mempool_test

import (
	"errors"
	"math/rand"

	log "cosmossdk.io/log"
	appmempool "github.com/classic-terra/core/v4/app/mempool"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func (s *MempoolTestSuite) TestContains() {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 1)
	mp := appmempool.NewFifoMempool()

	tx := testTx{nonce: 3, address: accounts[0].Address}
	s.Require().False(mp.Contains(tx))

	s.Require().NoError(mp.Insert(ctx, tx))
	s.Require().True(mp.Contains(tx))
	s.Require().False(mp.Contains(testTx{nonce: 4, address: accounts[0].Address}))

	s.Require().NoError(mp.Remove(tx))
	s.Require().False(mp.Contains(tx))

	// A disabled mempool stores nothing, so it must not report txs as missing.
	disabled := appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(-1))
	s.Require().True(disabled.Contains(tx))
}

func (s *MempoolTestSuite) TestParityCheckTxHandler() {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 1)
	tx := testTx{nonce: 7, address: accounts[0].Address}
	txBytes := []byte("tx")

	decoder := func(bz []byte) (sdk.Tx, error) {
		if string(bz) != string(txBytes) {
			return nil, errors.New("undecodable")
		}
		return tx, nil
	}

	tests := []struct {
		name       string
		reqType    abci.CheckTxType
		txBytes    []byte
		inMempool  bool
		runTxErr   error
		expectRun  bool
		expectPass bool
	}{
		{"new tx not yet in mempool runs", abci.CheckTxType_New, txBytes, false, nil, true, true},
		{"recheck of tx in mempool runs", abci.CheckTxType_Recheck, txBytes, true, nil, true, true},
		{"recheck of tx missing from mempool is rejected", abci.CheckTxType_Recheck, txBytes, false, nil, false, false},
		{"recheck of undecodable tx is left to runTx", abci.CheckTxType_Recheck, []byte("garbage"), false, errors.New("decode"), true, false},
		{"runTx error is returned", abci.CheckTxType_Recheck, txBytes, true, errors.New("ante failed"), true, false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			mp := appmempool.NewFifoMempool()
			if tc.inMempool {
				s.Require().NoError(mp.Insert(ctx, tx))
			}

			ran := false
			runTx := func(_ []byte, _ sdk.Tx) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				ran = true
				if tc.runTxErr != nil {
					return sdk.GasInfo{GasWanted: 10, GasUsed: 5}, nil, nil, tc.runTxErr
				}
				return sdk.GasInfo{GasWanted: 10, GasUsed: 5}, &sdk.Result{}, nil, nil
			}

			handler := appmempool.NewParityCheckTxHandler(mp, decoder, func() bool { return false })
			res, err := handler(runTx, &abci.RequestCheckTx{Tx: tc.txBytes, Type: tc.reqType})
			s.Require().NoError(err)
			s.Require().Equal(tc.expectRun, ran)
			s.Require().Equal(tc.expectPass, res.Code == 0, res.Log)
		})
	}
}

// runTx stubs below mimic BaseApp.runTx: in CheckTx it inserts the tx into the
// app-side mempool after the ante handler and before the post handler; in
// ReCheckTx it removes the tx only when the ante handler fails.
func (s *MempoolTestSuite) TestParityCheckTxHandlerDropsFailedTxs() {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 2)
	tx := testTx{nonce: 7, address: accounts[0].Address}
	other := testTx{nonce: 3, address: accounts[1].Address}
	txBytes := []byte("tx")
	decoder := func(bz []byte) (sdk.Tx, error) {
		if string(bz) != string(txBytes) {
			return nil, errors.New("undecodable")
		}
		return tx, nil
	}
	ok := func() (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
		return sdk.GasInfo{GasWanted: 10, GasUsed: 5}, &sdk.Result{}, nil, nil
	}
	fail := func(msg string) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
		return sdk.GasInfo{GasWanted: 10, GasUsed: 5}, nil, nil, errors.New(msg)
	}

	tests := []struct {
		name          string
		reqType       abci.CheckTxType
		txInMempool   bool
		runTx         func(mp *appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error)
		expectPass    bool
		expectTxKept  bool
		expectMempool int // app mempool size afterwards, incl. the other sender's tx
	}{
		{
			name:    "new: post handler fails after insert -> tx removed again",
			reqType: abci.CheckTxType_New,
			runTx: func(mp *appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				s.Require().NoError(mp.Insert(ctx, tx))
				return fail("post handler: out of gas")
			},
			expectMempool: 1,
		},
		{
			name:    "new: ante fails before insert -> nothing removed",
			reqType: abci.CheckTxType_New,
			runTx: func(*appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				return fail("ante: insufficient fees")
			},
			expectMempool: 1,
		},
		{
			name:    "new: success keeps the inserted tx",
			reqType: abci.CheckTxType_New,
			runTx: func(mp *appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				s.Require().NoError(mp.Insert(ctx, tx))
				return ok()
			},
			expectPass:    true,
			expectTxKept:  true,
			expectMempool: 2,
		},
		{
			name:        "recheck: post handler fails -> tx removed from app mempool",
			reqType:     abci.CheckTxType_Recheck,
			txInMempool: true,
			runTx: func(*appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				return fail("post handler: out of gas")
			},
			expectMempool: 1,
		},
		{
			name:        "recheck: ante fails, runTx already removed it -> no error",
			reqType:     abci.CheckTxType_Recheck,
			txInMempool: true,
			runTx: func(mp *appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				s.Require().NoError(mp.Remove(tx))
				return fail("ante: account sequence mismatch")
			},
			expectMempool: 1,
		},
		{
			name:        "recheck: success keeps the tx",
			reqType:     abci.CheckTxType_Recheck,
			txInMempool: true,
			runTx: func(*appmempool.FifoMempool) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				return ok()
			},
			expectPass:    true,
			expectTxKept:  true,
			expectMempool: 2,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			mp := appmempool.NewFifoMempool()
			s.Require().NoError(mp.Insert(ctx, other))
			if tc.txInMempool {
				s.Require().NoError(mp.Insert(ctx, tx))
			}

			var passed sdk.Tx
			runTx := func(_ []byte, t sdk.Tx) (sdk.GasInfo, *sdk.Result, []abci.Event, error) {
				passed = t
				return tc.runTx(mp)
			}

			handler := appmempool.NewParityCheckTxHandler(mp, decoder, func() bool { return false })
			res, err := handler(runTx, &abci.RequestCheckTx{Tx: txBytes, Type: tc.reqType})
			s.Require().NoError(err)
			s.Require().Equal(tc.expectPass, res.Code == 0, res.Log)
			s.Require().Equal(tx, passed, "decoded tx must be passed to runTx")
			s.Require().Equal(tc.expectTxKept, mp.Contains(tx))
			s.Require().True(mp.Contains(other), "another sender's tx must never be touched")
			s.Require().Equal(tc.expectMempool, mp.CountTx())
		})
	}
}
