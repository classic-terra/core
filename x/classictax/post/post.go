package post

import (
	core "github.com/classic-terra/core/v2/types"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	treasurykeeper "github.com/classic-terra/core/v2/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// ClassicTaxDecorator does post runMsg store
// modifications for dyncomm module
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

func (dd ClassicTaxDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	msgs := feeTx.GetMsgs()

	// the ante handler already deducted the fees for gas, but we now need to deduct tax

	// get tax from sent coins
	taxes, taxesUluna := dd.classictaxKeeper.GetTaxCoins(ctx, dd.treasuryKeeper, dd.oracleKeeper, msgs...)

	// get the parameter for gas prices
	gasPrice := dd.classictaxKeeper.GetGasFee(ctx)

	// get consumed gas
	requiredGas := ctx.GasMeter().GasConsumed()

	// get the gas fees
	taxGas := dd.classictaxKeeper.CalculateTaxGas(ctx, taxes, gasPrice)

	// we now need to increase the gas meter by the taxGas
	ctx.GasMeter().ConsumeGas(taxGas, "tax gas")

	allGas := requiredGas + taxGas

	ctx.Logger().Info("Gas", "required", requiredGas, "tax", taxGas, "total", allGas)

	if simulate {
		// we are done here as we just need to add the tax gas to the consumed gas
		return next(ctx, tx, simulate)
	}

	stabilityTaxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, dd.treasuryKeeper, msgs...)
	sentTaxFees, _, err := dd.classictaxKeeper.CalculateSentTax(ctx, feeTx, stabilityTaxes, dd.treasuryKeeper, dd.oracleKeeper)

	if err != nil {
		return ctx, err
	}

	distributeTax := taxes

	sentTaxFeesCoins, _ := sentTaxFees.TruncateDecimal()
	if taxes.IsAllGT(sentTaxFeesCoins) {
		// try uluna tax
		if taxesUluna.IsGTE(sdk.NewCoin(core.MicroLunaDenom, sentTaxFees.AmountOf(core.MicroLunaDenom).TruncateInt())) {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInsufficientFee, "insufficient tax sent")
		}

		distributeTax = sdk.NewCoins(taxesUluna)
	}

	err = dd.BurnTaxSplit(ctx, distributeTax)
	if err != nil {
		return ctx, err
	}

	// Record tax proceeds
	dd.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)

	return ctx, nil
}
