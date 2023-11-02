package ante

import (
	"math"

	expectedkeeper "github.com/classic-terra/core/v2/custom/auth/keeper"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
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

	var (
		priority int64
	)

	msgs := feeTx.GetMsgs()
	// Compute taxes
	taxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, fd.treasuryKeeper, msgs...)

	if !simulate {
		priority, err = fd.checkTxFee(ctx, tx, taxes)
		if err != nil {
			return ctx, err
		}
	}

	fee := feeTx.GetFee()
	// in case the tx or the posthandler fails, we only want the user to pay the gas, not the tax
	// so we need to reduce the fee by the tax amount (not stabilityTax) here
	if !simulate {
		_, feeGasOnly, err := fd.classictaxKeeper.CalculateSentTax(ctx, feeTx, taxes)

		if err != nil {
			return ctx, err
		}

		fee = feeGasOnly
	}

	if ctx, err := fd.classictaxKeeper.CheckDeductFee(ctx, feeTx, fee, taxes, simulate); err != nil {
		return ctx, err
	}

	newCtx = ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

// checkTxFee implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
// Transaction with only oracle messages will skip gas fee check and will have the most priority.
// It also checks enough fee for treasury tax
func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx, stabilityTaxes sdk.Coins) (int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return 0, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	msgs := feeTx.GetMsgs()
	isOracleTx := fd.classictaxKeeper.IsOracleTx(msgs)

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() && !isOracleTx {
		requiredGasFees, _ := fd.classictaxKeeper.GetFeeCoins(ctx, gas, stabilityTaxes)
		requiredTaxFees, requiredTaxFeesUluna := fd.classictaxKeeper.GetTaxCoins(ctx, msgs...)

		requiredFees := requiredGasFees.Sort()
		if !stabilityTaxes.IsZero() {
			requiredFees = requiredFees.Add(stabilityTaxes...)
		}

		// Check required fees
		// we ignore burn tax here as it is checked in the post handler
		if !requiredFees.IsZero() && !feeCoins.IsAnyGTE(requiredFees) {
			// add the tax to overall fees just for displaying it
			requiredTaxFees = requiredTaxFees.Sort()
			requiredFees.Add(requiredTaxFees...)
			return 0, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax)/%q(tax_uluna) + %q(stability)", feeCoins, requiredFees, requiredGasFees, requiredTaxFees, requiredTaxFeesUluna, stabilityTaxes)
		}
	}

	priority := int64(math.MaxInt64)

	if !isOracleTx {
		priority = getTxPriority(feeCoins, int64(gas))
	}

	return priority, nil
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
