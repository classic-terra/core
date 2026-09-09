package handlers_test

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	apphelpers "github.com/classic-terra/core/v4/app/testing"
	core "github.com/classic-terra/core/v4/types"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	"github.com/classic-terra/core/v4/x/tax/handlers"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	"github.com/cometbft/cometbft/crypto"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
)

// stubMarketMsgServer captures the offer coin that the tax wrapper forwards to
// the inner (market) handler. This lets the test assert precisely how much tax
// was deducted from the offer before the swap is executed, without depending on
// the full swap/oracle/pool machinery.
type stubMarketMsgServer struct {
	markettypes.UnimplementedMsgServer
	gotOffer sdk.Coin
}

func (s *stubMarketMsgServer) Swap(_ context.Context, msg *markettypes.MsgSwap) (*markettypes.MsgSwapResponse, error) {
	s.gotOffer = msg.OfferCoin
	return &markettypes.MsgSwapResponse{}, nil
}

func (s *stubMarketMsgServer) SwapSend(_ context.Context, msg *markettypes.MsgSwapSend) (*markettypes.MsgSwapSendResponse, error) {
	s.gotOffer = msg.OfferCoin
	return &markettypes.MsgSwapSendResponse{}, nil
}

// newReverseChargeCtx returns a context with reverse-charge tax enabled.
// IsReverseCharge reads the context value via an unchecked .(bool) assertion,
// so the value MUST be set explicitly (otherwise the handler panics).
func newReverseChargeCtx(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(taxtypes.ContextKeyTaxReverseCharge, true)
}

// TestMarketMsgServer_Swap_ReverseCharge verifies that when reverse-charge tax
// is active, the Swap handler deducts tax from the offer coin before forwarding
// it to the inner market handler, and routes the deducted tax to the FeeCollector.
// This closes the coverage gap noted in review: previously only SwapSend had
// any tax-path coverage, and in fact neither Swap nor SwapSend exercised the
// reverse-charge branch.
func TestMarketMsgServer_Swap_ReverseCharge(t *testing.T) {
	chainID := "tax-reverse-swap-test"
	app := apphelpers.SetupApp(t, chainID)
	ctx := app.NewUncachedContext(false, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})

	// Configure a non-zero burn tax rate. The tax is skipped on the bond denom
	// (uluna), so use a non-bond offer denom (uusd) to observe a deduction.
	burnRate := sdkmath.LegacyNewDecWithPrec(1, 2) // 1%
	// SetupApp does not fully populate treasury params, so initialize them here
	// (otherwise GetTaxCap decodes a nil entry and panics inside ComputeTax).
	app.TreasuryKeeper.SetParams(ctx, treasurytypes.DefaultParams())
	require.NoError(t, app.TaxKeeper.SetParams(ctx, taxtypes.Params{BurnTaxRate: burnRate}))
	// ProcessTaxSplits reads distribution params; initialize them too.
	app.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams())
	app.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())

	// Trader offers 1_000_000 uusd. Expected tax at 1% = 10_000 uusd.
	offer := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(1_000_000))
	trader := sdk.AccAddress(crypto.AddressHash([]byte("trader")))

	// Fund the trader so DeductTax can debit the tax to FeeCollector.
	// The treasury module holds the Minter permission in the app, so mint via it.
	require.NoError(t, app.BankKeeper.MintCoins(ctx, treasurytypes.ModuleName, sdk.NewCoins(offer)))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, treasurytypes.ModuleName, trader, sdk.NewCoins(offer)))

	// Build the wrapped server with a stub inner handler.
	stub := &stubMarketMsgServer{}
	wrapped := handlers.NewMarketMsgServer(app.MarketKeeper, app.TreasuryKeeper, app.TaxKeeper, stub)

	msg := &markettypes.MsgSwap{
		Trader:    trader.String(),
		OfferCoin: offer,
		AskDenom:  core.MicroLunaDenom,
	}

	_, err := wrapped.Swap(newReverseChargeCtx(ctx), msg)
	require.NoError(t, err)

	// The inner handler must have received the offer net of tax.
	expectedNet := offer.Amount.Sub(sdkmath.NewInt(10_000))
	require.Equal(t, expectedNet, stub.gotOffer.Amount, "inner handler should receive offer net of tax")
	require.Equal(t, core.MicroUSDDenom, stub.gotOffer.Denom)

	// The trader was debited the tax (the stub does not consume the forwarded
	// offer, so the only balance change is the tax that DeductTax pulled out and
	// ProcessTaxSplits then redistributed to burn/oracle/community/accumulator).
	traderBalanceAfter := app.BankKeeper.GetBalance(ctx, trader, core.MicroUSDDenom)
	require.Equal(t, offer.Amount.Sub(sdkmath.NewInt(10_000)), traderBalanceAfter.Amount,
		"trader should be debited exactly the tax amount")
}

