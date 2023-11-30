package keeper

import (
	"github.com/classic-terra/core/v2/x/taxexemption/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func HandleAddTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.AddTaxExemptionZoneProposal) error {
	k.AddTaxExemptionZone(ctx, types.Zone{Name: p.Zone, Outgoing: p.Outgoing, Incoming: p.Incoming, CrossZone: p.CrossZone})

	for _, address := range p.Addresses {
		k.AddTaxExemptionAddress(ctx, address, p.Zone)
	}

	return nil
}

func HandleRemoveTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.RemoveTaxExemptionZoneProposal) error {
	k.RemoveTaxExemptionZone(ctx, p.Zone)

	return nil
}

func HandleModifyTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.ModifyTaxExemptionZoneProposal) error {
	k.ModifyTaxExemptionZone(ctx, types.Zone{Name: p.Zone, Outgoing: p.Outgoing, Incoming: p.Incoming, CrossZone: p.CrossZone})

	return nil
}

func HandleAddTaxExemptionAddressProposal(ctx sdk.Context, k Keeper, p *types.AddTaxExemptionAddressProposal) error {
	for _, address := range p.Addresses {
		k.AddTaxExemptionAddress(ctx, address, p.Zone)
	}

	return nil
}

func HandleRemoveTaxExemptionAddressProposal(ctx sdk.Context, k Keeper, p *types.RemoveTaxExemptionAddressProposal) error {
	for _, address := range p.Addresses {
		k.RemoveTaxExemptionAddress(ctx, p.Zone, address)
	}

	return nil
}
