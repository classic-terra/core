package mempool

import (
	"context"
	"fmt"
	"math"

	"github.com/classic-terra/core/v3/custom/auth/ante"

	"github.com/huandu/skiplist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var (
	_ mempool.Mempool  = (*PriorityOracleMempool)(nil)
	_ mempool.Iterator = (*PriorityNonceIterator)(nil)
)

// PriorityOracleMempool is a mempool implementation that stores txs
// in a partially ordered set by 3 dimensions: oracle tx type, priority, and sender-nonce
// (sequence number). Oracle transactions always have higher priority than non-Oracle transactions.
//
// Internally it uses one priority ordered skip list and one skip list per sender ordered
// by sender-nonce (sequence number). The priority ordering first considers whether a transaction
// is an Oracle transaction (Oracle txs come first), then considers the priority value, and finally
// the sender-nonce.
//
// When there are multiple txs from the same sender, they are ordered as follows:
// 1. Oracle transactions come before non-Oracle transactions
// 2. Within each type (Oracle/non-Oracle), they are ordered by priority
// 3. For equal priorities, they are ordered by sender-nonce
//
// This ensures that:
// - Oracle transactions are always processed before non-Oracle transactions
// - Within each type, higher priority transactions are processed first
// - For transactions with equal priority, they maintain proper nonce ordering
type PriorityOracleMempool struct {
	priorityIndex  *skiplist.SkipList
	priorityCounts map[int64]int
	senderIndices  map[string]*skiplist.SkipList
	scores         map[txMeta]txMeta
	onRead         func(tx sdk.Tx)
	txReplacement  func(op, np int64, oTx, nTx sdk.Tx) bool
	maxTx          int
}

type PriorityNonceIterator struct {
	senderCursors map[string]*skiplist.Element
	nextPriority  int64
	sender        string
	priorityNode  *skiplist.Element
	mempool       *PriorityOracleMempool
}

// txMeta stores transaction metadata used in indices
type txMeta struct {
	// nonce is the sender's sequence number
	nonce uint64
	// priority is the transaction's priority
	priority int64
	// sender is the transaction's sender
	sender string
	// weight is the transaction's weight, used as a tiebreaker for transactions with the same priority
	weight int64
	// senderElement is a pointer to the transaction's element in the sender index
	senderElement *skiplist.Element
	// isOracleTx is a flag to check if the transaction is an oracle tx
	isOracleTx bool
}

// txMetaLess is a comparator for txKeys that first compares if tx is Oracle tx,
// then priority, then weight, then sender, then nonce, uniquely identifying a transaction.
//
// The comparison order is:
// 1. Oracle tx status (Oracle txs come before non-Oracle txs)
// 2. Priority value (higher priority comes first)
// 3. Weight (used as tiebreaker for same priority)
// 4. Sender address (for deterministic ordering)
// 5. Nonce (sequence number for ordering txs from same sender)
//
// Note, txMetaLess is used as the comparator in the priority index.
func txMetaLess(a, b any) int {
	keyA := a.(txMeta)
	keyB := b.(txMeta)

	//// First compare if tx is Oracle tx
	//if keyA.isOracleTx != keyB.isOracleTx {
	//	if keyA.isOracleTx {
	//		return -1 // A is Oracle tx, should come first
	//	}
	//	return 1 // B is Oracle tx, should come first
	//}

	res := skiplist.Int64.Compare(keyA.priority, keyB.priority)
	if res != 0 {
		return res
	}

	// Weight is used as a tiebreaker for transactions with the same priority.
	// Weight is calculated in a single pass in .Select(...) and so will be 0
	// on .Insert(...).
	res = skiplist.Int64.Compare(keyA.weight, keyB.weight)
	if res != 0 {
		return res
	}

	// Because weight will be 0 on .Insert(...), we must also compare sender and
	// nonce to resolve priority collisions. If we didn't then transactions with
	// the same priority would overwrite each other in the priority index.
	res = skiplist.String.Compare(keyA.sender, keyB.sender)
	if res != 0 {
		return res
	}

	return skiplist.Uint64.Compare(keyA.nonce, keyB.nonce)
}

