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

	// Compute taxes based on consumed gas
	gasPrices := fd.tax2gasKeeper.GetGasPrices(ctx)
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

	if !simulate && !taxes.IsZero() {
		// Fee has to at least be enough to cover taxes
		priority, err = fd.checkTxFee(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}

	// Try to deduct the gasConsumed fees
	paidDenom, err := fd.tryDeductFee(ctx, feeTx, gasConsumedFees, simulate)
	if err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority).WithValue(types.TaxGas, taxGas)
	if !taxGas.IsZero() {
		newCtx.TaxGasMeter().ConsumeGas(taxGas, "ante handler taxGas")
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
	return "", fmt.Errorf("can't find coin that matches. Expected %s, wanted %s", feeCoins, taxes)
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

// checkTxFee implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
// Transaction with only oracle messages will skip gas fee check and will have the most priority.
// It also checks enough fee for treasury tax
func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx) (int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()
	isOracleTx := tax2gasutils.IsOracleTx(msgs)
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

		// Check required fees
		if !requiredGasFees.IsZero() && !feeCoins.IsAnyGTE(requiredGasFees) {
			return 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q", feeCoins, requiredGasFees)
		}
	}

	priority := int64(math.MaxInt64)

	if !isOracleTx {
		priority = tax2gasutils.GetTxPriority(feeCoins, int64(gas))
	}

	return priority, nil
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
