package ante

import (
	"fmt"
	"math"

	expectedkeeper "github.com/classic-terra/core/v2/custom/auth/keeper"
	core "github.com/classic-terra/core/v2/types"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
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

	if ctx, err := fd.checkDeductFee(ctx, feeTx, taxes, simulate); err != nil {
		return ctx, err
	}

	newCtx = ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

func (fd FeeDecorator) checkDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, stabilityTaxes sdk.Coins, simulate bool) (newCtx sdk.Context, err error) {
	if addr := fd.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// in case the tx or the posthandler fails, we only want the user to pay the gas, not the tax
	// so we need to reduce the fee by the tax amount (not stabilityTax) here
	if !simulate {
		// log the whole ctx to see what is in there
		ctx.Logger().Info("ctx", "ctx", ctx)
		// also log the contents of the key value store
		fd.classictaxKeeper.Logger(ctx).Info("params", "params", fd.classictaxKeeper.GetParams(ctx).TaxableMsgTypes)
		ctx.Logger().Info("ctaxkep", "msgtypes", fd.classictaxKeeper.GetTaxableMsgTypes(ctx))
		_, feeGasOnly, err := fd.classictaxKeeper.CalculateSentTax(ctx, feeTx, stabilityTaxes, fd.treasuryKeeper, fd.oracleKeeper)

		if err != nil {
			return ctx, err
		}

		fee = feeGasOnly
	}

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if fd.feegrantKeeper == nil {
			return ctx, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := fd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, feeTx.GetMsgs())
			if err != nil {
				return ctx, sdkerrors.Wrapf(err, "%s does not not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := fd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !fee.IsZero() {
		err := DeductFees(fd.bankKeeper, ctx, deductFeesFromAcc, fee)
		if err != nil {
			return ctx, err
		}

		// this is now stability tax only, so no need to burn tax split
		if !stabilityTaxes.IsZero() && !simulate {
			if _, hasNeg := fee.SafeSub(stabilityTaxes...); hasNeg {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", fee, stabilityTaxes)
			}

			// Record tax proceeds, disabled for stability tax as since introduction of burn tax it was used for that purpose
			//fd.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return ctx, nil
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
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
		requiredGasFees, _ := fd.classictaxKeeper.GetFeeCoins(ctx, gas, stabilityTaxes, fd.oracleKeeper)
		requiredTaxFees, requiredTaxFeesUluna := fd.classictaxKeeper.GetTaxCoins(ctx, fd.treasuryKeeper, fd.oracleKeeper, msgs...)

		requiredTaxFeesUluna.SafeSub(sdk.NewCoin(core.MicroLunaDenom, requiredTaxFees.AmountOf(core.MicroLunaDenom)))

		requiredFees := requiredGasFees.Add(requiredTaxFees...).Add(stabilityTaxes...)

		// Check required fees
		if !requiredFees.IsZero() && !feeCoins.IsAnyGTE(requiredGasFees) {
			return 0, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(tax) + %q(stability)", feeCoins, requiredFees, requiredGasFees, requiredTaxFees, stabilityTaxes)
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
