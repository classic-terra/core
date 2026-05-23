package ante_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/classic-terra/core/v4/custom/auth/ante"
	core "github.com/classic-terra/core/v4/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// ReplacementTracker — pure unit tests (no app needed)
// ---------------------------------------------------------------------------

func TestReplacementTracker(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	const sender = "terra1abc"
	orig := []byte("new-tx-bytes")

	require.Nil(t, tracker.Get(sender))

	tracker.Set(sender, 432, orig)

	info := tracker.Get(sender)
	require.NotNil(t, info)
	require.Equal(t, uint64(432), info.FromSequence)
	require.True(t, info.Contains(orig))

	// Set must store a copy; mutating the slice must not affect stored bytes.
	origCopy := []byte("new-tx-bytes")
	orig[0] = 'X'
	require.True(t, info.Contains(origCopy)) // original bytes still registered
	require.False(t, info.Contains(orig))    // mutated bytes are not

	tracker.Clear(sender)
	require.Nil(t, tracker.Get(sender))

	// Clear on unknown sender is a no-op.
	tracker.Clear("unknown")

	// Multiple rapid replacements at the same sequence must all be registered so
	// the first replacement is not evicted when a second arrives.
	rep1 := []byte("replacement-1")
	rep2 := []byte("replacement-2")
	tracker.Set(sender, 5, rep1)
	tracker.Set(sender, 5, rep2)
	info = tracker.Get(sender)
	require.NotNil(t, info)
	require.True(t, info.Contains(rep1))
	require.True(t, info.Contains(rep2))

	// A new sequence must reset the set entirely.
	tracker.Set(sender, 6, rep1)
	info = tracker.Get(sender)
	require.Equal(t, uint64(6), info.FromSequence)
	require.True(t, info.Contains(rep1))
	require.False(t, info.Contains(rep2))

	// RemoveTxBytes: removing one of two entries keeps the sender key alive
	// so the original stuck tx is still evicted on the next recheck round.
	tracker.Set(sender, 7, rep1)
	tracker.Set(sender, 7, rep2)
	tracker.RemoveTxBytes(sender, rep1)
	info = tracker.Get(sender)
	require.NotNil(t, info, "sender entry must survive after first of two replacements rechecks")
	require.False(t, info.Contains(rep1))
	require.True(t, info.Contains(rep2))

	// Removing the last entry must delete the whole sender key.
	tracker.RemoveTxBytes(sender, rep2)
	require.Nil(t, tracker.Get(sender))

	// RemoveTxBytes on an unknown sender or unknown bytes is a no-op.
	tracker.RemoveTxBytes("unknown", rep1)
	tracker.Set(sender, 8, rep1)
	tracker.RemoveTxBytes(sender, rep2) // rep2 not in set — must not panic
	require.NotNil(t, tracker.Get(sender))
}

// ---------------------------------------------------------------------------
// TxReplacementDecorator — behaviour tests
// ---------------------------------------------------------------------------

type ReplacementDecoratorSuite struct {
	AnteTestSuite
}

func TestReplacementDecoratorSuite(t *testing.T) {
	suite.Run(t, new(ReplacementDecoratorSuite))
}

func noopHandler(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}

func (s *ReplacementDecoratorSuite) makeDecorator(tracker *ante.ReplacementTracker) ante.TxReplacementDecorator {
	return ante.NewTxReplacementDecorator(s.app.AccountKeeper, s.app.CommitMultiStore(), tracker)
}

func (s *ReplacementDecoratorSuite) makeTxAtSeq(priv cryptotypes.PrivKey, accNum, seq uint64) sdk.Tx {
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(
		testdata.NewTestMsg(sdk.AccAddress(priv.PubKey().Address())),
	))
	s.txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{accNum}, []uint64{seq}, s.ctx.ChainID())
	s.Require().NoError(err)
	return tx
}

func (s *ReplacementDecoratorSuite) makeBankSendTxAtSeqWithMemo(priv cryptotypes.PrivKey, accNum, seq uint64, memo string) sdk.Tx {
	addr := sdk.AccAddress(priv.PubKey().Address())
	s.clientCtx = s.clientCtx.WithTxConfig(s.app.GetTxConfig())
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(
		banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1)))),
	))
	s.txBuilder.SetMemo(memo)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1_000))))
	s.txBuilder.SetGasLimit(200000)
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{accNum}, []uint64{seq}, s.ctx.ChainID())
	s.Require().NoError(err)
	return tx
}

