package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

// Wrapper type for backward compatibility
type LegacyMsgUnjail struct {
	types.MsgUnjail
}

func (l LegacyMsgUnjail) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgUnjail)
}

func (l *LegacyMsgUnjail) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgUnjail)
}

// RegisterLegacyAminoCodec registers concrete types on LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgUnjail{}, "slashing/MsgUnjail")
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
