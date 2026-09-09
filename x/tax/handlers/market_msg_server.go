package handlers

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	marketkeeper "github.com/classic-terra/core/v4/x/market/keeper"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	taxkeeper "github.com/classic-terra/core/v4/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v4/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type MarketMsgServer struct {
	markettypes.UnimplementedMsgServer
	taxKeeper      taxkeeper.Keeper
	marketKeeper   marketkeeper.Keeper
	treasuryKeeper treasurykeeper.Keeper
	messageServer  markettypes.MsgServer
}

func NewMarketMsgServer(marketKeeper marketkeeper.Keeper, treasuryKeeper treasurykeeper.Keeper, taxKeeper taxkeeper.Keeper, messageServer markettypes.MsgServer) markettypes.MsgServer {
	return &MarketMsgServer{
		taxKeeper:      taxKeeper,
		marketKeeper:   marketKeeper,
		treasuryKeeper: treasuryKeeper,
		messageServer:  messageServer,
	}
}

// SwapSend handles MsgSwapSend with tax deduction
func (s *MarketMsgServer) SwapSend(ctx context.Context, msg *markettypes.MsgSwapSend) (*markettypes.MsgSwapSendResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
		return s.messageServer.SwapSend(ctx, msg)
	}

	sender := sdk.MustAccAddressFromBech32(msg.FromAddress)

	netOfferCoin, err := s.taxKeeper.DeductTax(sdkCtx, sender, sdk.NewCoins(msg.OfferCoin), false)
	if err != nil {
		return nil, err
	}
	if len(netOfferCoin) == 0 {
		// The whole offer was consumed by the tax; there is nothing left to swap.
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "offer coin %s is fully consumed by tax", msg.OfferCoin)
	}
	msg.OfferCoin = netOfferCoin[0]

	return s.messageServer.SwapSend(ctx, msg)
}

// Swap handles MsgSwap with tax deduction
func (s *MarketMsgServer) Swap(ctx context.Context, msg *markettypes.MsgSwap) (*markettypes.MsgSwapResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
		return s.messageServer.Swap(ctx, msg)
	}

	sender := sdk.MustAccAddressFromBech32(msg.Trader)
	netOfferCoin, err := s.taxKeeper.DeductTax(sdkCtx, sender, sdk.NewCoins(msg.OfferCoin), false)
	if err != nil {
		return nil, err
	}
	if len(netOfferCoin) == 0 {
		// The whole offer was consumed by the tax; there is nothing left to swap.
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "offer coin %s is fully consumed by tax", msg.OfferCoin)
	}
	msg.OfferCoin = netOfferCoin[0]

	return s.messageServer.Swap(ctx, msg)
}
