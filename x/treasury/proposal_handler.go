package treasury

import (
	"github.com/classic-terra/core/x/treasury/keeper"
	"github.com/classic-terra/core/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func NewProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SetWhitelistAddressProposal:
			return handleSetWhitelistAddressProposal(ctx, k, c)
		case *types.RemoveWhitelistAddressProposal:
			return handleRemoveWhitelistAddressProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized treasury proposal content type: %T", c)
		}
	}
}

func handleSetWhitelistAddressProposal(ctx sdk.Context, k keeper.Keeper, p *types.SetWhitelistAddressProposal) error {
	return keeper.HandleSetWhitelistAddressProposal(ctx, k, p)
}

func handleRemoveWhitelistAddressProposal(ctx sdk.Context, k keeper.Keeper, p *types.RemoveWhitelistAddressProposal) error {
	return keeper.HandleRemoveWhitelistAddressProposal(ctx, k, p)
}
