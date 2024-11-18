package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

// Wrapper type for backward compatibility
type LegacyMsgGrantAllowance struct {
	feegrant.MsgGrantAllowance
}

// Wrapper type for backward compatibility
type LegacyMsgRevokeAllowance struct {
	feegrant.MsgRevokeAllowance
}

// Wrapper type for backward compatibility
type LegacyBasicAllowance struct {
	feegrant.BasicAllowance
}

// Wrapper type for backward compatibility
type LegacyPeriodicAllowance struct {
	feegrant.PeriodicAllowance
}

// Wrapper type for backward compatibility
type LegacyAllowedMsgAllowance struct {
	feegrant.AllowedMsgAllowance
}

func (l LegacyMsgGrantAllowance) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgGrantAllowance)
}

func (l *LegacyMsgGrantAllowance) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgGrantAllowance)
}

func (l LegacyMsgRevokeAllowance) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgRevokeAllowance)
}

func (l *LegacyMsgRevokeAllowance) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgRevokeAllowance)
}

func (l LegacyBasicAllowance) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.BasicAllowance)
}

func (l *LegacyBasicAllowance) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.BasicAllowance)
}

func (l LegacyPeriodicAllowance) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.PeriodicAllowance)
}

func (l *LegacyPeriodicAllowance) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.PeriodicAllowance)
}

func (l LegacyAllowedMsgAllowance) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.AllowedMsgAllowance)
}

func (l *LegacyAllowedMsgAllowance) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.AllowedMsgAllowance)
}

// RegisterLegacyAminoCodec registers the necessary x/authz interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	feegrant.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgGrantAllowance{}, "feegrant/MsgGrantAllowance")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgRevokeAllowance{}, "feegrant/MsgRevokeAllowance")

	cdc.RegisterConcrete(&LegacyBasicAllowance{}, "feegrant/BasicAllowance", nil)
	cdc.RegisterConcrete(&LegacyPeriodicAllowance{}, "feegrant/PeriodicAllowance", nil)
	cdc.RegisterConcrete(&LegacyAllowedMsgAllowance{}, "feegrant/AllowedMsgAllowance", nil)
}
