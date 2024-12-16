package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

const (
	DefaultMaxMemoLength     = 1024 // need 1024 to work with skip protocol
	DefaultMaxReceiverLength = 128
)

const ModuleName = "ibcspamprevention"

var (
	ErrReceiverTooLong = sdkerrors.Register(ModuleName, 11, "receiver too long")
	ErrMemoTooLong     = sdkerrors.Register(ModuleName, 12, "memo too long")
)

type IBCTransferSpamPreventionDecorator struct{}

func NewIBCTransferSpamPreventionDecorator() IBCTransferSpamPreventionDecorator {
	return IBCTransferSpamPreventionDecorator{}
}

// AnteHandle checks IBC transfer messages for potential spam characteristics
func (ispd IBCTransferSpamPreventionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		if ibcTransferMsg, ok := msg.(*ibctransfertypes.MsgTransfer); ok {
			if len(ibcTransferMsg.Memo) > DefaultMaxMemoLength {
				return ctx, ErrMemoTooLong
			}
			if len(ibcTransferMsg.Receiver) > DefaultMaxReceiverLength {
				return ctx, ErrReceiverTooLong
			}
		}
	}

	return next(ctx, tx, simulate)
}
