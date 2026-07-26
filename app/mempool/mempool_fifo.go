package mempool

import (
	"context"
	"fmt"
	"sync"

	"github.com/classic-terra/core/v4/app/helper"
	"github.com/cometbft/cometbft/libs/clist"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var (
	_ mempool.Mempool  = (*FifoMempool)(nil)
	_ mempool.Iterator = (*fifoIterator)(nil)
)

var DefaultMaxTx = 5000

// FifoMempool is a mempool implementation that maintains two separate transaction pools:
// one for oracle transactions and another for regular transactions. Oracle transactions are given
// priority during iteration.
//
// Key characteristics:
// 1. Maintains two separate FIFO queues (CList) for transactions (oracle and regular)
// 2. Uses sync.Map for quick transaction lookup
// 3. During iteration:
//   - Oracle transactions are processed first in FIFO order
//   - Regular transactions follow in FIFO order
//
// 4. Transaction capacity is limited by maxTx (if > 0)
//
// Note: PrepareProposal may terminate iteration early if block size limits are reached.
type FifoMempool struct {
	mtx          cmtsync.RWMutex
	txs          *clist.CList // Regular transactions FIFO queue
	txsOracle    *clist.CList // Oracle transactions FIFO queue
	txsMap       sync.Map     // For quick lookup of existing transactions
	txsMapOracle sync.Map     // For quick lookup of existing transactions
	txsBytes     sync.Map     // For distinguishing replacements with the same sender and nonce
	maxTx        int
	txEncoder    sdk.TxEncoder
}

type FifoMempoolOptions func(mp *FifoMempool)

func NewFifoMempool(opts ...FifoMempoolOptions) *FifoMempool {
	mp := &FifoMempool{
		txs:       clist.New(),
		txsOracle: clist.New(),
		maxTx:     DefaultMaxTx,
	}

	for _, opt := range opts {
		opt(mp)
	}

	return mp
}

func FifoMaxTxOpt(maxTx int) FifoMempoolOptions {
	return func(mp *FifoMempool) {
		mp.maxTx = maxTx
	}
}

func FifoTxEncoderOpt(txEncoder sdk.TxEncoder) FifoMempoolOptions {
	return func(mp *FifoMempool) {
		mp.txEncoder = txEncoder
	}
}

func (mp *FifoMempool) Insert(_ context.Context, tx sdk.Tx) error {
	if mp.maxTx < 0 {
		return nil
	}

	// Marshal before taking the exclusive lock so CheckTx/RecheckTx admission
	// is not serialized behind proto encoding.
	txKey, err := getTxKey(tx)
	if err != nil {
		return err
	}
	txBytes, err := mp.getTxBytes(tx)
	if err != nil {
		return err
	}

	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	isOracle := helper.IsOracleTx(tx.GetMsgs())
	if elem, ok := mp.txsMap.Load(txKey); ok {
		if !isOracle {
			// In-place replacement requires an encoder to distinguish the new tx
			// from the old one at Remove time.  Without one, skip the update so
			// a later Remove of the superseded tx cannot accidentally evict the
			// current one.
			//
			// Mutating CElement.Value is safe only because the ABCI local client
			// serializes CheckTx against PrepareProposal, so no Select iterator
			// reads Value concurrently.  clist itself treats Value as write-once.
			if mp.txEncoder != nil {
				elem.(*clist.CElement).Value = tx
				mp.txsBytes.Store(txKey, txBytes)
			}
			return nil
		}
		mp.txsMap.Delete(txKey)
		mp.txsBytes.Delete(txKey)
		mp.txs.Remove(elem.(*clist.CElement))
	}
	if elem, ok := mp.txsMapOracle.Load(txKey); ok {
		if isOracle {
			if mp.txEncoder != nil {
				elem.(*clist.CElement).Value = tx
				mp.txsBytes.Store(txKey, txBytes)
			}
			return nil
		}
		mp.txsMapOracle.Delete(txKey)
		mp.txsBytes.Delete(txKey)
		mp.txsOracle.Remove(elem.(*clist.CElement))
	}

	totalTxs := mp.txs.Len() + mp.txsOracle.Len()
	if mp.maxTx >= 0 && totalTxs >= mp.maxTx {
		return mempool.ErrMempoolTxMaxCapacity
	}

	// Add to appropriate queue based on transaction type
	if isOracle {
		e := mp.txsOracle.PushBack(tx)
		mp.txsMapOracle.Store(txKey, e)
	} else {
		e := mp.txs.PushBack(tx)
		mp.txsMap.Store(txKey, e)
	}
	if txBytes != "" {
		mp.txsBytes.Store(txKey, txBytes)
	}

	return nil
}

func (mp *FifoMempool) Select(_ context.Context, _ [][]byte) mempool.Iterator {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	// Pre-allocate slice with exact capacity needed
	totalTxs := mp.txsOracle.Len() + mp.txs.Len()
	listTxKey := make([]customTxKey, 0, totalTxs)
	var newMapTxs sync.Map
	var newMapTxsOracle sync.Map
	for e := mp.txsOracle.Front(); e != nil; e = e.Next() {
		tx := e.Value.(sdk.Tx)
		txKey, _ := getTxKey(tx)
		listTxKey = append(listTxKey, txKey)
		newMapTxsOracle.Store(txKey, e)
	}
	for e := mp.txs.Front(); e != nil; e = e.Next() {
		tx := e.Value.(sdk.Tx)
		txKey, _ := getTxKey(tx)
		listTxKey = append(listTxKey, txKey)
		newMapTxs.Store(txKey, e)
	}

	iter := &fifoIterator{
		listTxKey:    listTxKey,
		mapTxs:       &newMapTxs,
		mapTxsOracle: &newMapTxsOracle,
	}
	return iter.Next()
}

type fifoIterator struct {
	currentTx    *clist.CElement
	listTxKey    []customTxKey
	mapTxs       *sync.Map
	mapTxsOracle *sync.Map
}

func (it *fifoIterator) Next() mempool.Iterator {
	// Return nil if we've processed all transactions
	if len(it.listTxKey) == 0 {
		return nil
	}

	// Get the next transaction key and remove it from the list
	txKey := it.listTxKey[0]
	it.listTxKey = it.listTxKey[1:]

	// Check oracle transactions first
	if elem, exists := it.mapTxsOracle.LoadAndDelete(txKey); exists {
		it.currentTx = elem.(*clist.CElement)
		return it
	}

	// Then check regular transactions
	if elem, exists := it.mapTxs.LoadAndDelete(txKey); exists {
		it.currentTx = elem.(*clist.CElement)
		return it
	}

	// If transaction was already removed, continue to next one
	return it.Next()
}

func (it *fifoIterator) Tx() sdk.Tx {
	return it.currentTx.Value.(sdk.Tx)
}

func (mp *FifoMempool) Remove(tx sdk.Tx) error {
	// Marshal before taking the exclusive lock (same rationale as Insert).
	txKey, err := getTxKey(tx)
	if err != nil {
		return err
	}
	txBytes, err := mp.getTxBytes(tx)
	if err != nil {
		return err
	}

	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	isOracle := helper.IsOracleTx(tx.GetMsgs())
	if isOracle {
		if elem, ok := mp.txsMapOracle.Load(txKey); ok {
			if !mp.matchesStoredTx(txKey, txBytes) {
				return nil
			}
			mp.txsMapOracle.Delete(txKey)
			mp.txsBytes.Delete(txKey)
			mp.txsOracle.Remove(elem.(*clist.CElement))
			return nil
		}
	} else {
		if elem, ok := mp.txsMap.Load(txKey); ok {
			if !mp.matchesStoredTx(txKey, txBytes) {
				return nil
			}
			mp.txsMap.Delete(txKey)
			mp.txsBytes.Delete(txKey)
			mp.txs.Remove(elem.(*clist.CElement))
			return nil
		}
	}

	return mempool.ErrTxNotFound
}

func (mp *FifoMempool) matchesStoredTx(txKey customTxKey, txBytes string) bool {
	storedTxBytes, ok := mp.txsBytes.Load(txKey)
	// No entry means no encoder was configured (bytes were never stored), so we
	// have no discriminator and must allow the removal.
	return !ok || storedTxBytes == txBytes
}

func (mp *FifoMempool) CountTx() int {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()
	return mp.txs.Len() + mp.txsOracle.Len()
}

func getTxKey(tx sdk.Tx) (customTxKey, error) {
	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return customTxKey{}, err
	}
	if len(sigs) == 0 {
		return customTxKey{}, fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence
	key := customTxKey{nonce: nonce, address: sender}
	return key, nil
}

func (mp *FifoMempool) getTxBytes(tx sdk.Tx) (string, error) {
	if mp.txEncoder == nil {
		return "", nil
	}

	txBytes, err := mp.txEncoder(tx)
	if err != nil {
		return "", err
	}

	return string(txBytes), nil
}

type customTxKey struct {
	address string
	nonce   uint64
}
