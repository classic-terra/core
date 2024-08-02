package post

import (
	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tax2gasKeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
	tax2gasutils "github.com/classic-terra/core/v3/x/tax2gas/utils"
)

type Tax2gasPostDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper types.FeegrantKeeper
	treasuryKeeper types.TreasuryKeeper
	distrKeeper    types.DistrKeeper
	tax2gasKeeper  tax2gasKeeper.Keeper
}

func NewTax2GasPostDecorator(accountKeeper ante.AccountKeeper, bankKeeper types.BankKeeper, feegrantKeeper types.FeegrantKeeper, treasuryKeeper types.TreasuryKeeper, distrKeeper types.DistrKeeper, tax2gasKeeper tax2gasKeeper.Keeper) Tax2gasPostDecorator {
	return Tax2gasPostDecorator{
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		feegrantKeeper: feegrantKeeper,
		treasuryKeeper: treasuryKeeper,
		distrKeeper:    distrKeeper,
		tax2gasKeeper:  tax2gasKeeper,
	}
}

func (tgd Tax2gasPostDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, success bool, next sdk.PostHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}
	msgs := feeTx.GetMsgs()
	if tax2gasutils.IsOracleTx(msgs) || !tgd.tax2gasKeeper.IsEnabled(ctx) {
		return next(ctx, tx, simulate, success)
	}

	feeCoins := feeTx.GetFee()
	anteConsumedGas, ok := ctx.Value(types.AnteConsumedGas).(uint64)
	if !simulate && !ok {
		// If cannot found the anteConsumedGas, that's mean the tx is bypass
		// Skip this tx as it's bypass
		return next(ctx, tx, simulate, success)
	}

	// Get paid denom identified in ante handler
	paidDenom, ok := ctx.Value(types.PaidDenom).(string)
	if !simulate && !ok {
		// If cannot found the paidDenom, that's mean this is the init genesis tx
		// Skip this tx as it's init genesis tx
		return next(ctx, tx, simulate, success)
	}

	gasPrices, ok := ctx.Value(types.FinalGasPrices).(sdk.DecCoins)
	if !ok {
		gasPrices = tgd.tax2gasKeeper.GetGasPrices(ctx)
	}

	found, paidDenomGasPrice := tax2gasutils.GetGasPriceByDenom(gasPrices, paidDenom)
	if !simulate && !found {
		return ctx, types.ErrDenomNotFound
	}
	paidAmount := paidDenomGasPrice.Mul(sdk.NewDec(int64(anteConsumedGas)))

	if !simulate {
		// Deduct feeCoins with paid amount
		feeCoins = feeCoins.Sub(sdk.NewCoin(paidDenom, paidAmount.Ceil().RoundInt()))
	}

	taxGas := ctx.TaxGasMeter().GasConsumed()

	// we consume the gas here as we need to calculate the tax for consumed gas
	// if the gas overflow, then that means the tx can't be estimates as normal way
	// we need to add the --fee flag manually
	totalGasConsumed := ctx.GasMeter().GasConsumed()

	if taxGas.IsUint64() {
		taxGasUint64 := taxGas.Uint64()
		// Check if gas not overflow
		if totalGasConsumed+taxGasUint64 >= totalGasConsumed && totalGasConsumed+taxGasUint64 >= taxGasUint64 {
			if simulate {
				ctx.GasMeter().ConsumeGas(taxGasUint64, "consume tax gas")
			}
		}
	}

	// Deduct the gas consumed amount spent on ante handler
	totalGasRemaining := sdkmath.NewInt(int64(totalGasConsumed - anteConsumedGas)).Add(taxGas)

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		allowance, err := tgd.feegrantKeeper.GetAllowance(ctx, feeGranter, feePayer)
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "fee-grant not found with granter %s and grantee %s", feeGranter, feePayer)
		}

		gasRemainingFees, err := tax2gasutils.ComputeFeesOnGasConsumed(tx, gasPrices, totalGasRemaining)
		if err != nil {
			return ctx, err
		}

		// For this tx, we only accept to pay by one denom
		for _, feeRequired := range gasRemainingFees {
			_, err := allowance.Accept(ctx, sdk.NewCoins(feeRequired), feeTx.GetMsgs())
			if err == nil {
				err = tgd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, sdk.NewCoins(feeRequired), feeTx.GetMsgs())
				if err != nil {
					return ctx, errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
				}
				feeGranter := tgd.accountKeeper.GetAccount(ctx, feeGranter)
				err = tgd.bankKeeper.SendCoinsFromAccountToModule(ctx, feeGranter.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(feeRequired))
				if err != nil {
					return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}

				// Calculate tax fee and BurnTaxSplit
				_, gasPrice := tax2gasutils.GetGasPriceByDenom(gasPrices, feeRequired.Denom)
				taxFee := gasPrice.MulInt(taxGas).Ceil().RoundInt()

				err := tgd.BurnTaxSplit(ctx, sdk.NewCoins(sdk.NewCoin(feeRequired.Denom, taxFee)))
				if err != nil {
					return ctx, err
				}
				return next(ctx, tx, simulate, success)
			}
		}
		return ctx, errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
	}

	// First, we will deduct the fees covered taxGas and handle BurnTaxSplit
	taxes, payableFees, gasRemaining := tax2gasutils.CalculateTaxesAndPayableFee(gasPrices, feeCoins, taxGas, totalGasRemaining)
	if !simulate && !ctx.IsCheckTx() && gasRemaining.IsPositive() {
		gasRemainingFees, err := tax2gasutils.ComputeFeesOnGasConsumed(tx, gasPrices, gasRemaining)
		if err != nil {
			return ctx, err
		}

		return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "fees are not enough to pay for gas, need to cover %s gas more, which equal to %q ", gasRemaining.String(), gasRemainingFees)
	}
	feePayerAccount := tgd.accountKeeper.GetAccount(ctx, feePayer)

	if !simulate && taxes.IsZero() {
		payableFees = feeCoins
	}
	err := tgd.bankKeeper.SendCoinsFromAccountToModule(ctx, feePayerAccount.GetAddress(), authtypes.FeeCollectorName, payableFees)
	if err != nil {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	err = tgd.BurnTaxSplit(ctx, taxes)
	if err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate, success)
}
