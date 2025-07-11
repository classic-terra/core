package ante

import (
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/classic-terra/core/v3/app/helper"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	taxexemptionkeeper "github.com/classic-terra/core/v3/x/taxexemption/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// FeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type FeeDecorator struct {
	accountKeeper      ante.AccountKeeper
	bankKeeper         BankKeeper
	feegrantKeeper     ante.FeegrantKeeper
	treasuryKeeper     TreasuryKeeper
	taxexemptionKeeper taxexemptionkeeper.Keeper
	distrKeeper        DistrKeeper
	taxKeeper          taxkeeper.Keeper
}

func NewFeeDecorator(ak ante.AccountKeeper, bk BankKeeper, fk ante.FeegrantKeeper, te taxexemptionkeeper.Keeper, tk TreasuryKeeper, dk DistrKeeper, th taxkeeper.Keeper) FeeDecorator {
	return FeeDecorator{
		accountKeeper:      ak,
		bankKeeper:         bk,
		feegrantKeeper:     fk,
		taxexemptionKeeper: te,
		treasuryKeeper:     tk,
		distrKeeper:        dk,
		taxKeeper:          th,
	}
}

func (fd FeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	var (
		priority int64
		err      error
	)

	msgs := feeTx.GetMsgs()
	// Compute taxes
	taxes, nonTaxableTaxes := FilterMsgAndComputeTax(ctx, fd.taxexemptionKeeper, fd.treasuryKeeper, fd.taxKeeper, simulate, msgs...)

	// check if the tx has paid fees for both(!) fee and tax
	// if not, then set the tax to zero at this point as it then is handled in the message route
	reverseCharge := false
	refundNonTaxableTax := false

	if !simulate {
		priority, reverseCharge, refundNonTaxableTax, err = fd.checkTxFee(ctx, tx, taxes, nonTaxableTaxes)
		if err != nil {
			return ctx, err
		}

		if !refundNonTaxableTax {
			nonTaxableTaxes = sdk.Coins{}
		}
	}

	if reverseCharge {
		// we don't have enough fees to pay for the gas and taxes
		// we set taxes to zero as they are handled in the message route
		taxes = sdk.Coins{}
	}

	newCtx, err := fd.checkDeductFee(ctx, feeTx, taxes, nonTaxableTaxes, simulate)
	if err != nil {
		return newCtx, err
	}

	newCtx = newCtx.WithPriority(priority).WithValue(taxtypes.ContextKeyTaxReverseCharge, reverseCharge)

	return next(newCtx, tx, simulate)
}

