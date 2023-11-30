package keeper

import (
	"github.com/classic-terra/core/v2/x/taxexemption/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func HandleAddTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.AddTaxExemptionZoneProposal) error {
	err := k.AddTaxExemptionZone(ctx, types.Zone{Name: p.Zone, Outgoing: p.Outgoing, Incoming: p.Incoming, CrossZone: p.CrossZone})
	if err != nil {
		return err
	}

	for _, address := range p.Addresses {
		err := k.AddTaxExemptionAddress(ctx, p.Zone, address)
		if err != nil {
			return err
		}
	}

	return nil
}

func HandleRemoveTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.RemoveTaxExemptionZoneProposal) error {
	return k.RemoveTaxExemptionZone(ctx, p.Zone)
}

func HandleModifyTaxExemptionZoneProposal(ctx sdk.Context, k Keeper, p *types.ModifyTaxExemptionZoneProposal) error {
	err := k.ModifyTaxExemptionZone(ctx, types.Zone{Name: p.Zone, Outgoing: p.Outgoing, Incoming: p.Incoming, CrossZone: p.CrossZone})

	return err
}

func HandleAddTaxExemptionAddressProposal(ctx sdk.Context, k Keeper, p *types.AddTaxExemptionAddressProposal) error {
	for _, address := range p.Addresses {
		err := k.AddTaxExemptionAddress(ctx, p.Zone, address)
		if err != nil {
			return err
		}
	}

	return nil
}

func HandleRemoveTaxExemptionAddressProposal(ctx sdk.Context, k Keeper, p *types.RemoveTaxExemptionAddressProposal) error {
	for _, address := range p.Addresses {
		err := k.RemoveTaxExemptionAddress(ctx, p.Zone, address)
		if err != nil {
			return err
		}
	}

	return nil
}
