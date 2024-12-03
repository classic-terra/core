package post

import (
	dyncommkeeper "github.com/classic-terra/core/v3/x/dyncomm/keeper"
	dyncommpost "github.com/classic-terra/core/v3/x/dyncomm/post"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	taxpost "github.com/classic-terra/core/v3/x/tax/post"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	DyncommKeeper  dyncommkeeper.Keeper
	TaxKeeper      taxkeeper.Keeper
	BankKeeper     bankkeeper.Keeper
	AccountKeeper  accountkeeper.AccountKeeper
	TreasuryKeeper treasurykeeper.Keeper
}

// NewPostHandler returns an PostHandler that checks and set target
// commission rate for msg create validator and msg edit validator
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	return sdk.ChainPostDecorators(
		dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
		taxpost.NewTaxDecorator(options.TaxKeeper, options.BankKeeper, options.AccountKeeper, options.TreasuryKeeper),
	), nil
}
