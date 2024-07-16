package post

import (
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
	treasuryKeeper types.TreasuryKeeper
	tax2gasKeeper  tax2gasKeeper.Keeper
}

func NewTax2GasPostDecorator(accountKeeper ante.AccountKeeper, bankKeeper types.BankKeeper, treasuryKeeper types.TreasuryKeeper, tax2gasKeeper tax2gasKeeper.Keeper) Tax2gasPostDecorator {
	return Tax2gasPostDecorator{
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		treasuryKeeper: treasuryKeeper,
		tax2gasKeeper:  tax2gasKeeper,
	}
}

func (dd Tax2gasPostDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, success bool, next sdk.PostHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	feeCoins := feeTx.GetFee()
	anteConsumedGas, ok := ctx.Value(types.AnteConsumedGas).(uint64)
	if !ok {
		return ctx, errorsmod.Wrap(types.ErrParsing, "Error parsing ante consumed gas")
	}

	if !feeCoins.IsZero() {
		// Get paid denom identified in ante handler
		paidDenom, ok := ctx.Value(types.PaidDenom).(string)
		if !ok {
			return ctx, errorsmod.Wrap(types.ErrParsing, "Error parsing fee denom")
		}

		gasPrices := dd.tax2gasKeeper.GetGasPrices(ctx)
		paidDenomGasPrice, found := tax2gasutils.GetGasPriceByDenom(ctx, gasPrices, paidDenom)
		if !found {
			return ctx, types.ErrDenomNotFound
		}
		paidAmount := paidDenomGasPrice.Mul(sdk.NewDec(int64(anteConsumedGas)))
		// Deduct feeCoins with paid amount
		feeCoins = feeCoins.Sub(sdk.NewCoin(paidDenom, paidAmount.Ceil().RoundInt()))

		taxGas, ok := ctx.Value(types.TaxGas).(uint64)
		if !ok {
			taxGas = 0
		}
		// we consume the gas here as we need to calculate the tax for consumed gas
		ctx.GasMeter().ConsumeGas(taxGas, "tax gas")
		gasConsumed := ctx.GasMeter().GasConsumed()

		// Deduct the gas consumed amount spent on ante handler
		gasRemaining := gasConsumed - anteConsumedGas

		if simulate {
			return next(ctx, tx, simulate, success)
		}

		for _, feeCoin := range feeCoins {
			feePayer := dd.accountKeeper.GetAccount(ctx, feeTx.FeePayer())
			gasPrice, found := tax2gasutils.GetGasPriceByDenom(ctx, gasPrices, feeCoin.Denom)
			if !found {
				continue
			}
			feeRequired := sdk.NewCoin(feeCoin.Denom, gasPrice.MulInt64(int64(gasRemaining)).Ceil().RoundInt())
			
			if feeCoin.IsGTE(feeRequired) {
				err := dd.bankKeeper.SendCoinsFromAccountToModule(ctx, feePayer.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(feeRequired))
				if err != nil {
					return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}
				gasRemaining = 0
				break
			} else {
				err := dd.bankKeeper.SendCoinsFromAccountToModule(ctx, feePayer.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(feeCoin))
				if err != nil {
					return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}
				feeRemaining := sdk.NewDecCoinFromCoin(feeRequired.Sub(feeCoin))
				gasRemaining = uint64(feeRemaining.Amount.Quo(gasPrice).Ceil().RoundInt64())
			}
		}
		if gasRemaining > 0 {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "fees are not enough to pay for gas")
		}
	}
	return next(ctx, tx, simulate, success)
}
