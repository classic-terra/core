package post

import (
	dyncommkeeper "github.com/classic-terra/core/v3/x/dyncomm/keeper"
	dyncommpost "github.com/classic-terra/core/v3/x/dyncomm/post"
	tax2gasKeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	tax2gasPost "github.com/classic-terra/core/v3/x/tax2gas/post"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
	tax2gasTypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	AccountKeeper  ante.AccountKeeper
	BankKeeper     types.BankKeeper
	DyncommKeeper  dyncommkeeper.Keeper
	TreasuryKeeper tax2gasTypes.TreasuryKeeper
	Tax2Gaskeeper  tax2gasKeeper.Keeper
}

// NewPostHandler returns an PostHandler that checks and set target
// commission rate for msg create validator and msg edit validator
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	return sdk.ChainPostDecorators(
		dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
		tax2gasPost.NewTax2GasPostDecorator(options.AccountKeeper, options.BankKeeper, options.TreasuryKeeper, options.Tax2Gaskeeper),
	), nil
}
