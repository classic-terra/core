package post

import (
	dyncommkeeper "github.com/classic-terra/core/v3/x/dyncomm/keeper"
	dyncommpost "github.com/classic-terra/core/v3/x/dyncomm/post"
	tax2gaskeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	tax2gaspost "github.com/classic-terra/core/v3/x/tax2gas/post"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	AccountKeeper  ante.AccountKeeper
	BankKeeper     tax2gastypes.BankKeeper
	FeegrantKeeper tax2gastypes.FeegrantKeeper
	DyncommKeeper  dyncommkeeper.Keeper
	TreasuryKeeper tax2gastypes.TreasuryKeeper
	DistrKeeper    tax2gastypes.DistrKeeper
	Tax2Gaskeeper  tax2gaskeeper.Keeper
}

// NewPostHandler returns an PostHandler that checks and set target
// commission rate for msg create validator and msg edit validator
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	return sdk.ChainPostDecorators(
		dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
		tax2gaspost.NewTax2GasPostDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TreasuryKeeper, options.DistrKeeper, options.Tax2Gaskeeper),
	), nil
}
