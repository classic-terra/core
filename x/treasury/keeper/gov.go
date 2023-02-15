package keeper

import (
	"github.com/classic-terra/core/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func HandleAddBurnTaxExemptionAddressProposal(ctx sdk.Context, k Keeper, p *types.AddBurnTaxExemptionAddressProposal) error {
	for _, address := range p.ExemptionAddress {
		k.SetExemptAddress(ctx, address)
	}

	return nil
}

func HandleRemoveWhitelistAddressProposal(ctx sdk.Context, k Keeper, p *types.RemoveBurnTaxExemptionAddressProposal) error {
	for _, address := range p.ExemptionAddress {
		err := k.RemoveExemptAddress(ctx, address)
		if err != nil {
			return err
		}
	}

	return nil
}