func (s *ReplacementDecoratorSuite) checkTx(tx sdk.Tx, expectedCode uint32) *abci.ResponseCheckTx {
	return s.checkTxWithType(tx, abci.CheckTxType_New, expectedCode)
}

func (s *ReplacementDecoratorSuite) checkTxWithType(tx sdk.Tx, checkType abci.CheckTxType, expectedCode uint32) *abci.ResponseCheckTx {
	txBytes, err := s.clientCtx.TxConfig.TxEncoder()(tx)
	s.Require().NoError(err)

	resp, err := s.app.CheckTx(&abci.RequestCheckTx{
		Tx:   txBytes,
		Type: checkType,
	})
	s.Require().NoError(err)
	s.Require().Equal(expectedCode, resp.Code, "CheckTx log: %s", resp.Log)
	return resp
}

func (s *ReplacementDecoratorSuite) selectedMempoolSequences(mp mempool.Mempool) []uint64 {
	itr := mp.Select(s.ctx, nil)
	var sequences []uint64
	for itr != nil {
		tx := itr.Tx()
		sigTx, ok := tx.(authsigning.SigVerifiableTx)
		s.Require().True(ok)
		sigs, err := sigTx.GetSignaturesV2()
		s.Require().NoError(err)
		s.Require().NotEmpty(sigs)
		sequences = append(sequences, sigs[0].Sequence)
		itr = itr.Next()
	}
	return sequences
}

func (s *ReplacementDecoratorSuite) selectedMempoolMemos(mp mempool.Mempool) []string {
	itr := mp.Select(s.ctx, nil)
	var memos []string
	for itr != nil {
		tx := itr.Tx()
		memoTx, ok := tx.(sdk.TxWithMemo)
		s.Require().True(ok)
		memos = append(memos, memoTx.GetMemo())
		itr = itr.Next()
	}
	return memos
}

// ---------------------------------------------------------------------------
// handleRecheck tests (ctx.IsReCheckTx = true)
// ---------------------------------------------------------------------------

// No tracker entry → decorator is transparent.
func (s *ReplacementDecoratorSuite) TestRecheck_NoTrackerEntry() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	dec := s.makeDecorator(ante.NewReplacementTracker())
	tx := s.makeTxAtSeq(priv, accNum, 5)

	_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), tx, false, noopHandler)
	s.Require().NoError(err)
}

// Old stuck tx at seq == fromSequence must be evicted (ErrWrongSequence).
func (s *ReplacementDecoratorSuite) TestRecheck_OldTxAtFromSequenceIsEvicted() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		s.Require().NoError(acc.SetSequence(432))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	tracker := ante.NewReplacementTracker()
	tracker.Set(sdk.AccAddress(priv.PubKey().Address()).String(), 432, []byte("replacement-bytes"))
	dec := s.makeDecorator(tracker)

	oldTx := s.makeTxAtSeq(priv, accNum, 432)
	_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), oldTx, false, noopHandler)
	s.Require().ErrorIs(err, sdkerrors.ErrWrongSequence)
}

// The replacement tx itself (bytes match stored entry) must pass and clear the tracker.
func (s *ReplacementDecoratorSuite) TestRecheck_ReplacementTxClearsTracker() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		s.Require().NoError(acc.SetSequence(432))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	replacementTx := s.makeTxAtSeq(priv, accNum, 432)
	txBytes, err := s.clientCtx.TxConfig.TxEncoder()(replacementTx)
	s.Require().NoError(err)

	sender := sdk.AccAddress(priv.PubKey().Address()).String()
	tracker := ante.NewReplacementTracker()
	tracker.Set(sender, 432, txBytes)
	dec := s.makeDecorator(tracker)

	ctx := s.ctx.WithIsReCheckTx(true).WithTxBytes(txBytes)
	_, err = dec.AnteHandle(ctx, replacementTx, false, noopHandler)
	s.Require().NoError(err)
	s.Require().Nil(tracker.Get(sender), "tracker entry must be cleared once replacement tx passes recheck")
}

