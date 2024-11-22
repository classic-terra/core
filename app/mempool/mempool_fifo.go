package mempool

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/classic-terra/core/v3/custom/auth/ante"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/huandu/skiplist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var cdc *codec.LegacyAmino

var (
	_ mempool.Mempool  = (*FifoSenderNonceMempool)(nil)
	_ mempool.Iterator = (*senderNonceMempoolIterator)(nil)
)

var DefaultMaxTx = 0

// Initialize the codec somewhere in your application setup
func init() {
	cdc = codec.NewLegacyAmino()
}

// FifoSenderNonceMempool is a mempool that prioritizes transactions within a sender
// by nonce, the lowest first, but selects a random sender on each iteration.
// The mempool is iterated by:
//
// 1) Maintaining a separate list of nonce ordered txs per sender
// 2) For each select iteration, randomly choose a sender and pick the next nonce ordered tx from their list
// 3) Repeat 1,2 until the mempool is exhausted
//
// Note that PrepareProposal could choose to stop iteration before reaching the
// end if maxBytes is reached.
type FifoSenderNonceMempool struct {
	senders       map[string]*skiplist.SkipList
	rnd           *rand.Rand
	maxTx         int
	existingTx    map[txKey]bool
	sendersOracle map[string]*skiplist.SkipList
}

type SenderNonceOptions func(mp *FifoSenderNonceMempool)

type txKey struct {
	address string
	nonce   uint64
}

// NewFifoSenderNonceMempool creates a new mempool that prioritizes transactions by
// nonce, the lowest first, picking a random sender on each iteration.
func NewFifoSenderNonceMempool(opts ...SenderNonceOptions) *FifoSenderNonceMempool {
	senderMap := make(map[string]*skiplist.SkipList)
	senderOracleMap := make(map[string]*skiplist.SkipList)
	existingTx := make(map[txKey]bool)
	snp := &FifoSenderNonceMempool{
		senders:       senderMap,
		maxTx:         DefaultMaxTx,
		existingTx:    existingTx,
		sendersOracle: senderOracleMap,
	}

	var seed int64
	err := binary.Read(crand.Reader, binary.BigEndian, &seed)
	if err != nil {
		panic(err)
	}

	snp.setSeed(seed)

	for _, opt := range opts {
		opt(snp)
	}

	return snp
}

// SenderNonceSeedOpt Option To add a Seed for random type when calling the
// constructor NewSenderNonceMempool.
//
// Example:
//
//	random_seed := int64(1000)
//	NewSenderNonceMempool(SenderNonceSeedTxOpt(random_seed))
func SenderNonceSeedOpt(seed int64) SenderNonceOptions {
	return func(snp *FifoSenderNonceMempool) {
		snp.setSeed(seed)
	}
}

// SenderNonceMaxTxOpt Option To set limit of max tx when calling the constructor
// NewSenderNonceMempool.
//
// Example:
//
//	NewSenderNonceMempool(SenderNonceMaxTxOpt(100))
func SenderNonceMaxTxOpt(maxTx int) SenderNonceOptions {
	return func(snp *FifoSenderNonceMempool) {
		snp.maxTx = maxTx
	}
}

func (snm *FifoSenderNonceMempool) setSeed(seed int64) {
	s1 := rand.NewSource(seed)
	snm.rnd = rand.New(s1) //#nosec // math/rand is seeded from crypto/rand by default
}

// NextSenderTx returns the next transaction for a given sender by nonce order,
// i.e. the next valid transaction for the sender. If no such transaction exists,
// nil will be returned.
func (mp *FifoSenderNonceMempool) NextSenderTx(sender string) sdk.Tx {
	senderIndex, ok := mp.senders[sender]
	if !ok {
		return nil
	}

	cursor := senderIndex.Front()
	return cursor.Value.(sdk.Tx)
}

// Insert adds a tx to the mempool. It returns an error if the tx does not have
// at least one signer. Note, priority is ignored.
func (snm *FifoSenderNonceMempool) Insert(_ context.Context, tx sdk.Tx) error {
	if snm.maxTx > 0 && snm.CountTx() >= snm.maxTx {
		return mempool.ErrMempoolTxMaxCapacity
	}
	if snm.maxTx < 0 {
		return nil
	}

	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence

	isOracleTx := ante.IsOracleTx(tx.GetMsgs())
	senderTxs, found := snm.senders[sender]
	if !found && !isOracleTx {
		senderTxs = skiplist.New(skiplist.Uint64)
		snm.senders[sender] = senderTxs
	}
	if !isOracleTx {
		senderTxs.Set(nonce, tx)
	}

	senderOracleTxs, found := snm.sendersOracle[sender]
	if !found && isOracleTx {
		senderOracleTxs = skiplist.New(skiplist.Uint64)
		snm.sendersOracle[sender] = senderOracleTxs
	}
	if isOracleTx {
		senderOracleTxs.Set(nonce, tx)
	}

	key := txKey{nonce: nonce, address: sender}
	snm.existingTx[key] = true

	return nil
}

