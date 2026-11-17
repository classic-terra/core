package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&AddBurnTaxExemptionAddressProposal{}, "treasury/AddBurnTaxExemptionAddressProposal", nil)
	cdc.RegisterConcrete(&RemoveBurnTaxExemptionAddressProposal{}, "treasury/RemoveBurnTaxExemptionAddressProposal", nil)
}

// RegisterInterfaces keeps the deprecated burn tax exemption proposal types
// registered. Do not remove: historical proposals stored in x/gov (e.g.
// mainnet #12175) contain them and must remain decodable.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&AddBurnTaxExemptionAddressProposal{},
		&RemoveBurnTaxExemptionAddressProposal{},
	)
}
