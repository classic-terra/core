package keeper

import (
	"context"

	"cosmossdk.io/math"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// Oracle freshness check: ensure oracle prices are recent enough (time-based, not block-based)
	lastTallyTime := k.GetLastOracleTallyTime(ctx)
	if lastTallyTime > 0 {
		currentTime := ctx.BlockTime().Unix()
		maxAgeSeconds := int64(k.MaxOracleAgeSeconds(ctx))

		// Calculate time elapsed since last tally
		secondsSinceTally := currentTime - lastTallyTime

		if secondsSinceTally > maxAgeSeconds {
			return nil, types.ErrOraclePriceStale
		}
	}

	// Compute exchange rates between the ask and offer
	swapDecCoin, spread, err := k.ComputeSwap(ctx, offerCoin, askDenom)
	if err != nil {
		return nil, err
	}

	// TWAP deviation check: ensure current price doesn't deviate too much from TWAP
	// We need to check TWAP for both sides of the swap to prevent manipulation
	maxDeviation := k.MaxTwapDeviation(ctx)

	// Helper function to check TWAP for a denom
	checkTWAPDeviation := func(denom string) error {
		// For USTC, use MetaUSDDenom (USTC price). For others, use the denom itself (LUNC price in that currency)
		twapDenom := denom
		if denom == core.MicroUSDDenom {
			twapDenom = oracletypes.MetaUSDDenom // Use USTC price for USD swaps
		}

		currentPrice, err := k.OracleKeeper.GetLunaExchangeRate(ctx, twapDenom)
		if err == nil && currentPrice.IsPositive() {
			twapPrice, twapErr := k.ComputeTWAP(ctx, twapDenom)
			if twapErr == nil && twapPrice.IsPositive() {
				// Calculate deviation
				var deviation math.LegacyDec
				if currentPrice.GT(twapPrice) {
					deviation = currentPrice.Sub(twapPrice).Quo(twapPrice)
				} else {
					deviation = twapPrice.Sub(currentPrice).Quo(twapPrice)
				}

				if deviation.GT(maxDeviation) {
					return types.ErrTWAPDeviation
				}
			}
			// If no TWAP data yet, allow the swap (bootstrapping phase)
		}
		return nil
	}

	// Check TWAP for offer denom (if not LUNC)
	if offerCoin.Denom != core.MicroLunaDenom {
		if err := checkTWAPDeviation(offerCoin.Denom); err != nil {
			return nil, err
		}
	}

	// Check TWAP for ask denom (if not LUNC)
	if askDenom != core.MicroLunaDenom {
		if err := checkTWAPDeviation(askDenom); err != nil {
			return nil, err
		}
	}

	// If either side is LUNC, also check LUNC price (MicroUSDDenom = LUNC price in USD)
	if offerCoin.Denom == core.MicroLunaDenom || askDenom == core.MicroLunaDenom {
		if err := checkTWAPDeviation(core.MicroUSDDenom); err != nil {
			return nil, err
		}
	}

	// Charge a spread if applicable; the spread is burned
	var feeDecCoin sdk.DecCoin
	if spread.IsPositive() {
		feeDecCoin = sdk.NewDecCoinFromDec(swapDecCoin.Denom, spread.Mul(swapDecCoin.Amount))
	} else {
		feeDecCoin = sdk.NewDecCoin(swapDecCoin.Denom, math.ZeroInt())
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

	// Daily cap check: ensure pool balance deviation from baseline doesn't exceed daily limit
	// Check AFTER coins are in the pool but BEFORE sending out, with actual final amounts
	if err := k.CheckAndUpdateDailyCapForSwap(ctx, offerCoin, swapCoin); err != nil {
		return nil, err
	}

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

	// Split fee according to governance parameters: burn, community pool, remainder to oracle
	if feeCoin.IsPositive() {
		burnRate := k.SwapFeeBurnRate(ctx)
		cpRate := k.SwapFeeCommunityRate(ctx)
		// compute amounts: floor for burn and CP; remainder to oracle
		feeAmtDec := math.LegacyNewDecFromInt(feeCoin.Amount)
		burnAmt := burnRate.Mul(feeAmtDec).TruncateInt()
		cpAmt := cpRate.Mul(feeAmtDec).TruncateInt()
		oracleAmt := feeCoin.Amount.Sub(burnAmt).Sub(cpAmt)

		if burnAmt.IsPositive() {
			if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(feeCoin.Denom, burnAmt))); err != nil {
				return nil, err
			}
		}
		if cpAmt.IsPositive() {
			// Fund community pool from market module account
			if err := k.DistrKeeper.FundCommunityPool(ctx, sdk.NewCoins(sdk.NewCoin(feeCoin.Denom, cpAmt)), k.AccountKeeper.GetModuleAddress(types.ModuleName)); err != nil {
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