// Txs at seq > fromSequence must NOT be evicted — they are valid queued txs that
// will be drained naturally as each block advances the committed sequence.
// This is the key behaviour introduced by the bug fix in handleRecheck.
func (s *ReplacementDecoratorSuite) TestRecheck_SubsequentTxsPassThrough() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		s.Require().NoError(acc.SetSequence(433))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	tracker := ante.NewReplacementTracker()
	// Replacement recorded at seq 432; the txs under test are at 433, 434.
	tracker.Set(sdk.AccAddress(priv.PubKey().Address()).String(), 432, []byte("replacement-bytes"))
	dec := s.makeDecorator(tracker)

	for _, seq := range []uint64{433, 434, 441} {
		tx := s.makeTxAtSeq(priv, accNum, seq)
		_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), tx, false, noopHandler)
		s.Require().NoError(err, "subsequent tx at seq=%d must pass through, not be evicted", seq)
	}
}

// Txs at seq < fromSequence are unaffected by the tracker.
func (s *ReplacementDecoratorSuite) TestRecheck_SeqBeforeFromSequencePassesThrough() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		s.Require().NoError(acc.SetSequence(430))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	tracker := ante.NewReplacementTracker()
	tracker.Set(sdk.AccAddress(priv.PubKey().Address()).String(), 432, []byte("replacement-bytes"))
	dec := s.makeDecorator(tracker)

	tx := s.makeTxAtSeq(priv, accNum, 430)
	_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), tx, false, noopHandler)
	s.Require().NoError(err)
}

// ---------------------------------------------------------------------------
// handleNewTx tests (ctx.IsCheckTx = true)
// ---------------------------------------------------------------------------

// Normal tx (txSeq >= pendingSeq) passes through without touching the tracker.
func (s *ReplacementDecoratorSuite) TestNewTx_NormalSequencePassThrough() {
	s.SetupTest(true)
	priv, addr, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, a := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, a)
		s.Require().NoError(acc.SetSequence(432))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, a, acc.GetAccountNumber()
	}()

	tracker := ante.NewReplacementTracker()
	dec := s.makeDecorator(tracker)

	tx := s.makeTxAtSeq(priv, accNum, 432)
	_, err := dec.AnteHandle(s.ctx.WithIsCheckTx(true), tx, false, noopHandler)
	s.Require().NoError(err)
	s.Require().Nil(tracker.Get(addr.String()), "tracker must stay empty for a normal (non-replacement) tx")
}

// Replacement tx: txSeq == committedSeq < pendingSeq.
// The decorator must reset the account to committed state, set the tracker entry,
// and allow the tx through so the downstream ante chain can verify it.
func (s *ReplacementDecoratorSuite) TestNewTx_ReplacementAtCommittedSeq() {
	s.SetupTest(true)
	priv, _, addr := testdata.KeyTestPubAddr()

	// Write the account with seq=432 directly into the committed store.
	// ctx.WithMultiStore(CommitMultiStore) gives a context that reads/writes
	// the committed (non-cached) store, mirroring what getCommittedAccount does.
	acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
	s.Require().NoError(acc.SetSequence(432))
	committedCtx := s.ctx.WithMultiStore(s.app.CommitMultiStore())
	s.app.AccountKeeper.SetAccount(committedCtx, acc)

	// Advance the account in the test (pending) context to seq=443,
	// simulating 11 queued txs (432-442) already accepted into the mempool.
	s.Require().NoError(acc.SetSequence(443))
	s.app.AccountKeeper.SetAccount(s.ctx, acc)

	tracker := ante.NewReplacementTracker()
	dec := s.makeDecorator(tracker)

	replacementTx := s.makeTxAtSeq(priv, acc.GetAccountNumber(), 432)
	txBytes, err := s.clientCtx.TxConfig.TxEncoder()(replacementTx)
	s.Require().NoError(err)

	ctx := s.ctx.WithIsCheckTx(true).WithTxBytes(txBytes)
	_, err = dec.AnteHandle(ctx, replacementTx, false, noopHandler)
	s.Require().NoError(err)

	sender := addr.String()
	info := tracker.Get(sender)
	s.Require().NotNil(info, "tracker must record the replacement")
	s.Require().Equal(uint64(432), info.FromSequence)
	s.Require().True(info.Contains(txBytes))

	// The account in ctx must have been reset to committed seq=432 so that the
	// downstream SigVerificationDecorator can pass.
	resetAcc := s.app.AccountKeeper.GetAccount(ctx, addr)
	s.Require().Equal(uint64(432), resetAcc.GetSequence())
}

