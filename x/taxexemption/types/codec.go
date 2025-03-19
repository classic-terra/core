package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
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
		(*govtypes.Content)(nil),
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
	legacy.RegisterAminoMsg(cdc, &MsgAddTaxExemptionZone{}, "taxexemption/AddTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveTaxExemptionZone{}, "taxexemption/RemoveTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgModifyTaxExemptionZone{}, "taxexemption/ModifyTaxExemptionZone")
	legacy.RegisterAminoMsg(cdc, &MsgAddTaxExemptionAddress{}, "taxexemption/AddTaxExemptionAddress")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveTaxExemptionAddress{}, "taxexemption/RemoveTaxExemptionAddress")
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)

	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
}
