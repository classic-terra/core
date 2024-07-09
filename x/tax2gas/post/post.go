package post

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tax2gasKeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
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
	paidFees := ctx.Value(types.ConsumedGasFee)
	paidFeeCoins, ok := paidFees.(sdk.Coins)
	if !ok {
		return ctx, errorsmod.Wrap(types.ErrParsing, "Error parsing coins")
	}

	taxGas, ok := ctx.Value(types.TaxGas).(uint64)
	if !ok {
		return ctx, errorsmod.Wrap(types.ErrParsing, "Error parsing tax gas")
	}
	// we consume the gas here as we need to calculate the tax for consumed gas
	ctx.GasMeter().ConsumeGas(taxGas, "tax gas")

	gasConsumed := ctx.GasMeter().GasConsumed()
	gasConsumedTax, err := dd.tax2gasKeeper.ComputeTaxOnGasConsumed(ctx, tx, dd.treasuryKeeper, gasConsumed)
	if err != nil {
		return ctx, err
	}

	if simulate {
		return next(ctx, tx, simulate, success)
	}

	var requiredFees sdk.Coins
	if gasConsumedTax != nil && feeCoins != nil {
		// Remove the paid fee coins in ante handler
		requiredFees = gasConsumedTax.Sub(paidFeeCoins...)

		// Check if fee coins contains at least one coin that can cover required fees
		if !feeCoins.IsAnyGTE(requiredFees) {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q, required: %q", feeCoins, requiredFees)
		}

		// Get fee denom identified in ante handler
		feeDenom, ok := ctx.Value(types.FeeDenom).(string)
		if !ok {
			return ctx, errorsmod.Wrap(types.ErrParsing, "Error parsing fee denom")
		}

		found, requiredFee := requiredFees.Find(feeDenom)
		if !found {
			return ctx, errorsmod.Wrapf(types.ErrCoinNotFound, "can'f find %s in %s", feeDenom, requiredFees)
		}

		feePayer := dd.accountKeeper.GetAccount(ctx, feeTx.FeePayer())

		err := dd.bankKeeper.SendCoinsFromAccountToModule(ctx, feePayer.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(requiredFee))
		if err != nil {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	return next(ctx, tx, simulate, success)
}
