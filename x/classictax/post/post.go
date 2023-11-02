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
	taxes, taxesUluna := dd.classictaxKeeper.GetTaxCoins(ctx, msgs...)

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

	stabilityTaxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, dd.treasuryKeeper, msgs...)
	sentTaxFees, remainingGasFee, remainingGas, err := dd.classictaxKeeper.CalculateSentTax(ctx, feeTx, stabilityTaxes)

	dd.classictaxKeeper.Logger(ctx).Info("RemainingGasFee", "sentFees", feeTx.GetFee(), "sentTaxFees", sentTaxFees, "remainingGasFee", remainingGasFee, "remainingGas", remainingGas)
	if paidFeeCoins.IsValid() {
		remainingGasFee, neg = remainingGasFee.SafeSub(paidFeeCoins...)
		if neg {
			remainingGasFee = sdk.NewCoins()
		}
	}

	dd.classictaxKeeper.Logger(ctx).Info("After checking paidFeeCoins", "remainingGasFee", remainingGasFee, "paidFeeCoins", paidFeeCoins)

	if err != nil {
		return ctx, err
	}

	distributeTax := taxes

	sentTaxFeesCoins, _ := sentTaxFees.TruncateDecimal()
	if !sentTaxFeesCoins.IsAllGT(taxes) {
		// try uluna tax
		sentUluna := sdk.NewCoin(core.MicroLunaDenom, sentTaxFees.AmountOf(core.MicroLunaDenom).TruncateInt())
		if !sentUluna.IsGTE(taxesUluna) {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient tax sent %q, required %q. reqgas %d, taxgas %d", sentTaxFees, taxes, requiredGas, taxGas)
		}

		distributeTax = sdk.NewCoins(taxesUluna)
	}

	// send tax to fee collector
	if ctx, err := dd.classictaxKeeper.CheckDeductTax(ctx, feeTx, distributeTax, simulate); err != nil {
		return ctx, err
	}

	dd.classictaxKeeper.Logger(ctx).Info("Distribute tax", "taxes", distributeTax, "sent", sentTaxFees)

	priority := int64(math.MaxInt64)

	if !dd.classictaxKeeper.IsOracleTx(msgs) {
		priority = getTxPriority(remainingGasFee, int64(remainingGas))
	}

	// deduct remaining feed if any
	if !remainingGasFee.IsZero() {
		if ctx, err := dd.classictaxKeeper.CheckDeductFee(ctx, feeTx, remainingGasFee, sdk.NewCoins(), simulate); err != nil {
			return ctx, err
		}
	}

	newCtx = ctx.WithPriority(priority)
	if distributeTax.IsZero() {
		return next(newCtx, tx, simulate)
	}

	err = dd.BurnTaxSplit(newCtx, distributeTax)
	if err != nil {
		return newCtx, err
	}

	// Record tax proceeds
	dd.treasuryKeeper.RecordEpochTaxProceeds(newCtx, taxes)

	dd.classictaxKeeper.Logger(newCtx).Info("End Posthandler", "gas", feeTx.GetGas(), "checktx", newCtx.IsCheckTx(), "consumed", newCtx.GasMeter().GasConsumed())

	// we now need to increase the gas meter by the taxGas
	newCtx.GasMeter().ConsumeGas(taxGas, "tax gas")
	allGas := requiredGas + taxGas
	dd.classictaxKeeper.Logger(newCtx).Info("Gas", "required", requiredGas, "tax", taxGas, "total", allGas)

	dd.classictaxKeeper.Logger(newCtx).Info("End Posthandler 2", "gas", feeTx.GetGas(), "checktx", newCtx.IsCheckTx(), "consumed", newCtx.GasMeter().GasConsumed())

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
		gasPrice := c.Amount.QuoRaw(gas)
		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
