package ante_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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

func TestReplacementTracker_ClearIfPast(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	const sender = "terra1abc"
	tracker.Set(sender, 432, []byte("rep"))

	tracker.ClearIfPast(sender, 432)
	require.NotNil(t, tracker.Get(sender), "equal sequence must not clear")

	tracker.ClearIfPast(sender, 433)
	require.Nil(t, tracker.Get(sender), "advanced sequence must clear")
}

func TestReplacementTracker_PruneBySequence(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	tracker.SetAtHeight("terra1a", 10, []byte("a"), 100)
	tracker.SetAtHeight("terra1b", 20, []byte("b"), 100)

	seqs := map[string]uint64{
		"terra1a": 11, // past → prune
		"terra1b": 20, // equal → keep
	}
	tracker.Prune(func(sender string) (uint64, bool) {
		seq, ok := seqs[sender]
		return seq, ok
	}, 100) // same height → no TTL expiry

	require.Nil(t, tracker.Get("terra1a"))
	require.NotNil(t, tracker.Get("terra1b"))
	require.Equal(t, 1, tracker.Len())
}

func TestReplacementTracker_PruneByTTL(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	tracker.SetAtHeight("terra1a", 10, []byte("a"), 100)

	tracker.Prune(func(sender string) (uint64, bool) {
		return 10, true // sequence not advanced
	}, 201) // 201-100 > DefaultReplacementTTLBlocks(100)

	require.Nil(t, tracker.Get("terra1a"))
	require.Equal(t, 0, tracker.Len())
}

func TestReplacementTracker_MaxEntries(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	// Shrink the cap for the test via filling past default is expensive;
	// instead verify Len and that SetAtHeight of many distinct senders is bounded
	// by forcing eviction through the unexported path via repeated Sets under a
	// temporary tracker with the production default and checking Len() <= default.
	for i := 0; i < ante.DefaultMaxReplacementEntries+10; i++ {
		tracker.Set(fmt.Sprintf("terra1sender%d", i), uint64(i), []byte("x"))
	}
	require.LessOrEqual(t, tracker.Len(), ante.DefaultMaxReplacementEntries)
	require.Equal(t, ante.DefaultMaxReplacementEntries, tracker.Len())
}

func TestReplacementTracker_MaxTxBytesPerSender(t *testing.T) {
	tracker := ante.NewReplacementTracker()
	const sender = "terra1abc"

	var first, last []byte
	for i := 0; i < ante.DefaultMaxTxBytesPerSender+5; i++ {
		txBytes := []byte(fmt.Sprintf("replacement-%d", i))
		if i == 0 {
			first = txBytes
		}
		last = txBytes
		tracker.Set(sender, 432, txBytes)
	}

	info := tracker.Get(sender)
	require.NotNil(t, info)
	require.False(t, info.Contains(first), "oldest replacements must be evicted once per-sender cap is exceeded")
	require.True(t, info.Contains(last), "newest replacement must remain tracked")

	// Exactly the newest DefaultMaxTxBytesPerSender entries should remain.
	kept := 0
	for i := 0; i < ante.DefaultMaxTxBytesPerSender+5; i++ {
		if info.Contains([]byte(fmt.Sprintf("replacement-%d", i))) {
			kept++
		}
	}
	require.Equal(t, ante.DefaultMaxTxBytesPerSender, kept)
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

// The replacement tx itself (bytes match stored entry) must pass and KEEP the
// tracker entry: rechecks repeat every block until the replacement is included,
// and the queued txs above FromSequence rely on the live entry to survive.
// The entry is cleared by DeliverTx / BeginBlock prune once the sequence advances.
func (s *ReplacementDecoratorSuite) TestRecheck_ReplacementTxSuccessKeepsTracker() {
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
	info := tracker.Get(sender)
	s.Require().NotNil(info, "tracker entry must survive a successful recheck — it protects the queued txs until the replacement is included")
	s.Require().True(info.Contains(txBytes))

	// A second recheck round (next block, replacement still not included) must
	// behave identically.
	_, err = dec.AnteHandle(ctx, replacementTx, false, noopHandler)
	s.Require().NoError(err)
	s.Require().NotNil(tracker.Get(sender))
}

// Recheck failure must also drop the replacement bytes — CometBFT evicts the tx.
func (s *ReplacementDecoratorSuite) TestRecheck_ReplacementTxFailureClearsTracker() {
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

	failHandler := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, sdkerrors.ErrInsufficientFunds
	}
	ctx := s.ctx.WithIsReCheckTx(true).WithTxBytes(txBytes)
	_, err = dec.AnteHandle(ctx, replacementTx, false, failHandler)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	s.Require().Nil(tracker.Get(sender), "failed recheck must drop tracker bytes to avoid leak")
}

