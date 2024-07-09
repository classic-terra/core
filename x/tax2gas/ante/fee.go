package ante

import (
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tax2gasKeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
)

// FeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type FeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper ante.FeegrantKeeper
	treasuryKeeper types.TreasuryKeeper
	tax2gasKeeper  tax2gasKeeper.Keeper
}

func NewFeeDecorator(ak ante.AccountKeeper, bk types.BankKeeper, fk ante.FeegrantKeeper, tk types.TreasuryKeeper, taxKeeper tax2gasKeeper.Keeper) FeeDecorator {
	return FeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		treasuryKeeper: tk,
		tax2gasKeeper:  taxKeeper,
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
	if isOracleTx(msgs) {
		return next(ctx, tx, simulate)
	}

	// Compute taxes based on consumed gas
	gasConsumed := ctx.GasMeter().GasConsumed()
	gasConsumedTax, err := fd.tax2gasKeeper.ComputeTaxOnGasConsumed(ctx, tx, fd.treasuryKeeper, gasConsumed)
	if err != nil {
		return ctx, err
	}

	// Compute taxes based on sent amount
	taxes := tax2gasKeeper.FilterMsgAndComputeTax(ctx, fd.treasuryKeeper, msgs...)
	// Convert taxes to gas
	taxGas, err := fd.tax2gasKeeper.ComputeGas(ctx, tx, taxes)
	if err != nil {
		return ctx, err
	}

	if feeTx.GetGas()-gasConsumed < taxGas {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide enough gas to cover taxes")
	}

	if !simulate && !taxes.IsZero() {
		// Fee has to at least be enough to cover taxes + gasConsumedTax
		priority, err = fd.checkTxFee(ctx, tx, gasConsumedTax.Add(taxes...))
		if err != nil {
			return ctx, err
		}
	}

	// Try to deduct the gasConsumed tax
	feeDenom, err := fd.checkDeductFee(ctx, feeTx, gasConsumedTax, simulate)
	if err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority).WithValue(types.TaxGas, taxGas)
	if taxGas != 0 {
		newCtx = newCtx.WithValue(types.TaxGas, taxGas)
	}
	if !gasConsumedTax.IsZero() {
		newCtx = newCtx.WithValue(types.ConsumedGasFee, gasConsumedTax)
	}
	if feeDenom != "" {
		newCtx = newCtx.WithValue(types.FeeDenom, feeDenom)
	}
	return next(newCtx, tx, simulate)
}

func (fd FeeDecorator) checkDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, taxes sdk.Coins, simulate bool) (string, error) {
	if addr := fd.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return "", fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	feeCoins := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	var foundCoins sdk.Coins

	if !taxes.IsZero() {
		for _, coin := range feeCoins {
			found, requiredFee := taxes.Find(coin.Denom)
			if !found {
				continue
			}
			if coin.Amount.GT(requiredFee.Amount) {
				foundCoins = sdk.NewCoins(requiredFee)
			}
		}
	} else {
		return "", nil
	}

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if fd.feegrantKeeper == nil {
			return "", sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := fd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, foundCoins, feeTx.GetMsgs())
			if err != nil {
				return "", errorsmod.Wrapf(err, "%s does not not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := fd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return "", sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !foundCoins.IsZero() {
		err := DeductFees(fd.bankKeeper, ctx, deductFeesFromAcc, foundCoins)
		if err != nil {
			return "", err
		}
		if !foundCoins.IsZero() && !simulate {
			err := fd.BurnTaxSplit(ctx, foundCoins)
			if err != nil {
				return "", err
			}

			// Record tax proceeds
			fd.treasuryKeeper.RecordEpochTaxProceeds(ctx, foundCoins)
		}

		events := sdk.Events{
			sdk.NewEvent(
				sdk.EventTypeTx,
				sdk.NewAttribute(sdk.AttributeKeyFee, foundCoins.String()),
				sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
			),
		}
		ctx.EventManager().EmitEvents(events)

		// As there is only 1 element
		return foundCoins.Denoms()[0], nil
	} else {
		return "", fmt.Errorf("can't find coin that matches. Expected %s, wanted %s", feeCoins, taxes)
	}
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper authtypes.BankKeeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}

// checkTxFee implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
// Transaction with only oracle messages will skip gas fee check and will have the most priority.
// It also checks enough fee for treasury tax
func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx, taxes sdk.Coins) (int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()
	isOracleTx := isOracleTx(msgs)
	gasPrices := fd.tax2gasKeeper.GetGasPrices(ctx)

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	requiredGasFees := make(sdk.Coins, len(gasPrices))
	if ctx.IsCheckTx() && !isOracleTx {
		glDec := sdk.NewDec(int64(gas))
		for i, gp := range gasPrices {
			fee := gp.Amount.Mul(glDec)
			requiredGasFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}

		requiredFees := requiredGasFees.Add(taxes...)

		// Check required fees
		if !requiredFees.IsZero() && !feeCoins.IsAnyGTE(requiredFees) {
			return 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q = %q(gas) + %q(stability)", feeCoins, requiredFees, requiredGasFees, taxes)
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