type PriorityOracleMempoolOption func(*PriorityOracleMempool)

// PriorityOracleWithOnRead sets a callback to be called when a tx is read from
// the mempool.
func PriorityOracleWithOnRead(onRead func(tx sdk.Tx)) PriorityOracleMempoolOption {
	return func(mp *PriorityOracleMempool) {
		mp.onRead = onRead
	}
}

// PriorityOracleWithTxReplacement sets a callback to be called when duplicated
// transaction nonce detected during mempool insert. An application can define a
// transaction replacement rule based on tx priority or certain transaction fields.
func PriorityOracleWithTxReplacement(txReplacementRule func(op, np int64, oTx, nTx sdk.Tx) bool) PriorityOracleMempoolOption {
	return func(mp *PriorityOracleMempool) {
		mp.txReplacement = txReplacementRule
	}
}

// PriorityOracleWithMaxTx sets the maximum number of transactions allowed in the
// mempool with the semantics:
//
// <0: disabled, `Insert` is a no-op
// 0: unlimited
// >0: maximum number of transactions allowed
func PriorityOracleWithMaxTx(maxTx int) PriorityOracleMempoolOption {
	return func(mp *PriorityOracleMempool) {
		mp.maxTx = maxTx
	}
}

// DefaultPriorityMempool returns a priorityNonceMempool with no options.
func DefaultPriorityMempool() mempool.Mempool {
	return NewPriorityMempool()
}

// NewPriorityMempool returns the SDK's default mempool implementation which
// returns txs in a partial order by 2 dimensions; priority, and sender-nonce.
func NewPriorityMempool(opts ...PriorityOracleMempoolOption) *PriorityOracleMempool {
	mp := &PriorityOracleMempool{
		priorityIndex:  skiplist.New(skiplist.LessThanFunc(txMetaLess)),
		priorityCounts: make(map[int64]int),
		senderIndices:  make(map[string]*skiplist.SkipList),
		scores:         make(map[txMeta]txMeta),
	}

	for _, opt := range opts {
		opt(mp)
	}

	return mp
}

// NextSenderTx returns the next transaction for a given sender by nonce order,
// i.e. the next valid transaction for the sender. If no such transaction exists,
// nil will be returned.
func (mp *PriorityOracleMempool) NextSenderTx(sender string) sdk.Tx {
	senderIndex, ok := mp.senderIndices[sender]
	if !ok {
		return nil
	}

	cursor := senderIndex.Front()
	return cursor.Value.(sdk.Tx)
}

