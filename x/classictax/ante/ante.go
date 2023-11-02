package ante

import (
	expectedkeeper "github.com/classic-terra/core/v2/custom/auth/keeper"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	"github.com/classic-terra/core/v2/x/classictax/types"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// FeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type FeeDecorator struct {
	accountKeeper    ante.AccountKeeper
	bankKeeper       expectedkeeper.BankKeeper
	feegrantKeeper   ante.FeegrantKeeper
	treasuryKeeper   expectedkeeper.TreasuryKeeper
	oracleKeeper     oraclekeeper.Keeper
	classictaxKeeper classictaxkeeper.Keeper
}

func NewClassicTaxFeeDecorator(ak ante.AccountKeeper, bk expectedkeeper.BankKeeper, fk ante.FeegrantKeeper, tk expectedkeeper.TreasuryKeeper, ok oraclekeeper.Keeper, ctk classictaxkeeper.Keeper) FeeDecorator {
	return FeeDecorator{
		accountKeeper:    ak,
		bankKeeper:       bk,
		feegrantKeeper:   fk,
		treasuryKeeper:   tk,
		oracleKeeper:     ok,
		classictaxKeeper: ctk,
	}
}

func (fd FeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	msgs := feeTx.GetMsgs()
	// Compute taxes
	stabilityTaxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, fd.treasuryKeeper, msgs...)
	fee := feeTx.GetFee()

	var (
		paidFeeCoins sdk.Coins
	)

	gasConsumed := ctx.GasMeter().GasConsumed()
	if !simulate {
		if err = fd.checkTxFee(ctx, tx, stabilityTaxes); err != nil {
			return ctx, err
		}

		requiredFee, _ := fd.classictaxKeeper.GetFeeCoins(ctx, gasConsumed, stabilityTaxes)
		taxCoins, taxCoinsUluna := fd.classictaxKeeper.GetTaxCoins(ctx, msgs...)

		// remove tax coins from sent fees
		availableFee, neg := fee.SafeSub(taxCoins...)
		if neg {
			availableFee, neg = fee.SafeSub(taxCoinsUluna)
			if neg {
				requiredFee = requiredFee.Add(taxCoins.Sort()...)
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", fee, requiredFee, requiredFee, taxCoins, taxCoinsUluna, stabilityTaxes)
			}
		}

		if !requiredFee.IsZero() {
			// we don't include the tax fees here
			if !availableFee.IsAnyGTE(requiredFee) {
				requiredFee = requiredFee.Add(taxCoins.Sort()...)
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", fee, requiredFee, requiredFee, taxCoins, taxCoinsUluna, stabilityTaxes)
			}

			// check if one of sent coins contains required fee
			// if yes, choose the first of those
			// if no, choose the first coin
			var feeCoin sdk.Coin

			for _, coin := range requiredFee {
				if found, foundCoin := availableFee.Sort().Find(coin.Denom); found {
					if foundCoin.IsGTE(coin) {
						feeCoin = coin
						break
					}
				}
			}

			if feeCoin.IsZero() {
				requiredFee = requiredFee.Add(taxCoins.Sort()...)
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", fee, requiredFee, requiredFee, taxCoins, taxCoinsUluna, stabilityTaxes)
			}

			paidFeeCoins = sdk.NewCoins(feeCoin)
			fd.classictaxKeeper.Logger(ctx).Info("AnteHandle", "sentgas", feeTx.GetGas(), "stability_tax", stabilityTaxes, "total", fee, "before", feeTx.GetFee(), "payfee", feeCoin, "simulate", simulate, "checktx", ctx.IsCheckTx(), "paidFeeCoins", paidFeeCoins)

			if ctx, err := fd.classictaxKeeper.CheckDeductFee(ctx, feeTx, paidFeeCoins, stabilityTaxes, simulate); err != nil {
				return ctx, err
			}
		}

		fd.classictaxKeeper.Logger(ctx).Info("End Antehandler", "sentgas", feeTx.GetGas(), "checktx", ctx.IsCheckTx(), "consumed", gasConsumed)
	}

	newCtx = ctx.WithValue(types.CtxFeeKey, paidFeeCoins)

	return next(newCtx, tx, simulate)
}

// checkTxFee implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
// Transaction with only oracle messages will skip gas fee check and will have the most priority.
// It also checks enough fee for treasury tax
func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx, stabilityTaxes sdk.Coins) error {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	//gas := feeTx.GetGas()
	usedGas := ctx.GasMeter().GasConsumed()

	msgs := feeTx.GetMsgs()
	isOracleTx := fd.classictaxKeeper.IsOracleTx(msgs)

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() && !isOracleTx {
		// this is the minimum gas fees (in coins) needed at this point,
		// based upon the consumed gas
		requiredGasFees, _ := fd.classictaxKeeper.GetFeeCoins(ctx, usedGas, stabilityTaxes)
		requiredTaxFees, requiredTaxFeesUluna := fd.classictaxKeeper.GetTaxCoins(ctx, msgs...)

		requiredFees := requiredGasFees.Sort()
		if !stabilityTaxes.IsZero() {
			requiredFees = requiredFees.Add(stabilityTaxes...)
		}

		// Check required fees
		// we ignore burn tax here as it is checked in the post handler
		if !requiredFees.IsZero() && !feeCoins.IsAnyGTE(requiredFees) {
			// add the tax to overall fees just for displaying it
			requiredFees = requiredFees.Add(requiredTaxFees.Sort()...)
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability), gas consumed: %d", feeCoins, requiredFees, requiredGasFees, requiredTaxFees, requiredTaxFeesUluna, stabilityTaxes, usedGas)
		}
	}

	return nil
}