func (fd FeeDecorator) checkDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, taxes sdk.Coins, nonTaxableTaxes sdk.Coins, simulate bool) (sdk.Context, error) {
	if addr := fd.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if fd.feegrantKeeper == nil {
			return ctx, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := fd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, feeTx.GetMsgs())
			if err != nil {
				return ctx, errorsmod.Wrapf(err, "%s does not not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := fd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	feesOrTax := fee

	if simulate {
		if fee.IsZero() {
			feesOrTax = taxes
		}

		// even if fee is not zero it might be it is lower than the increased tax from computeTax
		// so we need to check if the tax is higher than the fee to not run into deduction errors
		for _, tax := range taxes {
			feeAmount := feesOrTax.AmountOf(tax.Denom)
			// if the fee amount is zero, add the tax amount to feesOrTax
			if feeAmount.IsZero() {
				feesOrTax = feesOrTax.Add(tax)
			} else if feeAmount.LT(tax.Amount) {
				// Update feesOrTax if the tax amount is higher
				missingAmount := tax.Amount.Sub(feeAmount)
				feesOrTax = feesOrTax.Add(sdk.NewCoin(tax.Denom, missingAmount))
			}
		}

		// a further issue arises from the fact that simulations are sometimes run with
		// the full bank balance of the account, which can lead to a situation where
		// the fees are deducted from the account during simulation and so the account
		// balance is not enough to complete the simulation.
		// So ONLY during simulation, we MINT the fees to the account to avoid this issue.
		// We only mint the fees we are adding on top of the original fee (sent by user).
		if !feesOrTax.IsZero() {
			needMint := feesOrTax.Sort().Sub(fee.Sort()...)
			if !needMint.IsZero() {
				err := fd.bankKeeper.MintCoins(ctx, minttypes.ModuleName, needMint)
				if err != nil {
					return ctx, err
				}

				// we need to add the fees to the account balance to avoid deduction errors
				err = fd.bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, deductFeesFromAcc.GetAddress(), needMint)
				if err != nil {
					return ctx, err
				}
			}
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}

	if !feesOrTax.IsZero() {
		// we will only deduct the fees from the account, not the tax
		// the tax will be deducted in the message route for reverse charge
		// or in the post handler for normal tax charge
		deductFees := feesOrTax.Sub(taxes...) // feesOrTax can never be lower than taxes
		if !nonTaxableTaxes.IsZero() {
			// if we have non-taxable taxes, we need to subtract them from the fees to be deducted
			for _, coin := range nonTaxableTaxes {
				if deductFees.AmountOf(coin.Denom).GTE(coin.Amount) {
					deductFees = deductFees.Sub(coin)

					// add the non-taxable taxes to the events
					events = append(events, sdk.NewEvent(
						taxtypes.EventTypeTaxRefund,
						sdk.NewAttribute(taxtypes.AttributeKeyTaxAmount, coin.String()),
					))
				}
			}
		}

		ctx = ctx.WithValue(taxtypes.ContextKeyTaxDue, taxes).WithValue(taxtypes.ContextKeyTaxPayer, deductFeesFrom.String())

		if !deductFees.IsZero() {
			err := DeductFees(fd.bankKeeper, ctx, deductFeesFromAcc, deductFees)
			if err != nil {
				return ctx, err
			}
		}
	}

	ctx.EventManager().EmitEvents(events)

	return ctx, nil
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}

// checkTxFee implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
// Transaction with only oracle messages will skip gas fee check and will have the most priority.
// It also checks enough fee for treasury tax
func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx, taxes sdk.Coins, nonTaxableTaxes sdk.Coins) (int64, bool, bool, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return 0, false, false, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()
	isOracleTx := helper.IsOracleTx(msgs)
	reverseCharge := false
	refundNonTaxableTaxes := false

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// Check if the transaction is an oracle transaction and skip gas fees for such transactions.
	if !isOracleTx {
		minRequiredGasFees := sdk.Coins{}
		minGasPrices := fd.taxKeeper.GetEffectiveGasPrices(ctx)
		if !minGasPrices.IsZero() {
			// Determine the required fees by multiplying each required minimum gas
			// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
			glDec := sdk.NewDec(int64(gas))
			minRequiredGasFees = make(sdk.Coins, len(minGasPrices))
			for i, gasPrice := range minGasPrices {
				fee := gasPrice.Amount.Mul(glDec)
				minRequiredGasFees[i] = sdk.NewCoin(gasPrice.Denom, fee.Ceil().RoundInt())
			}
		}

		remainingFees := feeCoins

		// Check if taxes are covered by the fees
		if !taxes.IsZero() {
			if !remainingFees.IsAllGTE(taxes) {
				// If the fees do not cover the taxes, reverse charge
				reverseCharge = true
			} else {
				remainingFees = remainingFees.Sub(taxes...)

				// Check if remaining fees cover gas after taxes
				if !minRequiredGasFees.IsZero() && !remainingFees.IsAnyGTE(minRequiredGasFees) {
					// If the remaining fees do not cover the gas fees, tax cannot be covered
					// So fall back to reverse charge
					reverseCharge = true
					remainingFees = feeCoins
				}
			}
		}

		// Attempt to refund non-taxable taxes
		if !nonTaxableTaxes.IsZero() && remainingFees.IsAllGTE(nonTaxableTaxes) {
			feeCoinsAfterTax := remainingFees.Sub(nonTaxableTaxes...)

			// Check if remaining fees cover gas after non-taxable taxes
			if !minRequiredGasFees.IsZero() && !feeCoinsAfterTax.IsAnyGTE(minRequiredGasFees) {
				// If the remaining fees do not cover the gas fees, non-taxable taxes cannot be refunded
				// We cannot reset to feeCoins as tax might have been deducted earlier
				refundNonTaxableTaxes = false
			} else {
				refundNonTaxableTaxes = true
				remainingFees = feeCoinsAfterTax
			}
		}

		// Check if the remaining paid fees are enough to cover the gas fees
		if !minRequiredGasFees.IsZero() && !remainingFees.IsAnyGTE(minRequiredGasFees) {
			return 0, reverseCharge, refundNonTaxableTaxes, errorsmod.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"insufficient fees; got: %q, required: %q(gas) [+ %q(tax)]",
				feeCoins, minRequiredGasFees, taxes,
			)
		}
	}

	priority := int64(math.MaxInt64)

	if !isOracleTx {
		priority = getTxPriority(feeCoins, int64(gas))
	}

	return priority, reverseCharge, refundNonTaxableTaxes, nil
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
