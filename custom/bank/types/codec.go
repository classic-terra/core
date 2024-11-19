package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var (
	_ sdk.Msg = &LegacyMsgSend{}
	_ sdk.Msg = &LegacyMsgMultiSend{}
)

// Wrapper type for backward compatibility
type LegacyMsgSend struct {
	types.MsgSend
}

// Wrapper type for backward compatibility
type LegacyMsgMultiSend struct {
	types.MsgMultiSend
}

type LegacySendAuthorization struct {
	types.SendAuthorization
}

func (m LegacyMsgSend) ProtoMessage()      {}
func (m LegacyMsgMultiSend) ProtoMessage() {}

func (m LegacyMsgSend) MarshalJSON() ([]byte, error) {
	fmt.Println("MarshalJSON")
	return json.Marshal(m.MsgSend)
}

func (m LegacyMsgSend) UnmarshalJSON(data []byte) error {
	fmt.Println("UnmarshalJSON")
	return json.Unmarshal(data, &m.MsgSend)
}

// Implement sdk.Msg interface
func (msg LegacyMsgSend) Route() string {
	fmt.Println("Route")
	return msg.MsgSend.Route()
}
func (msg LegacyMsgSend) Type() string {
	fmt.Println("Type")
	return msg.MsgSend.Type()
}
func (msg LegacyMsgSend) ValidateBasic() error {
	fmt.Println("ValidateBasic")
	return msg.MsgSend.ValidateBasic()
}
func (msg LegacyMsgSend) GetSignBytes() []byte {
	fmt.Println("GetSignBytes")
	return msg.MsgSend.GetSignBytes()
}
func (msg LegacyMsgSend) GetSigners() []sdk.AccAddress {
	fmt.Println("GetSigners")
	return msg.MsgSend.GetSigners()
}

func (m LegacyMsgMultiSend) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.MsgMultiSend)
}

func (m *LegacyMsgMultiSend) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.MsgMultiSend)
}

func (m LegacySendAuthorization) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.SendAuthorization)
}

func (m *LegacySendAuthorization) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.SendAuthorization)
}

// RegisterLegacyAminoCodec registers the necessary x/bank interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// register the normal bank message types
	legacy.RegisterAminoMsg(cdc, &LegacyMsgSend{}, "bank/MsgSend")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgMultiSend{}, "bank/MsgMultiSend")
	cdc.RegisterConcrete(&LegacySendAuthorization{}, "msgauth/SendAuthorization", nil)
	types.RegisterLegacyAminoCodec(cdc)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// register the normal bank message types
	proto.RegisterType((*LegacyMsgSend)(nil), "cosmos.bank.v1beta1.LegacyMsgSend")
	proto.RegisterType((*LegacyMsgMultiSend)(nil), "cosmos.bank.v1beta1.LegacyMsgMultiSend")

	types.RegisterInterfaces(registry)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&LegacyMsgSend{},
		&LegacyMsgMultiSend{},
	)

}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
