package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
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

func (m LegacyMsgSend) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.MsgSend)
}

func (m *LegacyMsgSend) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.MsgSend)
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
	types.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgSend{}, "bank/MsgSend")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgMultiSend{}, "bank/MsgMultiSend")
	cdc.RegisterConcrete(&LegacySendAuthorization{}, "msgauth/SendAuthorization", nil)
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