// Insert attempts to insert a Tx into the app-side mempool in O(log n) time,
// returning an error if unsuccessful. Sender and nonce are derived from the
// transaction's first signature.
//
// Transactions are unique by sender and nonce. Inserting a duplicate tx is an
// O(log n) no-op.
//
// Inserting a duplicate tx with a different priority overwrites the existing tx,
// changing the total order of the mempool.
func (mp *PriorityOracleMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if mp.maxTx > 0 && mp.CountTx() >= mp.maxTx {
		return mempool.ErrMempoolTxMaxCapacity
	} else if mp.maxTx < 0 {
		return nil
	}

	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sdkContext := sdk.UnwrapSDKContext(ctx)
	priority := sdkContext.Priority()
	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence

	// Boost priority for Oracle transactions to ensure they come first
	isOracleTx := ante.IsOracleTx(tx.GetMsgs())
	if isOracleTx {
		if len(tx.GetMsgs()) == 1 {
			priority += 1000000 + priority
		}
	}

	fmt.Printf("Insert the tx %d %v\n", priority, isOracleTx)

	key := txMeta{nonce: nonce, priority: priority, sender: sender, isOracleTx: isOracleTx}

	senderIndex, ok := mp.senderIndices[sender]
	if !ok {
		senderIndex = skiplist.New(skiplist.LessThanFunc(func(a, b any) int {
			return skiplist.Uint64.Compare(b.(txMeta).nonce, a.(txMeta).nonce)
		}))

		// initialize sender index if not found
		mp.senderIndices[sender] = senderIndex
	}

	// Since mp.priorityIndex is scored by priority, then sender, then nonce, a
	// changed priority will create a new key, so we must remove the old key and
	// re-insert it to avoid having the same tx with different priorityIndex indexed
	// twice in the mempool.
	//
	// This O(log n) remove operation is rare and only happens when a tx's priority
	// changes.
	sk := txMeta{nonce: nonce, sender: sender}
	if oldScore, txExists := mp.scores[sk]; txExists {
		if mp.txReplacement != nil && !mp.txReplacement(oldScore.priority, priority, senderIndex.Get(key).Value.(sdk.Tx), tx) {
			return fmt.Errorf(
				"tx doesn't fit the replacement rule, oldPriority: %v, newPriority: %v, oldTx: %v, newTx: %v",
				oldScore.priority,
				priority,
				senderIndex.Get(key).Value.(sdk.Tx),
				tx,
			)
		}

		mp.priorityIndex.Remove(txMeta{
			nonce:    nonce,
			sender:   sender,
			priority: oldScore.priority,
			weight:   oldScore.weight,
		})
		mp.priorityCounts[oldScore.priority]--
	}

	mp.priorityCounts[priority]++

	// Since senderIndex is scored by nonce, a changed priority will overwrite the
	// existing key.
	key.senderElement = senderIndex.Set(key, tx)

	mp.scores[sk] = txMeta{priority: priority}
	mp.priorityIndex.Set(key, tx)

	return nil
}

func (i *PriorityNonceIterator) iteratePriority() mempool.Iterator {
	// beginning of priority iteration
	if i.priorityNode == nil {
		i.priorityNode = i.mempool.priorityIndex.Front()
	} else {
		i.priorityNode = i.priorityNode.Next()
	}

	// end of priority iteration
	if i.priorityNode == nil {
		return nil
	}

	i.sender = i.priorityNode.Key().(txMeta).sender

	nextPriorityNode := i.priorityNode.Next()
	if nextPriorityNode != nil {
		i.nextPriority = nextPriorityNode.Key().(txMeta).priority
	} else {
		i.nextPriority = math.MinInt64
	}

	return i.Next()
}

func (i *PriorityNonceIterator) Next() mempool.Iterator {
	if i.priorityNode == nil {
		return nil
	}

	cursor, ok := i.senderCursors[i.sender]
	if !ok {
		// beginning of sender iteration
		cursor = i.mempool.senderIndices[i.sender].Front()
	} else {
		// middle of sender iteration
		cursor = cursor.Next()
	}

	// end of sender iteration
	if cursor == nil {
		return i.iteratePriority()
	}

	key := cursor.Key().(txMeta)

	// We've reached a transaction with a priority lower than the next highest
	// priority in the pool.
	if key.priority < i.nextPriority {
		return i.iteratePriority()
	} else if key.priority == i.nextPriority && i.priorityNode.Next() != nil {
		// Weight is incorporated into the priority index key only (not sender index)
		// so we must fetch it here from the scores map.
		weight := i.mempool.scores[txMeta{nonce: key.nonce, sender: key.sender}].weight
		if weight < i.priorityNode.Next().Key().(txMeta).weight {
			return i.iteratePriority()
		}
	}

	i.senderCursors[i.sender] = cursor
	return i
}

func (i *PriorityNonceIterator) Tx() sdk.Tx {
	return i.senderCursors[i.sender].Value.(sdk.Tx)
}

// Select returns a set of transactions from the mempool, ordered by priority
// and sender-nonce in O(n) time. The passed in list of transactions are ignored.
// This is a readonly operation, the mempool is not modified.
//
// The maxBytes parameter defines the maximum number of bytes of transactions to
// return.
//
// NOTE: It is not safe to use this iterator while removing transactions from
// the underlying mempool.
func (mp *PriorityOracleMempool) Select(_ context.Context, _ [][]byte) mempool.Iterator {
	fmt.Println("Mempool priority Select")
	if mp.priorityIndex.Len() == 0 {
		return nil
	}

	mp.reorderPriorityTies()

	iterator := &PriorityNonceIterator{
		mempool:       mp,
		senderCursors: make(map[string]*skiplist.Element),
	}

	return iterator.iteratePriority()
}

