package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	core "github.com/classic-terra/core/v3/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the market MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, err
	}

	return k.handleSwapRequest(ctx, addr, addr, msg.OfferCoin, msg.AskDenom)
}

func (k msgServer) SwapSend(goCtx context.Context, msg *types.MsgSwapSend) (*types.MsgSwapSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	fromAddr, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, err
	}

	toAddr, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}

	res, err := k.handleSwapRequest(ctx, fromAddr, toAddr, msg.OfferCoin, msg.AskDenom)
	if err != nil {
		return nil, err
	}

	return &types.MsgSwapSendResponse{
		SwapCoin: res.SwapCoin,
		SwapFee:  res.SwapFee,
	}, nil
}

// handleMsgSwap handles the logic of a MsgSwap
// This function does not repeat checks that have already been performed in msg.ValidateBasic()
// Ex) assert(offerCoin.Denom != askDenom)
func (k msgServer) handleSwapRequest(ctx sdk.Context,
	trader sdk.AccAddress, receiver sdk.AccAddress,
	offerCoin sdk.Coin, askDenom string,
) (*types.MsgSwapResponse, error) {
	// Only allow swaps between uluna and denoms in the allowed set (spread-fee path only)
	if !((offerCoin.Denom == core.MicroLunaDenom && k.isAllowedSwapDenom(askDenom)) ||
		(askDenom == core.MicroLunaDenom && k.isAllowedSwapDenom(offerCoin.Denom))) {
		return nil, types.ErrInvalidSwapPair
	}

	// Oracle guard: require Luna/USD meta rate to be present
	if _, err := k.OracleKeeper.GetLunaExchangeRate(ctx, oracletypes.MetaUSDDenom); err != nil {
		return nil, types.ErrNoEffectivePrice
	}

	// Compute exchange rates between the ask and offer
	swapDecCoin, spread, err := k.ComputeSwap(ctx, offerCoin, askDenom)
	if err != nil {
		return nil, err
	}

	// Charge a spread if applicable; the spread is burned
	var feeDecCoin sdk.DecCoin
	if spread.IsPositive() {
		feeDecCoin = sdk.NewDecCoinFromDec(swapDecCoin.Denom, spread.Mul(swapDecCoin.Amount))
	} else {
		feeDecCoin = sdk.NewDecCoin(swapDecCoin.Denom, sdk.ZeroInt())
	}

	// Subtract fee from the swap coin
	swapDecCoin.Amount = swapDecCoin.Amount.Sub(feeDecCoin.Amount)

	// Update pool delta
	if err := k.ApplySwapToPool(ctx, offerCoin, swapDecCoin); err != nil {
		return nil, err
	}

	// Send offer coins to module account
	offerCoins := sdk.NewCoins(offerCoin)
	err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, types.ModuleName, offerCoins)
	if err != nil {
		return nil, err
	}

	// Determine amounts to transfer out of the pool
	swapCoin, decimalCoin := swapDecCoin.TruncateDecimal()

	// Ensure to fail the swap tx when zero swap coin
	if !swapCoin.IsPositive() {
		return nil, types.ErrZeroSwapCoin
	}

	feeDecCoin = feeDecCoin.Add(decimalCoin) // add truncated decimalCoin to swapFee
	feeCoin, _ := feeDecCoin.TruncateDecimal()

	// Check pool liquidity for ask denom: must cover swapCoin + feeCoin (fee will be split out)
	poolBal := k.BankKeeper.GetBalance(ctx, k.AccountKeeper.GetModuleAddress(types.ModuleName), swapCoin.Denom)
	requiredOut := swapCoin.Amount.Add(feeCoin.Amount)
	if poolBal.Amount.LT(requiredOut) {
		return nil, types.ErrInsufficientLiquidity
	}

	// Transfer swap coin to receiver
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, sdk.NewCoins(sdk.NewCoin(swapCoin.Denom, swapCoin.Amount))); err != nil {
		return nil, err
	}

	// Split fee: 50% burn, 50% to oracle
	// TODO: this should be a governance parameter, maybe also including community pool
	if feeCoin.IsPositive() {
		half := feeCoin.Amount.QuoRaw(2)
		burnAmt := half
		oracleAmt := feeCoin.Amount.Sub(half) // remainder to oracle to avoid dust issues

		if burnAmt.IsPositive() {
			if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(feeCoin.Denom, burnAmt))); err != nil {
				return nil, err
			}
		}
		if oracleAmt.IsPositive() {
			if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, oracletypes.ModuleName, sdk.NewCoins(sdk.NewCoin(feeCoin.Denom, oracleAmt))); err != nil {
				return nil, err
			}
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventSwap,
			sdk.NewAttribute(types.AttributeKeyOffer, offerCoin.String()),
			sdk.NewAttribute(types.AttributeKeyTrader, trader.String()),
			sdk.NewAttribute(types.AttributeKeyRecipient, receiver.String()),
			sdk.NewAttribute(types.AttributeKeySwapCoin, swapCoin.String()),
			sdk.NewAttribute(types.AttributeKeySwapFee, feeCoin.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})

	return &types.MsgSwapResponse{
		SwapCoin: swapCoin,
		SwapFee:  feeCoin,
	}, nil
}
