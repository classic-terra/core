package post

import (
	"math"

	core "github.com/classic-terra/core/v2/types"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	classictaxtypes "github.com/classic-terra/core/v2/x/classictax/types"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	treasurykeeper "github.com/classic-terra/core/v2/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// ClassicTaxDecorator does post runMsg store
// modifications for classictax module
type ClassicTaxDecorator struct {
	classictaxKeeper classictaxkeeper.Keeper
	treasuryKeeper   treasurykeeper.Keeper
	bankKeeper       bankkeeper.Keeper
	oracleKeeper     oraclekeeper.Keeper
}

func NewClassicTaxPostDecorator(dk classictaxkeeper.Keeper, tk treasurykeeper.Keeper, bk bankkeeper.Keeper, ok oraclekeeper.Keeper) ClassicTaxDecorator {
	return ClassicTaxDecorator{
		classictaxKeeper: dk,
		treasuryKeeper:   tk,
		bankKeeper:       bk,
		oracleKeeper:     ok,
	}
}

func (dd ClassicTaxDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	msgs := feeTx.GetMsgs()

	// the ante handler already deducted the fees for gas, but we now need to deduct tax and the remaining fees
	paidFees := ctx.Value(classictaxtypes.CtxFeeKey)
	paidFeeCoins, ok := paidFees.(sdk.Coins)
	if !ok {
		paidFeeCoins = sdk.NewCoins()
	}

	// get tax from sent coins
	taxes, taxesUluna, _, err := dd.classictaxKeeper.GetTaxCoins(ctx, msgs...)
	if err != nil {
		return ctx, err
	}

	// get the parameter for gas prices
	gasPrices := dd.classictaxKeeper.GetGasPrices(ctx)

	// get consumed gas
	requiredGas := ctx.GasMeter().GasConsumed()

	// get the gas fees
	taxGas, err := dd.classictaxKeeper.CalculateTaxGas(ctx, taxes, gasPrices)
	if err != nil {
		return ctx, err
	}

	if simulate {
		// we need to do this here as the rest of the function is not executed.
		// it will be done in the else block later
		ctx.GasMeter().ConsumeGas(taxGas, "tax gas")
		// we are done here as we just need to add the tax gas to the consumed gas
		return next(ctx, tx, simulate)
	}

	var (
		neg bool
	)

	// get the stability taxes here, although we are not letting the user pay it
	// we need it for calculations
	stabilityTaxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, dd.treasuryKeeper, msgs...)
	// requiredGasFees, _ := dd.classictaxKeeper.GetFeeCoins(ctx, requiredGas, stabilityTaxes)

	// sentTaxFees and sentTaxFeesUluna are _virtual_ values calculated by the assumed multiplier
	// they have to be checked against the actual sent fees
	sentTaxFees, sentTaxFeesUluna, remainingGasFee, remainingGas, err := dd.classictaxKeeper.CalculateSentTax(ctx, feeTx, stabilityTaxes)

	// store the value for later use
	remainingGasFeeAll := remainingGasFee

	// we also take the actual sent fees here and reduce it by the already paid fees
	availableSentFee := feeTx.GetFee()

	dd.classictaxKeeper.Logger(ctx).Info("RemainingGasFee", "sentFees", feeTx.GetFee(), "sentTaxFees", sentTaxFees, "remainingGasFee", remainingGasFee, "remainingGas", remainingGas)
	// if we already deducted fees in the ante handler, reduce the remaining gas fee by the fee already paid
	if paidFeeCoins.IsValid() {
		remainingGasFee, neg = remainingGasFee.SafeSub(paidFeeCoins...)
		if neg {
			remainingGasFee = sdk.NewCoins()
		}

		availableSentFee, neg = availableSentFee.SafeSub(paidFeeCoins...)
		if neg {
			availableSentFee = sdk.NewCoins()
		}
	}

	dd.classictaxKeeper.Logger(ctx).Info("After checking paidFeeCoins", "remainingGasFee", remainingGasFee, "paidFeeCoins", paidFeeCoins)

	if err != nil {
		return ctx, err
	}

	sentTaxFeesCoins, _ := sentTaxFees.TruncateDecimal()

	sentTaxFeesCoinsUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())
	if sentTaxFeesUluna.IsValid() {
		sentTaxFeesCoinsUluna, _ = sentTaxFeesUluna.TruncateDecimal()
	}

	// the available sent fees has to be maxed out at the virtually sent tax part as everything else accounts to gas only
	// for all denoms in the availableSentFee, check if the virtual tax fees are lower than the available sent fees
	// if so, set the available sent fees to the virtual tax fees
	// we need to handle it separately for uluna as there could be mixed taxes including uluna
	var (
		newAvailableSentFee sdk.Coins
		availableSentUluna  sdk.DecCoin
	)

	if !sentTaxFeesUluna.IsValid() || availableSentFee.AmountOf(core.MicroLunaDenom).GT(sentTaxFeesUluna.Amount.TruncateInt()) {
		availableSentUluna = sentTaxFeesUluna
	} else {
		availableSentUluna = sdk.NewDecCoin(core.MicroLunaDenom, availableSentFee.AmountOf(core.MicroLunaDenom))
	}

	for _, coin := range availableSentFee {
		if found, foundCoin := sentTaxFeesCoins.Find(coin.Denom); found {
			if foundCoin.IsLT(coin) {
				// If the virtual sent tax fee is lower, use it instead
				newAvailableSentFee = append(newAvailableSentFee, foundCoin)
			} else {
				newAvailableSentFee = append(newAvailableSentFee, coin)
			}
		}
	}
	availableSentFee = newAvailableSentFee

	distributeTax := taxes
	// can we pay in actual currency?
	if _, neg := availableSentFee.SafeSub(taxes...); neg {
		// if we have not enough coins in the sent fees for the tax, try to use only uluna for payment
		if taxesUluna.IsZero() || !sentTaxFeesCoinsUluna.IsGTE(taxesUluna) {
			// if uluna taxes are zero, we need to error as there is no exchange rate
			// need to remove stability from gas fees for error display

			remainingFeeSum := remainingGasFeeAll.Add(taxes...)
			if !stabilityTaxes.IsZero() {
				remainingFeeSum = remainingFeeSum.Add(stabilityTaxes...)
			}
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", feeTx.GetFee(), remainingFeeSum, remainingGasFeeAll, taxes, taxesUluna, stabilityTaxes)
		}

		// switch to uluna only
		// check if we have enough coins in the sent fees for the tax
		if availableSentUluna.IsLT(sdk.NewDecCoinFromCoin(taxesUluna)) {
			remainingFeeSum := remainingGasFeeAll.Add(taxes...)
			if !stabilityTaxes.IsZero() {
				remainingFeeSum = remainingFeeSum.Add(stabilityTaxes...)
			}
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", feeTx.GetFee(), remainingFeeSum, remainingGasFeeAll, taxes, taxesUluna, stabilityTaxes)
		}

		// we have enough coins in the sent fees for the tax in uluna, but not in other denom, so we can use this
		distributeTax = sdk.NewCoins(taxesUluna)
	}

	// send tax to fee collector
	if ctx, err := dd.classictaxKeeper.CheckDeductTax(ctx, feeTx, distributeTax, simulate); err != nil {
		return ctx, err
	}

	dd.classictaxKeeper.Logger(ctx).Info("Distribute tax", "taxes", distributeTax, "sent", sentTaxFees)

	// calculate the tx priority
	// we moved this from the ante handler as we need the actual gas and fees
	priority := int64(math.MaxInt64)

	if !dd.classictaxKeeper.IsOracleTx(msgs) {
		priority = getTxPriority(remainingGasFee, int64(remainingGas))
	}

	newCtx = ctx.WithPriority(priority)

	// deduct remaining fees if any
	if !remainingGasFee.IsZero() {
		if newCtx, err = dd.classictaxKeeper.CheckDeductFee(newCtx, feeTx, remainingGasFee, sdk.NewCoins(), simulate); err != nil {
			return newCtx, err
		}
	}

	if distributeTax.IsZero() {
		return next(newCtx, tx, simulate)
	}

	// send the burn tax to the distribution split function (burn/distribution[rewards/cp])
	err = dd.classictaxKeeper.BurnTaxSplit(newCtx, distributeTax)
	if err != nil {
		return newCtx, err
	}

	dd.classictaxKeeper.Logger(newCtx).Info("End Posthandler", "gas", feeTx.GetGas(), "checktx", newCtx.IsCheckTx(), "consumed", newCtx.GasMeter().GasConsumed())

	// we now need to increase the gas meter by the taxGas
	// TODO: check if we really want to do that here or not
	newCtx.GasMeter().ConsumeGas(taxGas, "tax gas")

	allGas := requiredGas + taxGas
	dd.classictaxKeeper.Logger(newCtx).Info("End Posthandler 2", "gas", feeTx.GetGas(), "checktx", newCtx.IsCheckTx(), "consumed", newCtx.GasMeter().GasConsumed(), "requiredGas", requiredGas, "taxGas", taxGas, "totalGas", allGas)

	return next(newCtx, tx, simulate)
}

// getTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it opens potential attack vectors
// where txs with multiple coins could not be prioritize as expected.
func getTxPriority(fee sdk.Coins, gas int64) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		if gas == 0 {
			p = 0
		} else {
			gasPrice := c.Amount.QuoRaw(gas)
			if gasPrice.IsInt64() {
				p = gasPrice.Int64()
			}
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
