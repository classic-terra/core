package post

import (
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	classictaxpost "github.com/classic-terra/core/v2/x/classictax/post"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	treasurykeeper "github.com/classic-terra/core/v2/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	ClassicTaxKeeper classictaxkeeper.Keeper
	TreasuryKeeper   treasurykeeper.Keeper
	BankKeeper       bankkeeper.Keeper
	OracleKeeper     oraclekeeper.Keeper
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewPostHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	return sdk.ChainAnteDecorators(
		classictaxpost.NewClassicTaxPostDecorator(options.ClassicTaxKeeper, options.TreasuryKeeper, options.BankKeeper, options.OracleKeeper),
	), nil
}
