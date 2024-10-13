package v2lunc1

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// governance module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &govv1.MsgSubmitProposal{}, "cosmos-sdk/v1/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &govv1.MsgDeposit{}, "cosmos-sdk/v1/MsgDeposit")
	legacy.RegisterAminoMsg(cdc, &govv1.MsgVote{}, "cosmos-sdk/v1/MsgVote")
	legacy.RegisterAminoMsg(cdc, &govv1.MsgVoteWeighted{}, "cosmos-sdk/v1/MsgVoteWeighted")
	legacy.RegisterAminoMsg(cdc, &govv1.MsgExecLegacyContent{}, "cosmos-sdk/v1/MsgExecLegacyContent")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "cosmos-sdk/x/gov/v1/MsgUpdateParams")
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&govv1.MsgSubmitProposal{},
		&govv1.MsgVote{},
		&govv1.MsgVoteWeighted{},
		&govv1.MsgDeposit{},
		&govv1.MsgExecLegacyContent{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/gov module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding as Amino is
	// still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/gov and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)
