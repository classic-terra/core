package handlers

import (
	"context"

	marketkeeper "github.com/classic-terra/core/v3/x/market/keeper"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	sender := sdk.MustAccAddressFromBech32(msg.FromAddress)

	netOfferCoin, err := s.taxKeeper.DeductTax(sdkCtx, sender, sdk.NewCoins(msg.OfferCoin))
	if err != nil {
		return nil, err
	}
	msg.OfferCoin = netOfferCoin[0]

	return s.messageServer.SwapSend(ctx, msg)
}
