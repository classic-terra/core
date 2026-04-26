package ante_test

import (
	"testing"

	"github.com/classic-terra/core/v4/custom/auth/ante"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	require.Equal(t, orig, info.NewTxBytes)

	// Set must store a copy; mutating original must not affect stored bytes.
	orig[0] = 'X'
	require.Equal(t, byte('n'), tracker.Get(sender).NewTxBytes[0])

	tracker.Clear(sender)
	require.Nil(t, tracker.Get(sender))

	// Clear on unknown sender is a no-op.
	tracker.Clear("unknown")
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
	s.Require().Equal(txBytes, info.NewTxBytes)

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
