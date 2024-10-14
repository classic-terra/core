package post

import (
	authante "github.com/classic-terra/core/v3/custom/auth/ante"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// DyncommDecorator does post runMsg store
// modifications for dyncomm module
type TaxDecorator struct {
	taxKeeper      taxkeeper.Keeper
	bankKeeper     bankkeeper.Keeper
	accountKeeper  accountkeeper.AccountKeeper
	treasuryKeeper treasurykeeper.Keeper
}

func NewTaxDecorator(tk taxkeeper.Keeper, bk bankkeeper.Keeper, ak accountkeeper.AccountKeeper, trk treasurykeeper.Keeper) TaxDecorator {
	return TaxDecorator{
		taxKeeper:      tk,
		bankKeeper:     bk,
		accountKeeper:  ak,
		treasuryKeeper: trk,
	}
}

func (dd TaxDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {

	value := ctx.Value(taxtypes.ContextKeyTaxDue)
	dueTax, ok := value.(sdk.Coins)
	if !ok {
		// no tax is due
		return next(ctx, tx, simulate, success)
	}

	if dueTax.IsZero() {
		// no tax is due
		return next(ctx, tx, simulate, success)
	}

	value = ctx.Value(taxtypes.ContextKeyTaxPayer)
	deductFeesFrom, err := sdk.AccAddressFromBech32(value.(string))
	if err != nil {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "tax payer address not found")
	}

	deductFeesFromAcc := dd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	err = authante.DeductFees(dd.bankKeeper, ctx, deductFeesFromAcc, dueTax)
	if err != nil {
		return ctx, err
	}
	// pay the tax
	err = dd.taxKeeper.ProcessTaxSplits(ctx, dueTax)
	if err != nil {
		return ctx, err
	}
	// Record tax proceeds
	dd.treasuryKeeper.RecordEpochTaxProceeds(ctx, dueTax)

	return next(ctx, tx, simulate, success)
}
