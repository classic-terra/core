package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// RegisterLegacyAminoCodec registers the account interfaces and concrete types on the
// provided LegacyAmino codec. These types are used for Amino JSON serialization
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&AddTaxExemptionZoneProposal{}, "taxexemption/AddTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&RemoveTaxExemptionZoneProposal{}, "taxexemption/RemoveTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&ModifyTaxExemptionZoneProposal{}, "taxexemption/ModifyTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&AddTaxExemptionAddressProposal{}, "taxexemption/AddTaxExemptionAddressProposal", nil)
	cdc.RegisterConcrete(&RemoveTaxExemptionAddressProposal{}, "taxexemption/RemoveTaxExemptionAddressProposal", nil)
}

// RegisterInterfaces associates protoName with AccountI and VestingAccount
// Interfaces and creates a registry of it's concrete implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&AddTaxExemptionZoneProposal{},
		&RemoveTaxExemptionZoneProposal{},
		&ModifyTaxExemptionZoneProposal{},
		&AddTaxExemptionAddressProposal{},
		&RemoveTaxExemptionAddressProposal{},
	)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/oracle module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding as Amino is
	// still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/staking and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