type reorderKey struct {
	deleteKey txMeta
	insertKey txMeta
	tx        sdk.Tx
}

func (mp *PriorityOracleMempool) reorderPriorityTies() {
	node := mp.priorityIndex.Front()

	var reordering []reorderKey
	for node != nil {
		key := node.Key().(txMeta)
		if mp.priorityCounts[key.priority] > 1 {
			newKey := key
			newKey.weight = senderWeight(key.senderElement)
			reordering = append(reordering, reorderKey{deleteKey: key, insertKey: newKey, tx: node.Value.(sdk.Tx)})
		}

		node = node.Next()
	}

	for _, k := range reordering {
		mp.priorityIndex.Remove(k.deleteKey)
		delete(mp.scores, txMeta{nonce: k.deleteKey.nonce, sender: k.deleteKey.sender})
		mp.priorityIndex.Set(k.insertKey, k.tx)
		mp.scores[txMeta{nonce: k.insertKey.nonce, sender: k.insertKey.sender}] = k.insertKey
	}
}

// senderWeight returns the weight of a given tx (t) at senderCursor. Weight is
// defined as the first (nonce-wise) same sender tx with a priority not equal to
// t. It is used to resolve priority collisions, that is when 2 or more txs from
// different senders have the same priority.
func senderWeight(senderCursor *skiplist.Element) int64 {
	if senderCursor == nil {
		return 0
	}

	weight := senderCursor.Key().(txMeta).priority
	senderCursor = senderCursor.Next()
	for senderCursor != nil {
		p := senderCursor.Key().(txMeta).priority
		if p != weight {
			weight = p
		}

		senderCursor = senderCursor.Next()
	}

	return weight
}

// CountTx returns the number of transactions in the mempool.
func (mp *PriorityOracleMempool) CountTx() int {
	return mp.priorityIndex.Len()
}

// Remove removes a transaction from the mempool in O(log n) time, returning an
// error if unsuccessful.
func (mp *PriorityOracleMempool) Remove(tx sdk.Tx) error {
	fmt.Printf("Remove tx %v\n", tx.GetMsgs())
	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("attempted to remove a tx with no signatures")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence

	scoreKey := txMeta{nonce: nonce, sender: sender}
	score, ok := mp.scores[scoreKey]
	if !ok {
		return mempool.ErrTxNotFound
	}
	tk := txMeta{nonce: nonce, priority: score.priority, sender: sender, weight: score.weight}

	senderTxs, ok := mp.senderIndices[sender]
	if !ok {
		return fmt.Errorf("sender %s not found", sender)
	}

	mp.priorityIndex.Remove(tk)
	senderTxs.Remove(tk)
	delete(mp.scores, scoreKey)
	mp.priorityCounts[score.priority]--

	return nil
}

func IsEmpty(mempool mempool.Mempool) error {
	mp := mempool.(*PriorityOracleMempool)
	if mp.priorityIndex.Len() != 0 {
		return fmt.Errorf("priorityIndex not empty")
	}

	var countKeys []int64
	for k := range mp.priorityCounts {
		countKeys = append(countKeys, k)
	}

	for _, k := range countKeys {
		if mp.priorityCounts[k] != 0 {
			return fmt.Errorf("priorityCounts not zero at %v, got %v", k, mp.priorityCounts[k])
		}
	}

	var senderKeys []string
	for k := range mp.senderIndices {
		senderKeys = append(senderKeys, k)
	}

	for _, k := range senderKeys {
		if mp.senderIndices[k].Len() != 0 {
			return fmt.Errorf("senderIndex not empty for sender %v", k)
		}
	}

	return nil
}
