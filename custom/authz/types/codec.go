package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// Wrapper type for backward compatibility
type LegacyMsgGrant struct {
	authz.MsgGrant
}

// Wrapper type for backward compatibility
type LegacyMsgRevoke struct {
	authz.MsgRevoke
}

// Wrapper type for backward compatibility
type LegacyMsgExec struct {
	authz.MsgExec
}

// Wrapper type for backward compatibility
type LegacyGenericAuthorization struct {
	authz.GenericAuthorization
}

func (l LegacyMsgGrant) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgGrant)
}

func (l *LegacyMsgGrant) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgGrant)
}

func (l LegacyMsgRevoke) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgRevoke)
}

func (l *LegacyMsgRevoke) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgRevoke)
}

func (l LegacyMsgExec) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgExec)
}

func (l *LegacyMsgExec) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgExec)
}

func (l LegacyGenericAuthorization) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.GenericAuthorization)
}

func (l *LegacyGenericAuthorization) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.GenericAuthorization)
}

// RegisterLegacyAminoCodec registers the necessary x/authz interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	authz.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgGrant{}, "msgauth/MsgGrantAuthorization")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgRevoke{}, "msgauth/MsgRevokeAuthorization")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgExec{}, "msgauth/MsgExecAuthorized")

	cdc.RegisterConcrete(&LegacyGenericAuthorization{}, "msgauth/GenericAuthorization", nil)
}