// DeliverTx success must clear the tracker once the sequence advances past
// FromSequence — the common path when a replacement is reaped into a block
// without a prior recheck.
func (s *ReplacementDecoratorSuite) TestDeliverTx_ClearsTrackerWhenSequenceAdvances() {
	s.SetupTest(true)
	priv, addr, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, a := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, a)
		s.Require().NoError(acc.SetSequence(432))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, a, acc.GetAccountNumber()
	}()

	sender := addr.String()
	tracker := ante.NewReplacementTracker()
	tracker.Set(sender, 432, []byte("replacement-bytes"))
	dec := s.makeDecorator(tracker)

	tx := s.makeTxAtSeq(priv, accNum, 432)
	bumpSeq := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		acc := s.app.AccountKeeper.GetAccount(ctx, addr)
		s.Require().NoError(acc.SetSequence(433))
		s.app.AccountKeeper.SetAccount(ctx, acc)
		return ctx, nil
	}

	// DeliverTx path: neither CheckTx nor ReCheckTx.
	_, err := dec.AnteHandle(s.ctx.WithIsCheckTx(false).WithIsReCheckTx(false), tx, false, bumpSeq)
	s.Require().NoError(err)
	s.Require().Nil(tracker.Get(sender), "DeliverTx must clear tracker after sequence advances")
}

// Txs at seq > fromSequence must NOT be evicted — evicting the tx at
// fromSequence breaks sequence continuity for the queue, so running these txs
// through the real recheck chain would fail them in SigVerificationDecorator
// (and a failed recheck also removes the tx from the app-side mempool, see
// baseapp.runTx).  The decorator must therefore short-circuit them to success
// while the tracker entry is live.  The failing `next` handler below simulates
// SigVerificationDecorator's wrong-sequence error: if the decorator ever calls
// it, the queue would be flushed.
func (s *ReplacementDecoratorSuite) TestRecheck_SubsequentTxsKeptAlive() {
	s.SetupTest(true)
	priv, _, accNum := func() (cryptotypes.PrivKey, sdk.AccAddress, uint64) {
		k, _, addr := testdata.KeyTestPubAddr()
		acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
		// Committed sequence is still 432: the stuck tx was evicted, the
		// replacement has not been included yet.
		s.Require().NoError(acc.SetSequence(432))
		s.app.AccountKeeper.SetAccount(s.ctx, acc)
		return k, addr, acc.GetAccountNumber()
	}()

	tracker := ante.NewReplacementTracker()
	// Replacement recorded at seq 432; the txs under test are at 433+.
	tracker.Set(sdk.AccAddress(priv.PubKey().Address()).String(), 432, []byte("replacement-bytes"))
	dec := s.makeDecorator(tracker)

	sigVerifyHandler := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, sdkerrors.ErrWrongSequence.Wrap("account sequence mismatch")
	}
	for _, seq := range []uint64{433, 434, 441} {
		tx := s.makeTxAtSeq(priv, accNum, seq)
		_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), tx, false, sigVerifyHandler)
		s.Require().NoError(err, "queued tx at seq=%d must be kept alive while the replacement is pending", seq)
	}

	// Once the tracker entry is gone (replacement included, entry cleared),
	// queued txs must go through the normal recheck chain again.
	tracker.Clear(sdk.AccAddress(priv.PubKey().Address()).String())
	tx := s.makeTxAtSeq(priv, accNum, 433)
	_, err := dec.AnteHandle(s.ctx.WithIsReCheckTx(true), tx, false, sigVerifyHandler)
	s.Require().ErrorIs(err, sdkerrors.ErrWrongSequence, "without a live entry the decorator must be transparent")
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

