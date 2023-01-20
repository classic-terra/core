package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/x/treasury/types"
)

func HandleSetWhitelistAddressProposal(ctx sdk.Context, k Keeper, p *types.SetWhitelistAddressProposal) error {
	for _, address := range p.WhitelistAddress {
		k.SetWhitelistAddress(ctx, address)
	}

	return nil
}

func HandleRemoveWhitelistAddressProposal(ctx sdk.Context, k Keeper, p *types.RemoveWhitelistAddressProposal) error {
	for _, address := range p.WhitelistAddress {
		err := k.RemoveWhitelistAddress(ctx, address)
		if err != nil {
			return err
		}
	}

	return nil
}
