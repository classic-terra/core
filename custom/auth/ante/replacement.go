package ante

import (
	"sync"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// ReplacementTracker tracks pending tx replacements so old txs can be
// evicted during CometBFT recheck.  Thread-safe.
type ReplacementTracker struct {
	mu           sync.RWMutex
	replacements map[string]*ReplacementInfo
}

type ReplacementInfo struct {
	FromSequence  uint64
	newTxBytesSet map[string]struct{} // keyed by string(txBytes) for O(1) membership
}

// Contains reports whether txBytes is a registered replacement for this entry.
func (info *ReplacementInfo) Contains(txBytes []byte) bool {
	_, ok := info.newTxBytesSet[string(txBytes)]
	return ok
}

func NewReplacementTracker() *ReplacementTracker {
	return &ReplacementTracker{
		replacements: make(map[string]*ReplacementInfo),
	}
}

func (rt *ReplacementTracker) Set(sender string, fromSeq uint64, newTxBytes []byte) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	info, exists := rt.replacements[sender]
	if !exists || info.FromSequence != fromSeq {
		// New sender or different sequence: start a fresh set.
		rt.replacements[sender] = &ReplacementInfo{
			FromSequence:  fromSeq,
			newTxBytesSet: map[string]struct{}{string(newTxBytes): {}},
		}
		return
	}
	// Same sequence: register alongside any prior ones so all rapid replacements
	// are tracked and none is incorrectly evicted during recheck.
	//
	// Copy-on-write: callers that received a *ReplacementInfo via Get hold a
	// reference to the old map and may call Contains concurrently.  Mutating
	// the existing map in-place while they read it would be a data race.
	// Publishing a brand-new ReplacementInfo with a fresh map is safe because
	// the old pointer remains valid for already-in-progress readers.
	newSet := make(map[string]struct{}, len(info.newTxBytesSet)+1)
	for k := range info.newTxBytesSet {
		newSet[k] = struct{}{}
	}
	newSet[string(newTxBytes)] = struct{}{}
	rt.replacements[sender] = &ReplacementInfo{
		FromSequence:  fromSeq,
		newTxBytesSet: newSet,
	}
}

// RemoveTxBytes removes a single replacement byte string from the tracked set.
// When the last entry is removed the sender key is deleted entirely, which
// preserves the invariant that tracker.Get returns nil once all replacements
// have been rechecked successfully.  Copy-on-write is used for the same
// concurrency reason as in Set.
func (rt *ReplacementTracker) RemoveTxBytes(sender string, txBytes []byte) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	info, ok := rt.replacements[sender]
	if !ok {
		return
	}
	key := string(txBytes)
	if _, found := info.newTxBytesSet[key]; !found {
		return
	}
	if len(info.newTxBytesSet) <= 1 {
		delete(rt.replacements, sender)
		return
	}
	newSet := make(map[string]struct{}, len(info.newTxBytesSet)-1)
	for k := range info.newTxBytesSet {
		if k != key {
			newSet[k] = struct{}{}
		}
	}
	rt.replacements[sender] = &ReplacementInfo{
		FromSequence:  info.FromSequence,
		newTxBytesSet: newSet,
	}
}

func (rt *ReplacementTracker) Get(sender string) *ReplacementInfo {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.replacements[sender]
}

func (rt *ReplacementTracker) Clear(sender string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.replacements, sender)
}

// TxReplacementDecorator allows a sender to replace a stuck pending tx by
// submitting a new tx at the same (committed) sequence number.
//
// On new CheckTx:
//   - If the tx sequence < pending sequence but == committed sequence,
//     the account is reset to committed state so the replacement can pass
//     signature verification.
//
// On recheck:
//   - Old txs from the same sender (at or above the replaced sequence) are
//     rejected, causing CometBFT to evict them from its mempool.
type TxReplacementDecorator struct {
	ak      ante.AccountKeeper
	cms     storetypes.CommitMultiStore
	tracker *ReplacementTracker
}

func NewTxReplacementDecorator(
	ak ante.AccountKeeper,
	cms storetypes.CommitMultiStore,
	tracker *ReplacementTracker,
) TxReplacementDecorator {
	return TxReplacementDecorator{ak: ak, cms: cms, tracker: tracker}
}

func (d TxReplacementDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	if ctx.IsReCheckTx() {
		return d.handleRecheck(ctx, tx, next)
	}

	if ctx.IsCheckTx() {
		return d.handleNewTx(ctx, tx, next)
	}

	return next(ctx, tx, false)
}