// A would-be replacement that fails the downstream ante chain (e.g. bad
// signature) must NOT be registered in the tracker.  Registration happens only
// after next() succeeds; otherwise an attacker could evict a victim's pending
// tx with a forged, unverifiable tx.
func (s *ReplacementDecoratorSuite) TestNewTx_FailedReplacementDoesNotRegister() {
	s.SetupTest(true)
	priv, _, addr := testdata.KeyTestPubAddr()

	acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
	s.Require().NoError(acc.SetSequence(432))
	committedCtx := s.ctx.WithMultiStore(s.app.CommitMultiStore())
	s.app.AccountKeeper.SetAccount(committedCtx, acc)
	s.Require().NoError(acc.SetSequence(443))
	s.app.AccountKeeper.SetAccount(s.ctx, acc)

	tracker := ante.NewReplacementTracker()
	dec := s.makeDecorator(tracker)

	tx := s.makeTxAtSeq(priv, acc.GetAccountNumber(), 432)
	failHandler := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, sdkerrors.ErrUnauthorized.Wrap("signature verification failed")
	}
	_, err := dec.AnteHandle(s.ctx.WithIsCheckTx(true).WithTxBytes([]byte("forged")), tx, false, failHandler)
	s.Require().ErrorIs(err, sdkerrors.ErrUnauthorized)
	s.Require().Nil(tracker.Get(addr.String()), "failed replacement must not poison the tracker")
}

// A panic in a downstream decorator (e.g. out-of-gas in SigGasConsumeDecorator,
// recovered by SetUpContextDecorator far above this frame) must not leave a
// poisoned tracker entry either.  Before the fix, the entry was registered
// before next() and only cleaned up on the error return path, so a panic
// skipped the cleanup.
func (s *ReplacementDecoratorSuite) TestNewTx_PanickingReplacementDoesNotRegister() {
	s.SetupTest(true)
	priv, _, addr := testdata.KeyTestPubAddr()

	acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, addr)
	s.Require().NoError(acc.SetSequence(432))
	committedCtx := s.ctx.WithMultiStore(s.app.CommitMultiStore())
	s.app.AccountKeeper.SetAccount(committedCtx, acc)
	s.Require().NoError(acc.SetSequence(443))
	s.app.AccountKeeper.SetAccount(s.ctx, acc)

	tracker := ante.NewReplacementTracker()
	dec := s.makeDecorator(tracker)

	tx := s.makeTxAtSeq(priv, acc.GetAccountNumber(), 432)
	panicHandler := func(_ sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		panic(storetypes.ErrorOutOfGas{Descriptor: "ante gas"})
	}
	s.Require().Panics(func() {
		_, _ = dec.AnteHandle(s.ctx.WithIsCheckTx(true).WithTxBytes([]byte("forged")), tx, false, panicHandler)
	})
	s.Require().Nil(tracker.Get(addr.String()), "a downstream panic must not poison the tracker")
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
	queuedTxs := make(map[uint64]sdk.Tx)
	for seq := uint64(432); seq <= 442; seq++ {
		tx := s.makeBankSendTxAtSeqWithMemo(priv, acc.GetAccountNumber(), seq, "queued-tx")
		if seq == 432 {
			originalSeq432Tx = tx
		} else {
			queuedTxs[seq] = tx
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

	// CometBFT rechecks in FIFO order: the queued txs at 433..442 come BEFORE
	// the replacement (it was admitted last).  With the stuck tx at 432 just
	// evicted, sequence continuity is broken, so without the keep-alive in
	// handleRecheck every queued tx would fail SigVerification here — and a
	// failed recheck removes the tx from the app mempool too (baseapp.runTx),
	// flushing the sender's whole queue from every node.
	for seq := uint64(433); seq <= 442; seq++ {
		s.checkTxWithType(queuedTxs[seq], abci.CheckTxType_Recheck, 0)
	}
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)

	s.checkTxWithType(replacementTx, abci.CheckTxType_Recheck, 0)
	s.Require().Equal(11, s.app.Mempool().CountTx())
	s.Require().Equal(
		[]uint64{432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442},
		s.selectedMempoolSequences(s.app.Mempool()),
	)
	s.Require().Equal("replacement-at-committed-sequence", s.selectedMempoolMemos(s.app.Mempool())[0])
}