// TestMarketMsgServer_Swap_NoReverseCharge verifies that without reverse-charge
// tax the handler forwards the offer coin unchanged (no deduction).
func TestMarketMsgServer_Swap_NoReverseCharge(t *testing.T) {
	chainID := "tax-reverse-swap-nocharge-test"
	app := apphelpers.SetupApp(t, chainID)
	ctx := app.NewUncachedContext(false, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})

	burnRate := sdkmath.LegacyNewDecWithPrec(1, 2) // 1%
	// SetupApp does not fully populate treasury params, so initialize them here
	// (otherwise GetTaxCap decodes a nil entry and panics inside ComputeTax).
	app.TreasuryKeeper.SetParams(ctx, treasurytypes.DefaultParams())
	require.NoError(t, app.TaxKeeper.SetParams(ctx, taxtypes.Params{BurnTaxRate: burnRate}))

	offer := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(1_000_000))
	trader := sdk.AccAddress(crypto.AddressHash([]byte("trader")))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, treasurytypes.ModuleName, sdk.NewCoins(offer)))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, treasurytypes.ModuleName, trader, sdk.NewCoins(offer)))

	stub := &stubMarketMsgServer{}
	wrapped := handlers.NewMarketMsgServer(app.MarketKeeper, app.TreasuryKeeper, app.TaxKeeper, stub)

	msg := &markettypes.MsgSwap{
		Trader:    trader.String(),
		OfferCoin: offer,
		AskDenom:  core.MicroLunaDenom,
	}

	// Reverse charge disabled: IsReverseCharge must see a false value, not nil.
	noChargeCtx := ctx.WithValue(taxtypes.ContextKeyTaxReverseCharge, false)
	_, err := wrapped.Swap(noChargeCtx, msg)
	require.NoError(t, err)

	// Offer forwarded unchanged.
	require.Equal(t, offer, stub.gotOffer, "offer should be unchanged when reverse charge is off")
}

// TestMarketMsgServer_SwapSend_ReverseCharge is the SwapSend equivalent of the
// Swap reverse-charge test, covering the second intercepted handler.
func TestMarketMsgServer_SwapSend_ReverseCharge(t *testing.T) {
	chainID := "tax-reverse-swap-send-test"
	app := apphelpers.SetupApp(t, chainID)
	ctx := app.NewUncachedContext(false, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})

	burnRate := sdkmath.LegacyNewDecWithPrec(1, 2) // 1%
	// SetupApp does not fully populate treasury params, so initialize them here
	// (otherwise GetTaxCap decodes a nil entry and panics inside ComputeTax).
	app.TreasuryKeeper.SetParams(ctx, treasurytypes.DefaultParams())
	require.NoError(t, app.TaxKeeper.SetParams(ctx, taxtypes.Params{BurnTaxRate: burnRate}))
	// ProcessTaxSplits reads distribution params; initialize them too.
	app.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams())
	app.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())

	offer := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(1_000_000))
	sender := sdk.AccAddress(crypto.AddressHash([]byte("sender")))
	receiver := sdk.AccAddress(crypto.AddressHash([]byte("receiver")))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, treasurytypes.ModuleName, sdk.NewCoins(offer)))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, treasurytypes.ModuleName, sender, sdk.NewCoins(offer)))

	stub := &stubMarketMsgServer{}
	wrapped := handlers.NewMarketMsgServer(app.MarketKeeper, app.TreasuryKeeper, app.TaxKeeper, stub)

	msg := &markettypes.MsgSwapSend{
		FromAddress: sender.String(),
		ToAddress:   receiver.String(),
		OfferCoin:   offer,
		AskDenom:    core.MicroLunaDenom,
	}

	_, err := wrapped.SwapSend(newReverseChargeCtx(ctx), msg)
	require.NoError(t, err)

	expectedNet := offer.Amount.Sub(sdkmath.NewInt(10_000))
	require.Equal(t, expectedNet, stub.gotOffer.Amount, "inner handler should receive offer net of tax")

	// The sender was debited the tax (the stub does not consume the forwarded
	// offer, so the only balance change is the tax DeductTax pulled out).
	senderBalanceAfter := app.BankKeeper.GetBalance(ctx, sender, core.MicroUSDDenom)
	require.Equal(t, offer.Amount.Sub(sdkmath.NewInt(10_000)), senderBalanceAfter.Amount,
		"sender should be debited exactly the tax amount")
}
