package ante

import (
	"sync"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

const (
	// DefaultMaxReplacementEntries caps distinct senders tracked at once.
	// Prevents unbounded growth if a cleanup path is missed.
	DefaultMaxReplacementEntries = 4096

	// DefaultMaxTxBytesPerSender caps how many distinct replacement txs are
	// retained for a single sender+sequence. Rapid replacements stay tracked,
	// but a single account cannot grow the set without bound.
	DefaultMaxTxBytesPerSender = 16

	// DefaultReplacementTTLBlocks drops tracker entries that survive this many
	// blocks without being cleared by recheck/deliver. Acts as a backstop for
	// the rare path where CometBFT drops a replacement without rechecking it.
	DefaultReplacementTTLBlocks = 100
)

// ReplacementTracker tracks pending tx replacements so old txs can be
// evicted during CometBFT recheck.  Thread-safe.
type ReplacementTracker struct {
	mu                  sync.RWMutex
	replacements        map[string]*ReplacementInfo
	order               []string // insertion order for max-entries eviction
	maxEntries          int
	maxTxBytesPerSender int
	ttlBlocks           int64
}

type ReplacementInfo struct {
	FromSequence    uint64
	SetHeight       int64
	newTxBytesSet   map[string]struct{} // keyed by string(txBytes) for O(1) membership
	newTxBytesOrder []string            // insertion order for per-sender cap
}

// Contains reports whether txBytes is a registered replacement for this entry.
func (info *ReplacementInfo) Contains(txBytes []byte) bool {
	_, ok := info.newTxBytesSet[string(txBytes)]
	return ok
}

func NewReplacementTracker() *ReplacementTracker {
	return &ReplacementTracker{
		replacements:        make(map[string]*ReplacementInfo),
		maxEntries:          DefaultMaxReplacementEntries,
		maxTxBytesPerSender: DefaultMaxTxBytesPerSender,
		ttlBlocks:           DefaultReplacementTTLBlocks,
	}
}

func (rt *ReplacementTracker) Set(sender string, fromSeq uint64, newTxBytes []byte) {
	rt.SetAtHeight(sender, fromSeq, newTxBytes, 0)
}

func (rt *ReplacementTracker) SetAtHeight(sender string, fromSeq uint64, newTxBytes []byte, height int64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	key := string(newTxBytes)
	info, exists := rt.replacements[sender]
	if !exists || info.FromSequence != fromSeq {
		// New sender or different sequence: start a fresh set.
		rt.replacements[sender] = &ReplacementInfo{
			FromSequence:    fromSeq,
			SetHeight:       height,
			newTxBytesSet:   map[string]struct{}{key: {}},
			newTxBytesOrder: []string{key},
		}
		if !exists {
			rt.order = append(rt.order, sender)
		} else {
			rt.moveToEndLocked(sender)
		}
		rt.enforceMaxEntriesLocked(sender)
		return
	}
	if _, already := info.newTxBytesSet[key]; already {
		rt.moveToEndLocked(sender)
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
	newSet[key] = struct{}{}
	newOrder := make([]string, len(info.newTxBytesOrder), len(info.newTxBytesOrder)+1)
	copy(newOrder, info.newTxBytesOrder)
	newOrder = append(newOrder, key)

	// Drop oldest replacements once the per-sender cap is exceeded.
	maxPerSender := rt.maxTxBytesPerSender
	if maxPerSender <= 0 {
		maxPerSender = DefaultMaxTxBytesPerSender
	}
	for len(newOrder) > maxPerSender {
		victim := newOrder[0]
		newOrder = newOrder[1:]
		delete(newSet, victim)
	}

	rt.replacements[sender] = &ReplacementInfo{
		FromSequence:    fromSeq,
		SetHeight:       info.SetHeight,
		newTxBytesSet:   newSet,
		newTxBytesOrder: newOrder,
	}
	rt.moveToEndLocked(sender)
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
		rt.removeFromOrderLocked(sender)
		return
	}
	newSet := make(map[string]struct{}, len(info.newTxBytesSet)-1)
	for k := range info.newTxBytesSet {
		if k != key {
			newSet[k] = struct{}{}
		}
	}
	newOrder := make([]string, 0, len(info.newTxBytesOrder)-1)
	for _, k := range info.newTxBytesOrder {
		if k != key {
			newOrder = append(newOrder, k)
		}
	}
	rt.replacements[sender] = &ReplacementInfo{
		FromSequence:    info.FromSequence,
		SetHeight:       info.SetHeight,
		newTxBytesSet:   newSet,
		newTxBytesOrder: newOrder,
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
	if _, ok := rt.replacements[sender]; !ok {
		return
	}
	delete(rt.replacements, sender)
	rt.removeFromOrderLocked(sender)
}

// ClearIfPast deletes the sender's entry when the committed sequence has
// advanced past FromSequence (replacement was included / sequence consumed).
func (rt *ReplacementTracker) ClearIfPast(sender string, committedSeq uint64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	info, ok := rt.replacements[sender]
	if !ok {
		return
	}
	if committedSeq > info.FromSequence {
		delete(rt.replacements, sender)
		rt.removeFromOrderLocked(sender)
	}
}

// Prune removes entries whose committed sequence has advanced past FromSequence
// and entries older than ttlBlocks. getSeq returns (sequence, true) for known
// accounts; returning false drops the entry as orphaned.
func (rt *ReplacementTracker) Prune(getSeq func(sender string) (uint64, bool), height int64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	kept := rt.order[:0]
	for _, sender := range rt.order {
		info, ok := rt.replacements[sender]
		if !ok {
			continue
		}
		if rt.ttlBlocks > 0 && info.SetHeight > 0 && height-info.SetHeight > rt.ttlBlocks {
			delete(rt.replacements, sender)
			continue
		}
		seq, found := getSeq(sender)
		if !found || seq > info.FromSequence {
			delete(rt.replacements, sender)
			continue
		}
		kept = append(kept, sender)
	}
	rt.order = kept
}

func (rt *ReplacementTracker) Len() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return len(rt.replacements)
}

func (rt *ReplacementTracker) moveToEndLocked(sender string) {
	rt.removeFromOrderLocked(sender)
	rt.order = append(rt.order, sender)
}

func (rt *ReplacementTracker) removeFromOrderLocked(sender string) {
	for i, s := range rt.order {
		if s == sender {
			rt.order = append(rt.order[:i], rt.order[i+1:]...)
			return
		}
	}
}

func (rt *ReplacementTracker) enforceMaxEntriesLocked(keepSender string) {
	if rt.maxEntries <= 0 {
		return
	}
	for len(rt.replacements) > rt.maxEntries && len(rt.order) > 0 {
		victim := rt.order[0]
		if victim == keepSender {
			if len(rt.order) == 1 {
				return
			}
			// Rotate keepSender to the end and evict the next oldest.
			rt.order = append(rt.order[1:], victim)
			victim = rt.order[0]
		}
		delete(rt.replacements, victim)
		rt.order = rt.order[1:]
	}
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
//   - The superseded tx at exactly the replaced sequence is rejected, causing
//     CometBFT to evict it from its mempool.
//   - Queued txs at seq > FromSequence are deliberately short-circuited to
//     success while the tracker entry is live.  Evicting the tx at
//     FromSequence breaks sequence continuity for the rest of the queue, so
//     letting them run through SigVerificationDecorator would evict them all
//     (a failed recheck also removes the tx from the app-side mempool, see
//     baseapp.runTx).  They were fully validated at admission and are
//     re-validated at PrepareProposal/ProcessProposal/DeliverTx.
//
// On DeliverTx:
//   - Once the account sequence advances past FromSequence the tracker entry
//     is cleared (covers the common path where a replacement is reaped
//     straight into a block without a recheck).  This — together with the
//     BeginBlock prune and the TTL backstop — is what ends the entry's
//     lifetime; a successful recheck alone does not, because the entry must
//     survive every recheck round until the replacement is actually included.
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

	// DeliverTx: after IncrementSequenceDecorator runs inside next(), clear
	// the tracker once the sequence has advanced past the replaced one.
	newCtx, err := next(ctx, tx, false)
	if err == nil {
		d.clearAfterDeliver(newCtx, tx)
	}
	return newCtx, err
}

func (d TxReplacementDecorator) clearAfterDeliver(ctx sdk.Context, tx sdk.Tx) {
	sender, _, err := firstSignerSeq(tx)
	if err != nil {
		return
	}
	acc := d.ak.GetAccount(ctx, sdk.MustAccAddressFromBech32(sender))
	if acc == nil {
		return
	}
	d.tracker.ClearIfPast(sender, acc.GetSequence())
}

// handleRecheck rejects the old tx at fromSequence that has been superseded by
// a replacement, and keeps the rest of the sender's queue alive while the
// replacement is still pending (see the type-level comment for the rationale).
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
		// Drop this replacement's bytes only on failure — CometBFT will evict
		// the tx, so keeping the bytes would only leak until a later prune.
		// On success the entry MUST survive: rechecks happen once per block
		// until the replacement is included, and the queued txs above
		// FromSequence rely on the live entry to stay in the pool.  The entry
		// is cleared once the sequence advances (DeliverTx / BeginBlock prune).
		newCtx, err := next(ctx, tx, false)
		if err != nil {
			d.tracker.RemoveTxBytes(sender, ctx.TxBytes())
		}
		return newCtx, err
	}

	// Evict only the original stuck tx at the replaced sequence.
	if seq == info.FromSequence {
		return ctx, sdkerrors.ErrWrongSequence.Wrapf(
			"tx superseded: a replacement tx was submitted for %s at sequence %d",
			sender, info.FromSequence,
		)
	}

	// Queued txs above the replaced sequence: skip the rest of the recheck
	// chain.  With the tx at FromSequence evicted, SigVerificationDecorator
	// would see a sequence gap and fail them, and a failed recheck removes the
	// tx from both CometBFT's and the app's mempool — flushing the sender's
	// whole queue.  Keep them alive until the replacement fills the gap; if it
	// never does, the entry expires (TTL/prune) and normal rechecks resume.
	if seq > info.FromSequence {
		return ctx, nil
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

	newCtx, err := next(ctx, tx, false)
	if err != nil {
		return newCtx, err
	}

	// Register the replacement only AFTER the downstream ante chain (including
	// SigVerificationDecorator) has accepted the tx.  Registering before would
	// let an attacker poison the tracker with an unsigned tx naming a victim
	// as first signer: a panic in a later decorator (e.g. out of gas during
	// SigGasConsumeDecorator) unwinds past this frame to SetUpContextDecorator's
	// recover, skipping any error-path cleanup — and the poisoned entry would
	// then evict the victim's legitimate pending tx on every recheck.
	d.tracker.SetAtHeight(sender, committedSeq, ctx.TxBytes(), ctx.BlockHeight())
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