// txSeq < pendingSeq but txSeq != committedSeq → not a valid replacement,
// decorator must pass through and leave tracker untouched.
func (s *ReplacementDecoratorSuite) TestNewTx_BelowPendingButNotAtCommitted() {
	s.SetupTest(true)
	priv, _, addr := testdata.KeyTestPubAddr()

	// Committed seq = 432.
	acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
	s.Require().NoError(acc.SetSequence(432))
	committedCtx := s.ctx.WithMultiStore(s.app.CommitMultiStore())
	s.app.AccountKeeper.SetAccount(committedCtx, acc)

	// Pending seq = 443.
	s.Require().NoError(acc.SetSequence(443))
	s.app.AccountKeeper.SetAccount(s.ctx, acc)

	tracker := ante.NewReplacementTracker()
	dec := s.makeDecorator(tracker)

	// Tx at seq=430: below pending (443) but also below committed (432) → not a replacement.
	tx := s.makeTxAtSeq(priv, acc.GetAccountNumber(), 430)
	_, err := dec.AnteHandle(s.ctx.WithIsCheckTx(true), tx, false, noopHandler)
	s.Require().NoError(err)
	s.Require().Nil(tracker.Get(addr.String()), "tracker must stay empty — txSeq 430 != committedSeq 432")
}

// This reproduces the production failure mode through the real TerraApp
// CheckTx path: queued txs advance the local CheckTx account sequence to 443
// while the committed account remains at 432. Before TxReplacementDecorator,
// another tx at sequence 432 failed with "expected 443, got 432"; now it is
// accepted as a replacement for the stuck committed sequence.
func (s *ReplacementDecoratorSuite) TestCheckTx_ReplacesCommittedSequenceAfterPendingSequenceAdvanced() {
	s.SetupTest(true)
	priv, _, addr := testdata.KeyTestPubAddr()

	committedCtx := s.ctx.WithMultiStore(s.app.CommitMultiStore())
	acc := s.app.AccountKeeper.NewAccountWithAddress(committedCtx, addr)
	s.Require().NoError(acc.SetSequence(432))
	s.app.AccountKeeper.SetAccount(committedCtx, acc)
	s.Require().NoError(banktestutil.FundAccount(
		committedCtx,
		s.app.BankKeeper,
		addr,
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1_000_000_000))),
	))

	var originalSeq432Tx sdk.Tx
	for seq := uint64(432); seq <= 442; seq++ {
		tx := s.makeBankSendTxAtSeqWithMemo(priv, acc.GetAccountNumber(), seq, "queued-tx")
		if seq == 432 {
			originalSeq432Tx = tx
		}
		s.checkTx(tx, 0)
	}
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)

	replacementTx := s.makeBankSendTxAtSeqWithMemo(priv, acc.GetAccountNumber(), 432, "replacement-at-committed-sequence")
	s.checkTx(replacementTx, 0)
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)
	s.Require().Equal("replacement-at-committed-sequence", s.selectedMempoolMemos(s.app.Mempool())[0])

	recheckResp := s.checkTxWithType(originalSeq432Tx, abci.CheckTxType_Recheck, sdkerrors.ErrWrongSequence.ABCICode())
	s.Require().Contains(recheckResp.Log, "tx superseded")
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)
	s.Require().Equal("replacement-at-committed-sequence", s.selectedMempoolMemos(s.app.Mempool())[0])

	s.checkTxWithType(replacementTx, abci.CheckTxType_Recheck, 0)
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)
	s.Require().Equal("replacement-at-committed-sequence", s.selectedMempoolMemos(s.app.Mempool())[0])
}
