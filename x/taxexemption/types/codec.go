package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	govamino "github.com/classic-terra/core/v3/custom/gov/types"
)

// RegisterInterfaces associates protoName with the new message types
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddTaxExemptionZone{},
		&MsgRemoveTaxExemptionZone{},
		&MsgModifyTaxExemptionZone{},
		&MsgAddTaxExemptionAddress{},
		&MsgRemoveTaxExemptionAddress{},
	)

	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&AddTaxExemptionZoneProposal{},
		&RemoveTaxExemptionZoneProposal{},
		&ModifyTaxExemptionZoneProposal{},
		&AddTaxExemptionAddressProposal{},
		&RemoveTaxExemptionAddressProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the concrete types on the Amino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Amino Msg types
	legacy.RegisterAminoMsg(cdc, &MsgAddTaxExemptionZone{}, "taxexemption/AddTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveTaxExemptionZone{}, "taxexemption/RemoveTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgModifyTaxExemptionZone{}, "taxexemption/ModifyTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgAddTaxExemptionAddress{}, "taxexemption/AddTaxExemptionAddress")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveTaxExemptionAddress{}, "taxexemption/RemoveTaxExemptionAddress")

	// Legacy proposal contents (required for gov v1beta1 submit-legacy-proposal)
	cdc.RegisterConcrete(&AddTaxExemptionZoneProposal{}, "taxexemption/AddTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&RemoveTaxExemptionZoneProposal{}, "taxexemption/RemoveTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&ModifyTaxExemptionZoneProposal{}, "taxexemption/ModifyTaxExemptionZoneProposal", nil)
	cdc.RegisterConcrete(&AddTaxExemptionAddressProposal{}, "taxexemption/AddTaxExemptionAddressProposal", nil)
	cdc.RegisterConcrete(&RemoveTaxExemptionAddressProposal{}, "taxexemption/RemoveTaxExemptionAddressProposal", nil)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)

	// Hook proposal content types into custom gov ModuleCdc so submit-legacy-proposal recognizes them
	govamino.RegisterProposalTypeCodec(&AddTaxExemptionZoneProposal{}, "taxexemption/AddTaxExemptionZoneProposal")
	govamino.RegisterProposalTypeCodec(&RemoveTaxExemptionZoneProposal{}, "taxexemption/RemoveTaxExemptionZoneProposal")
	govamino.RegisterProposalTypeCodec(&ModifyTaxExemptionZoneProposal{}, "taxexemption/ModifyTaxExemptionZoneProposal")
	govamino.RegisterProposalTypeCodec(&AddTaxExemptionAddressProposal{}, "taxexemption/AddTaxExemptionAddressProposal")
	govamino.RegisterProposalTypeCodec(&RemoveTaxExemptionAddressProposal{}, "taxexemption/RemoveTaxExemptionAddressProposal")
}
