package ante

import (
	"fmt"
	"math"

	tmstrings "github.com/cometbft/cometbft/libs/strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tax2gasKeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
	tax2gasutils "github.com/classic-terra/core/v3/x/tax2gas/utils"
)

// FeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type FeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper types.FeegrantKeeper
	treasuryKeeper types.TreasuryKeeper
	tax2gasKeeper  tax2gasKeeper.Keeper
}

func NewFeeDecorator(ak ante.AccountKeeper, bk types.BankKeeper, fk types.FeegrantKeeper, tk types.TreasuryKeeper, taxKeeper tax2gasKeeper.Keeper) FeeDecorator {
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
	if tax2gasutils.IsOracleTx(msgs) || !fd.tax2gasKeeper.IsEnabled(ctx) {
		return next(ctx, tx, simulate)
	}

	// Check if the gas price node set is larger than the current gas price
	// it will be the new gas price
	gasPrices := fd.GetFinalGasPrices(ctx)
	// Compute taxes based on consumed gas
	gasConsumed := ctx.GasMeter().GasConsumed()
	gasConsumedFees, err := tax2gasutils.ComputeFeesOnGasConsumed(tx, gasPrices, sdk.NewInt(int64(gasConsumed)))
	if err != nil {
		return ctx, err
	}

	// Compute taxes based on sent amount
	burnTaxRate := fd.tax2gasKeeper.GetBurnTaxRate(ctx)
	taxes := tax2gasutils.FilterMsgAndComputeTax(ctx, fd.treasuryKeeper, burnTaxRate, msgs...)
	// Convert taxes to gas
	taxGas, err := tax2gasutils.ComputeGas(gasPrices, taxes)
	if err != nil {
		return ctx, err
	}

	// Bypass min fee requires:
	// 	- the tx contains only message types that can bypass the minimum fee,
	//	see BypassMinFeeMsgTypes;
	//	- the total gas limit per message does not exceed MaxTotalBypassMinFeeMsgGasUsage,
	//	i.e., totalGas <=  MaxTotalBypassMinFeeMsgGasUsage
	// Otherwise, minimum fees and global fees are checked to prevent spam.
	maxTotalBypassMinFeeMsgGasUsage := fd.tax2gasKeeper.GetMaxTotalBypassMinFeeMsgGasUsage(ctx)
	doesNotExceedMaxGasUsage := feeTx.GetGas() <= maxTotalBypassMinFeeMsgGasUsage
	allBypassMsgs := fd.ContainsOnlyBypassMinFeeMsgs(ctx, msgs)
	allowedToBypassMinFee := allBypassMsgs && doesNotExceedMaxGasUsage

	if allowedToBypassMinFee {
		return next(ctx, tx, simulate)
	}

	if !simulate {
		isOracleTx := tax2gasutils.IsOracleTx(msgs)
		// the priority to be added in mempool is based on
		// the tax gas that user need to pay
		priority = int64(math.MaxInt64)
		if !isOracleTx {
			priority = int64(1)
		}
	}

	// Try to deduct the gasConsumed fees
	paidDenom, err := fd.tryDeductFee(ctx, feeTx, gasConsumedFees, simulate)
	if err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority).
		WithValue(types.TaxGas, taxGas).
		WithValue(types.FinalGasPrices, gasPrices)
	if !taxGas.IsZero() {
		gasMeter, ok := ctx.GasMeter().(*types.Tax2GasMeter)
		if !ok {
			return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidType, "invalid gas meter")
		}
		gasMeter.ConsumeTax(taxGas, "ante handler taxGas")
	}
	newCtx = newCtx.WithValue(types.AnteConsumedGas, gasConsumed)
	if paidDenom != "" {
		newCtx = newCtx.WithValue(types.PaidDenom, paidDenom)
	}

	return next(newCtx, tx, simulate)
}

func (fd FeeDecorator) tryDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, taxes sdk.Coins, simulate bool) (string, error) {
	if addr := fd.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return "", fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	feeCoins := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	foundCoins := sdk.Coins{}
	if !taxes.IsZero() {
		for _, coin := range feeCoins {
			found, requiredFee := taxes.Find(coin.Denom)
			if !found {
				continue
			}
			if coin.Amount.GTE(requiredFee.Amount) {
				foundCoins = foundCoins.Add(requiredFee)
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
			allowance, err := fd.feegrantKeeper.GetAllowance(ctx, feeGranter, feePayer)
			if err != nil {
				return "", errorsmod.Wrapf(err, "fee-grant not found with granter %s and grantee %s", feeGranter, feePayer)
			}

			granted := false
			for _, foundCoin := range foundCoins {
				_, err := allowance.Accept(ctx, sdk.NewCoins(foundCoin), feeTx.GetMsgs())
				if err == nil {
					foundCoins = sdk.NewCoins(foundCoin)
					granted = true
					err = fd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, foundCoins, feeTx.GetMsgs())
					if err != nil {
						return "", errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
					}
					break
				}
			}

			if !granted {
				return "", errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
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
		foundCoins, err := DeductFees(fd.bankKeeper, ctx, deductFeesFromAcc, foundCoins)
		if err != nil {
			return "", err
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
	}
	if simulate {
		return "", nil
	}
	return "", fmt.Errorf("can't find coin that matches. Expected %q, wanted %q", feeCoins, taxes)
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc authtypes.AccountI, fees sdk.Coins) (sdk.Coins, error) {
	if !fees.IsValid() {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	for _, fee := range fees {
		balance := bankKeeper.GetBalance(ctx, acc.GetAddress(), fee.Denom)
		if balance.IsGTE(fee) {
			err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(fee))
			if err != nil {
				return nil, errorsmod.Wrapf(err, "failed to send fee to fee collector: %s", fee)
			}
			return sdk.NewCoins(fee), nil
		}
	}

	return nil, sdkerrors.ErrInsufficientFunds
}

func (fd FeeDecorator) ContainsOnlyBypassMinFeeMsgs(ctx sdk.Context, msgs []sdk.Msg) bool {
	bypassMsgTypes := fd.tax2gasKeeper.GetBypassMinFeeMsgTypes(ctx)

	for _, msg := range msgs {
		if tmstrings.StringInSlice(sdk.MsgTypeURL(msg), bypassMsgTypes) {
			continue
		}
		return false
	}

	return true
}

func (fd FeeDecorator) GetFinalGasPrices(ctx sdk.Context) sdk.DecCoins {
	tax2gasGasPrices := fd.tax2gasKeeper.GetGasPrices(ctx)
	minGasPrices := ctx.MinGasPrices()
	gasPrices := make(sdk.DecCoins, len(tax2gasGasPrices))

	for i, gasPrice := range tax2gasGasPrices {
		maxGasPrice := sdk.DecCoin{
			Denom: gasPrice.Denom,
			Amount: sdk.MaxDec(
				minGasPrices.AmountOf(gasPrice.Denom),
				gasPrice.Amount,
			),
		}

		gasPrices[i] = maxGasPrice
	}

	return gasPrices
}
