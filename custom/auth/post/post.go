package post

import (
	dyncommkeeper "github.com/classic-terra/core/v3/x/dyncomm/keeper"
	dyncommpost "github.com/classic-terra/core/v3/x/dyncomm/post"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	DyncommKeeper dyncommkeeper.Keeper
}

// NewPostHandler returns an PostHandler that checks and set target
// commission rate for msg create validator and msg edit validator
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	return sdk.ChainPostDecorators(
		dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
	), nil
}
