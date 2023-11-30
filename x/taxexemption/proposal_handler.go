package taxexemption

import (
	"github.com/classic-terra/core/v2/x/taxexemption/keeper"
	"github.com/classic-terra/core/v2/x/taxexemption/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func NewProposalHandler(k keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		switch c := content.(type) {
		case *types.AddTaxExemptionZoneProposal:
			return handleAddTaxExemptionZoneProposal(ctx, k, c)
		case *types.RemoveTaxExemptionZoneProposal:
			return handleRemoveTaxExemptionZoneProposal(ctx, k, c)
		case *types.ModifyTaxExemptionZoneProposal:
			return handleModifyTaxExemptionZoneProposal(ctx, k, c)
		case *types.AddTaxExemptionAddressProposal:
			return handleAddTaxExemptionAddressProposal(ctx, k, c)
		case *types.RemoveTaxExemptionAddressProposal:
			return handleRemoveTaxExemptionAddressProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized treasury proposal content type: %T", c)
		}
	}
}

func handleAddTaxExemptionZoneProposal(ctx sdk.Context, k keeper.Keeper, p *types.AddTaxExemptionZoneProposal) error {
	return keeper.HandleAddTaxExemptionZoneProposal(ctx, k, p)
}

func handleRemoveTaxExemptionZoneProposal(ctx sdk.Context, k keeper.Keeper, p *types.RemoveTaxExemptionZoneProposal) error {
	return keeper.HandleRemoveTaxExemptionZoneProposal(ctx, k, p)
}

func handleModifyTaxExemptionZoneProposal(ctx sdk.Context, k keeper.Keeper, p *types.ModifyTaxExemptionZoneProposal) error {
	return keeper.HandleModifyTaxExemptionZoneProposal(ctx, k, p)
}

func handleAddTaxExemptionAddressProposal(ctx sdk.Context, k keeper.Keeper, p *types.AddTaxExemptionAddressProposal) error {
	return keeper.HandleAddTaxExemptionAddressProposal(ctx, k, p)
}

func handleRemoveTaxExemptionAddressProposal(ctx sdk.Context, k keeper.Keeper, p *types.RemoveTaxExemptionAddressProposal) error {
	return keeper.HandleRemoveTaxExemptionAddressProposal(ctx, k, p)
}
