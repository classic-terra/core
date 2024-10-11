package handlers

import (
	"context"

	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BankMsgServer struct {
	banktypes.UnimplementedMsgServer
	taxKeeper      taxkeeper.Keeper
	bankKeeper     bankkeeper.Keeper
	treasuryKeeper treasurykeeper.Keeper
	messageServer  banktypes.MsgServer
}

func NewBankMsgServer(bankKeeper bankkeeper.Keeper, treasuryKeeper treasurykeeper.Keeper, taxKeeper taxkeeper.Keeper, messageServer banktypes.MsgServer) banktypes.MsgServer {
	return &BankMsgServer{
		bankKeeper:     bankKeeper,
		treasuryKeeper: treasuryKeeper,
		taxKeeper:      taxKeeper,
		messageServer:  messageServer,
	}
}

// Send handles MsgSend with tax deduction
func (s *BankMsgServer) Send(ctx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	fromAddr := sdk.MustAccAddressFromBech32(msg.FromAddress)

	if !s.treasuryKeeper.HasBurnTaxExemptionAddress(sdkCtx, msg.FromAddress, msg.ToAddress) {
		netAmount, err := s.taxKeeper.DeductTax(sdkCtx, fromAddr, msg.Amount)
		if err != nil {
			return nil, err
		}
		msg.Amount = netAmount
	}

	sdkCtx.Logger().Info("Custom Send handler altered the message", "newAmount", msg.Amount)

	return s.messageServer.Send(ctx, msg)
}

// MultiSend handles MsgMultiSend with tax deduction
func (s *BankMsgServer) MultiSend(ctx context.Context, msg *banktypes.MsgMultiSend) (*banktypes.MsgMultiSendResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	tainted := false
	for _, input := range msg.Inputs {
		if s.treasuryKeeper.HasBurnTaxExemptionAddress(sdkCtx, input.Address) {
			tainted = true
			break
		}
	}

	if !tainted {
		for i, input := range msg.Inputs {
			fromAddr := sdk.MustAccAddressFromBech32(input.Address)
			netCoins, err := s.taxKeeper.DeductTax(sdkCtx, fromAddr, input.Coins)
			if err != nil {
				return nil, err
			}
			msg.Inputs[i].Coins = netCoins
		}
	}

	sdkCtx.Logger().Info("Custom MultiSend handler altered the message", "newAmount", msg.Inputs)

	return s.messageServer.MultiSend(ctx, msg)
}
