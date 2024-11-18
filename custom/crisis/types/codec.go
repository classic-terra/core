package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/crisis/types"
)

// Wrapper type for backward compatibility
type LegacyMsgVerifyInvariant struct {
	types.MsgVerifyInvariant
}

func (l LegacyMsgVerifyInvariant) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgVerifyInvariant)
}

func (l *LegacyMsgVerifyInvariant) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgVerifyInvariant)
}

// RegisterLegacyAminoCodec registers the necessary x/crisis interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
	legacy.RegisterAminoMsg(cdc, &LegacyMsgVerifyInvariant{}, "crisis/MsgVerifyInvariant")
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
