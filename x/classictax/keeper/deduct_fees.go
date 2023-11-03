package keeper

import (
	"fmt"

	"github.com/classic-terra/core/v2/x/classictax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// function to get the account to deduct fees from
func (k Keeper) getDeductFromAcc(ctx sdk.Context, feeTx sdk.FeeTx) (authtypes.AccountI, error) {
	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if k.feegrantKeeper == nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := k.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, feeTx.GetMsgs())
			if err != nil {
				return nil, sdkerrors.Wrapf(err, "%s does not not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := k.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return nil, sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	return deductFeesFromAcc, nil
}

func (k Keeper) CheckDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, fee sdk.Coins, stabilityTaxes sdk.Coins, simulate bool) (newCtx sdk.Context, err error) {
	if addr := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	deductFeesFrom := feeTx.FeePayer()
	deductFeesFromAcc, err := k.getDeductFromAcc(ctx, feeTx)
	if err != nil {
		return ctx, err
	}

	// deduct the fees
	if !fee.IsZero() {
		if !stabilityTaxes.IsZero() && !simulate {
			newFee, hasNeg := fee.SafeSub(stabilityTaxes...)

			if hasNeg {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s(stability)", fee, stabilityTaxes)
			}

			fee = newFee

			err := DeductFees(k.bankKeeper, ctx, deductFeesFromAcc, stabilityTaxes, simulate)
			if err != nil {
				return ctx, err
			}
			// this is now stability tax only, so no need to burn tax split
			// Record tax proceeds, disabled for stability tax as since introduction of burn tax it was used for that purpose
			//fd.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)
		}

		err := DeductFees(k.bankKeeper, ctx, deductFeesFromAcc, fee, simulate)
		if err != nil {
			return ctx, err
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

func (k Keeper) CheckDeductTax(ctx sdk.Context, feeTx sdk.FeeTx, tax sdk.Coins, simulate bool) (newCtx sdk.Context, err error) {
	if addr := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	deductFeesFrom := feeTx.FeePayer()
	deductFeesFromAcc, err := k.getDeductFromAcc(ctx, feeTx)
	if err != nil {
		return ctx, err
	}

	// deduct the tax amount
	if !tax.IsZero() {
		err := DeductFees(k.bankKeeper, ctx, deductFeesFromAcc, tax, simulate)
		if err != nil {
			return ctx, err
		}

		k.treasuryKeeper.RecordEpochTaxProceeds(ctx, tax)
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(types.AttributeKeyTax, tax.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return ctx, nil
}

// DeductFees deducts fees from the given account or throws an error.
func DeductFees(bankKeeper authtypes.BankKeeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins, simulate bool) error {
	if !fees.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees)
	ctx.Logger().Info("DeductFees", "acc", acc.GetAddress(), "fees", fees, "module", types.ModuleName, "checktx", ctx.IsCheckTx(), "simulate", simulate)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}
