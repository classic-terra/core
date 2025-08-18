package handlers

import (
	"context"

	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	taxexemptionkeeper "github.com/classic-terra/core/v3/x/taxexemption/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BankMsgServer struct {
	banktypes.UnimplementedMsgServer
	taxKeeper          taxkeeper.Keeper
	bankKeeper         bankkeeper.Keeper
	treasuryKeeper     treasurykeeper.Keeper
	messageServer      banktypes.MsgServer
	taxexemptionKeeper taxexemptionkeeper.Keeper
}

func NewBankMsgServer(bankKeeper bankkeeper.Keeper, taxexemptionKeeper taxexemptionkeeper.Keeper, treasuryKeeper treasurykeeper.Keeper, taxKeeper taxkeeper.Keeper, messageServer banktypes.MsgServer) banktypes.MsgServer {
	return &BankMsgServer{
		bankKeeper:         bankKeeper,
		treasuryKeeper:     treasuryKeeper,
		taxKeeper:          taxKeeper,
		messageServer:      messageServer,
		taxexemptionKeeper: taxexemptionKeeper,
	}
}

// Send handles MsgSend with tax deduction
func (s *BankMsgServer) Send(ctx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
		return s.messageServer.Send(ctx, msg)
	}

	fromAddr := sdk.MustAccAddressFromBech32(msg.FromAddress)

	if !s.taxexemptionKeeper.IsExemptedFromTax(sdkCtx, msg.FromAddress, msg.ToAddress) {
		netAmount, err := s.taxKeeper.DeductTax(sdkCtx, fromAddr, msg.Amount, false)
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

	if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
		return s.messageServer.MultiSend(ctx, msg)
	}

	tainted := false
	// make list of output addresses
	outputAddresses := make([]string, len(msg.Outputs))
	for i, output := range msg.Outputs {
		outputAddresses[i] = output.Address
	}
	for _, input := range msg.Inputs {
		if !s.taxexemptionKeeper.IsExemptedFromTax(sdkCtx, input.Address, outputAddresses...) {
			tainted = true
			break
		}
	}

	if tainted {
		for i, input := range msg.Inputs {
			fromAddr := sdk.MustAccAddressFromBech32(input.Address)
			netCoins, err := s.taxKeeper.DeductTax(sdkCtx, fromAddr, input.Coins, false)
			if err != nil {
				return nil, err
			}
			msg.Inputs[i].Coins = netCoins
		}

		for i, output := range msg.Outputs {
			toAddr := sdk.MustAccAddressFromBech32(output.Address)
			netCoins, err := s.taxKeeper.DeductTax(sdkCtx, toAddr, output.Coins, true)
			if err != nil {
				return nil, err
			}
			msg.Outputs[i].Coins = netCoins
		}
	}

	sdkCtx.Logger().Info("Custom MultiSend handler altered the message", "newAmount", msg.Inputs, msg.Outputs)

	return s.messageServer.MultiSend(ctx, msg)
}
