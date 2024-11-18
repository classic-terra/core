package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/evidence/types"
)

// Wrapper type for backward compatibility
type LegacyMsgSubmitEvidence struct {
	types.MsgSubmitEvidence
}

// Wrapper type for backward compatibility
type LegacyEquivocation struct {
	types.Equivocation
}

func (l LegacyMsgSubmitEvidence) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgSubmitEvidence)
}

func (l *LegacyMsgSubmitEvidence) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgSubmitEvidence)
}

func (l LegacyEquivocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Equivocation)
}

func (l *LegacyEquivocation) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.Equivocation)
}

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// evidence module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgSubmitEvidence{}, "evidence/MsgSubmitEvidence")
	cdc.RegisterConcrete(&LegacyEquivocation{}, "evidence/Equivocation", nil)
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
