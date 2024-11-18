package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Wrapper type for backward compatibility
type LegacyMsgCreateValidator struct {
	types.MsgCreateValidator
}

// Wrapper type for backward compatibility
type LegacyMsgEditValidator struct {
	types.MsgEditValidator
}

// Wrapper type for backward compatibility
type LegacyMsgDelegate struct {
	types.MsgDelegate
}

// Wrapper type for backward compatibility
type LegacyMsgUndelegate struct {
	types.MsgUndelegate
}

// Wrapper type for backward compatibility
type LegacyMsgBeginRedelegate struct {
	types.MsgBeginRedelegate
}

// Wrapper type for backward compatibility
type LegacyStakeAuthorization struct {
	types.StakeAuthorization
}

func (l LegacyMsgCreateValidator) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgCreateValidator)
}

func (l *LegacyMsgCreateValidator) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgCreateValidator)
}

func (l LegacyMsgEditValidator) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgEditValidator)
}

func (l *LegacyMsgEditValidator) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgEditValidator)
}

func (l LegacyMsgDelegate) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgDelegate)
}

func (l *LegacyMsgDelegate) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgDelegate)
}

func (l LegacyMsgUndelegate) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgUndelegate)
}

func (l *LegacyMsgUndelegate) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgUndelegate)
}

func (l LegacyMsgBeginRedelegate) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgBeginRedelegate)
}

func (l *LegacyMsgBeginRedelegate) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgBeginRedelegate)
}

func (l LegacyStakeAuthorization) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.StakeAuthorization)
}

func (l *LegacyStakeAuthorization) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.StakeAuthorization)
}

// RegisterLegacyAminoCodec registers the necessary x/staking interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	legacy.RegisterAminoMsg(cdc, &LegacyMsgCreateValidator{}, "staking/MsgCreateValidator")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgEditValidator{}, "staking/MsgEditValidator")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgDelegate{}, "staking/MsgDelegate")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgUndelegate{}, "staking/MsgUndelegate")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgBeginRedelegate{}, "staking/MsgBeginRedelegate")

	cdc.RegisterConcrete(&LegacyStakeAuthorization{}, "msgauth/StakeAuthorization", nil)
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
