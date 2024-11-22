package mempool

import (
	"context"
	"fmt"

	"github.com/classic-terra/core/v3/custom/auth/ante"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/huandu/skiplist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var (
	_ mempool.Mempool  = (*FifoSenderNonceMempool)(nil)
	_ mempool.Iterator = (*senderNonceMempoolIterator)(nil)
)

var DefaultMaxTx = 0

// FifoSenderNonceMempool is a mempool that prioritizes oracle transactions first,
// followed by regular transactions. Within each type, transactions are ordered by sender
// and nonce.
// The mempool is iterated by:
//
// 1) Maintaining separate lists of nonce-ordered txs per sender for both oracle and regular transactions
// 2) Processing all oracle transactions first, in order of sender address and nonce
// 3) After oracle transactions are exhausted, processing regular transactions in order of sender address and nonce
// 4) Repeat until the mempool is exhausted
//
// Note that PrepareProposal could choose to stop iteration before reaching the
// end if maxBytes is reached.
type FifoSenderNonceMempool struct {
	senders       map[string]*skiplist.SkipList
	maxTx         int
	existingTx    map[txKey]bool
	sendersOracle map[string]*skiplist.SkipList
}

type SenderNonceOptions func(mp *FifoSenderNonceMempool)

type txKey struct {
	address string
	nonce   uint64
}

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

	for _, opt := range opts {
		opt(snp)
	}

	return snp
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
		// Add oracle prefix to distinguish from regular transactions
		oracleSenders = append(oracleSenders, sender)
		oracleKey := "oracle:" + sender
		senderCursors[oracleKey] = snm.sendersOracle[sender].Front()
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
		regularKey := "regular:" + sender
		senderCursors[regularKey] = snm.senders[sender].Front()
		s = s.Next()
	}

	// Combine senders with oracle transactions first
	senders := append(oracleSenders, regularSenders...)

	iter := &senderNonceMempoolIterator{
		senders:       senders,
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

	if ante.IsOracleTx(tx.GetMsgs()) && found2 {
		res := senderOracleTxs.Remove(nonce)
		if res == nil {
			return mempool.ErrTxNotFound
		}

		if senderOracleTxs.Len() == 0 {
			delete(snm.senders, sender)
		}
	} else if found1 {
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
	currentTx     *skiplist.Element
	senders       []string
	senderCursors map[string]*skiplist.Element
}

// Next returns the next iterator state which will contain a tx with the next
// smallest nonce of a randomly selected sender.
func (i *senderNonceMempoolIterator) Next() mempool.Iterator {
	for len(i.senders) > 0 {
		senderIndex := 0
		sender := i.senders[senderIndex]
		isOracle := false
		checkKey := "oracle:" + sender
		_, ok := i.senderCursors[checkKey]
		if ok {
			isOracle = true
		}
		if isOracle {
			checkKey = "oracle:" + sender
			senderCursor, found := i.senderCursors[checkKey]
			if !found {
				i.senders = removeAtIndex(i.senders, senderIndex)
				continue
			}

			if nextCursor := senderCursor.Next(); nextCursor != nil {
				i.senderCursors[checkKey] = nextCursor
			} else {
				i.senders = removeAtIndex(i.senders, senderIndex)
				delete(i.senderCursors, checkKey)
			}

			return &senderNonceMempoolIterator{
				senders:       i.senders,
				currentTx:     senderCursor,
				senderCursors: i.senderCursors,
			}
		} else {
			checkKey = "regular:" + sender
			senderCursor, found := i.senderCursors[checkKey]
			if !found {
				i.senders = removeAtIndex(i.senders, senderIndex)
				continue
			}

			if nextCursor := senderCursor.Next(); nextCursor != nil {
				i.senderCursors[checkKey] = nextCursor
			} else {
				i.senders = removeAtIndex(i.senders, senderIndex)
				delete(i.senderCursors, checkKey)
			}

			return &senderNonceMempoolIterator{
				senders:       i.senders,
				currentTx:     senderCursor,
				senderCursors: i.senderCursors,
			}
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
