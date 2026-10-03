package mempool

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewParityCheckTxHandler returns a CheckTx handler that keeps the CometBFT
// mempool in line with the app-side mempool. trace reports whether errors
// should carry stack traces (BaseApp.Trace); it is read on every call because
// the BaseApp option setting it may be applied after this handler is built.
func NewParityCheckTxHandler(mp *FifoMempool, txDecoder sdk.TxDecoder, trace func() bool) sdk.CheckTxHandler {
	return func(runTx sdk.RunTx, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
		isRecheck := req.Type == abci.CheckTxType_Recheck

		// Undecodable txs are left to runTx, which returns the decode error.
		var tx sdk.Tx
		if decoded, err := txDecoder(req.Tx); err == nil {
			tx = decoded
		}

		// on ReCheckTx: evict txs from the CometBFT mempool that are no longer
		// in the app-side mempool (e.g. dropped by PrepareProposal)
		if isRecheck && tx != nil && !mp.Contains(tx) {
			return sdkerrors.ResponseCheckTxWithEvents(
				sdkerrors.ErrInvalidRequest.Wrap("tx is no longer in the app-side mempool"),
				0, 0, nil, trace(),
			), nil
		}

		before := mp.CountTx()
		gInfo, result, anteEvents, err := runTx(req.Tx, tx)
		if err != nil {
			// CometBFT drops the tx on error, so the app-side mempool must not
			// keep it either. runTx inserts before the post handler runs
			// (CheckTx) and removes only on ante failures (ReCheckTx), so a
			// post-handler failure would otherwise leave it behind. ABCI calls
			// are serialized, so a grown count means this tx was inserted.
			//
			// BaseApp has already written the ante state (sequence increment,
			// fees) to the check state at this point and offers no way to roll
			// it back, so after a post-handler failure a retry with the same
			// sequence is rejected until the next commit.
			if tx != nil && (isRecheck || mp.CountTx() > before) {
				_ = mp.Remove(tx)
			}
			return sdkerrors.ResponseCheckTxWithEvents(err, gInfo.GasWanted, gInfo.GasUsed, anteEvents, trace()), nil
		}

		return &abci.ResponseCheckTx{
			GasWanted: int64(gInfo.GasWanted),
			GasUsed:   int64(gInfo.GasUsed),
			Log:       result.Log,
			Data:      result.Data,
			Events:    sdk.MarkEventsToIndex(result.Events, nil),
		}, nil
	}
}