// handleRecheck rejects the old tx at fromSequence that has been superseded by
// a replacement.  Subsequent txs (seq > fromSequence) are left in the pool so
// CometBFT's normal sequence validation handles them after each committed block.
func (d TxReplacementDecorator) handleRecheck(ctx sdk.Context, tx sdk.Tx, next sdk.AnteHandler) (sdk.Context, error) {
	sender, seq, err := firstSignerSeq(tx)
	if err != nil {
		return next(ctx, tx, false)
	}

	info := d.tracker.Get(sender)
	if info == nil {
		return next(ctx, tx, false)
	}

	// This is one of the registered replacement txs — reset the account and allow it.
	if info.Contains(ctx.TxBytes()) {
		committedAcc := d.getCommittedAccount(ctx, sdk.MustAccAddressFromBech32(sender))
		// Reset account to committed state when available so SigVerificationDecorator
		// sees the correct sequence.  If committedAcc is nil (very rare: pruned account),
		// skip the reset and let next() try with whatever state it has.
		if committedAcc != nil && seq == committedAcc.GetSequence() {
			d.ak.SetAccount(ctx, committedAcc)
		}
		// Remove only this replacement's bytes after next() succeeds.  Using
		// RemoveTxBytes instead of Clear ensures that if a second replacement for
		// the same sender/sequence was registered, the sender entry (and its
		// fromSequence eviction rule) survives until all replacements have been
		// rechecked.  If next() fails keep the tracker alive so the replacement
		// can be retried rather than silently losing both txs.
		newCtx, err := next(ctx, tx, false)
		if err == nil {
			d.tracker.RemoveTxBytes(sender, ctx.TxBytes())
		}
		return newCtx, err
	}

	// Evict only the original stuck tx at the replaced sequence.
	// Txs at seq > fromSequence are valid and will be handled by normal
	// sequence validation as the committed sequence advances.
	if seq == info.FromSequence {
		return ctx, sdkerrors.ErrWrongSequence.Wrapf(
			"tx superseded: a replacement tx was submitted for %s at sequence %d",
			sender, info.FromSequence,
		)
	}

	return next(ctx, tx, false)
}

// handleNewTx detects when a tx's sequence equals the committed (on-chain)
// sequence but is below the pending (mempool-accumulated) sequence. In that
// case, the account state in the ante context is reset to the committed
// version so that the signature/sequence check can pass.
func (d TxReplacementDecorator) handleNewTx(ctx sdk.Context, tx sdk.Tx, next sdk.AnteHandler) (sdk.Context, error) {
	sender, txSeq, err := firstSignerSeq(tx)
	if err != nil {
		return next(ctx, tx, false)
	}

	acc := d.ak.GetAccount(ctx, sdk.MustAccAddressFromBech32(sender))
	if acc == nil {
		return next(ctx, tx, false)
	}

	pendingSeq := acc.GetSequence()
	if txSeq >= pendingSeq {
		return next(ctx, tx, false)
	}

	// txSeq < pendingSeq → possible replacement.
	// Verify it matches the committed (on-chain) sequence.
	committedAcc := d.getCommittedAccount(ctx, sdk.MustAccAddressFromBech32(sender))
	if committedAcc == nil {
		return next(ctx, tx, false)
	}

	committedSeq := committedAcc.GetSequence()
	if txSeq != committedSeq {
		// Only allow replacement starting from the committed sequence.
		return next(ctx, tx, false)
	}

	// Reset the account in the (branched) ante context to committed state.
	// If the full ante chain succeeds the branch is written; otherwise discarded.
	d.ak.SetAccount(ctx, committedAcc)

	d.tracker.Set(sender, committedSeq, ctx.TxBytes())

	newCtx, err := next(ctx, tx, false)
	if err != nil {
		// The replacement tx was rejected downstream; clear the tracker so the
		// original stuck tx is not evicted during recheck without a valid replacement.
		d.tracker.Clear(sender)
		return newCtx, err
	}
	return newCtx, nil
}

func (d TxReplacementDecorator) getCommittedAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI {
	committedCtx := ctx.WithMultiStore(d.cms)
	return d.ak.GetAccount(committedCtx, addr)
}

// firstSignerSeq returns the bech32 address and sequence of the first signer.
func firstSignerSeq(tx sdk.Tx) (string, uint64, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return "", 0, sdkerrors.ErrTxDecode.Wrap("tx is not SigVerifiableTx")
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return "", 0, err
	}
	if len(sigs) == 0 {
		return "", 0, sdkerrors.ErrNoSignatures.Wrap("tx has no signatures")
	}
	addr := sdk.AccAddress(sigs[0].PubKey.Address()).String()
	return addr, sigs[0].Sequence, nil
}