func (snm *FifoSenderNonceMempool) handleOracleTransactions() mempool.Iterator {
	var senders []string
	senderCursors := make(map[string]*skiplist.Element)

	orderedSenders := skiplist.New(skiplist.String)

	// #nosec
	for s := range snm.sendersOracle {
		orderedSenders.Set(s, s)
	}

	s := orderedSenders.Front()
	for s != nil {
		sender := s.Value.(string)
		senders = append(senders, sender)
		senderCursors[sender] = snm.sendersOracle[sender].Front()
		s = s.Next()
	}

	iter := &senderNonceMempoolIterator{
		senders:       senders,
		rnd:           snm.rnd,
		senderCursors: senderCursors,
	}

	return iter.Next()
}

// Select returns an iterator ordering transactions the mempool with the lowest
// nonce of a random selected sender first.
//
// NOTE: It is not safe to use this iterator while removing transactions from
// the underlying mempool.
func (snm *FifoSenderNonceMempool) Select(_ context.Context, _ [][]byte) mempool.Iterator {
	var oracleSenders, regularSenders []string
	senderCursors := make(map[string]*skiplist.Element)

	// Handle oracle transactions first
	orderedSendersOracle := skiplist.New(skiplist.String)
	for s := range snm.sendersOracle {
		orderedSendersOracle.Set(s, s)
	}

	s1 := orderedSendersOracle.Front()
	for s1 != nil {
		sender := s1.Value.(string)
		oracleSenders = append(oracleSenders, sender)
		// Add oracle prefix to distinguish from regular transactions
		senderCursors[sender] = snm.sendersOracle[sender].Front()
		s1 = s1.Next()
	}

	// Handle regular transactions
	orderedSenders := skiplist.New(skiplist.String)
	for s := range snm.senders {
		orderedSenders.Set(s, s)
	}

	s := orderedSenders.Front()
	for s != nil {
		sender := s.Value.(string)
		regularSenders = append(regularSenders, sender)
		senderCursors[sender] = snm.senders[sender].Front()
		s = s.Next()
	}

	// Combine senders with oracle transactions first
	senders := append(oracleSenders, regularSenders...)

	iter := &senderNonceMempoolIterator{
		senders:       senders,
		rnd:           snm.rnd,
		senderCursors: senderCursors,
	}
	return iter.Next()
}

// CountTx returns the total count of txs in the mempool.
func (snm *FifoSenderNonceMempool) CountTx() int {
	return len(snm.existingTx)
}

// Remove removes a tx from the mempool. It returns an error if the tx does not
// have at least one signer or the tx was not found in the pool.
func (snm *FifoSenderNonceMempool) Remove(tx sdk.Tx) error {
	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence

	senderTxs, found1 := snm.senders[sender]
	senderOracleTxs, found2 := snm.sendersOracle[sender]
	if !found1 && !found2 {
		return mempool.ErrTxNotFound
	}

	if ante.IsOracleTx(tx.GetMsgs()) {
		res := senderOracleTxs.Remove(nonce)
		if res == nil {
			return mempool.ErrTxNotFound
		}

		if senderOracleTxs.Len() == 0 {
			delete(snm.senders, sender)
		}
	} else {
		res := senderTxs.Remove(nonce)
		if res == nil {
			return mempool.ErrTxNotFound
		}

		if senderTxs.Len() == 0 {
			delete(snm.senders, sender)
		}
	}

	key := txKey{nonce: nonce, address: sender}
	delete(snm.existingTx, key)

	return nil
}

type senderNonceMempoolIterator struct {
	rnd           *rand.Rand
	currentTx     *skiplist.Element
	senders       []string
	senderCursors map[string]*skiplist.Element
}

// Next returns the next iterator state which will contain a tx with the next
// smallest nonce of a randomly selected sender.
func (i *senderNonceMempoolIterator) Next() mempool.Iterator {
	for len(i.senders) > 0 {
		senderIndex := i.rnd.Intn(len(i.senders))
		sender := i.senders[senderIndex]
		// Check if it's from oracle list
		senderCursor, found := i.senderCursors[sender]
		if !found {
			i.senders = removeAtIndex(i.senders, senderIndex)
			continue
		}

		if nextCursor := senderCursor.Next(); nextCursor != nil {
			i.senderCursors[sender] = nextCursor
		} else {
			i.senders = removeAtIndex(i.senders, senderIndex)
		}

		return &senderNonceMempoolIterator{
			senders:       i.senders,
			currentTx:     senderCursor,
			rnd:           i.rnd,
			senderCursors: i.senderCursors,
		}
	}

	return nil
}

func (i *senderNonceMempoolIterator) Tx() sdk.Tx {
	return i.currentTx.Value.(sdk.Tx)
}

func removeAtIndex[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}
