package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type LegacyMsgSend struct {
	*banktypes.MsgSend
}

type LegacyMsgMultiSend struct {
	*banktypes.MsgMultiSend
}

const (
	LegacyMsgSendRoute = "bank"
	LegacyMsgSendType  = "MsgSend"
)

// CustomAminoCodec wraps the original codec to handle both legacy and new formats
type CustomAminoCodec struct {
	*codec.LegacyAmino
}

// NewCustomAminoCodec creates a new CustomAminoCodec
func NewCustomAminoCodec() *CustomAminoCodec {
	return &CustomAminoCodec{codec.NewLegacyAmino()}
}

// MarshalJSON implements custom marshaling
func (msg *LegacyMsgSend) MarshalJSON() ([]byte, error) {
	if msg.Type() == LegacyMsgSendType {
		// Handle legacy format
		return ModuleCdc.MarshalJSON(msg.MsgSend)
	}
	// Handle new format
	return ModuleCdc.MarshalJSON(msg)
}

// UnmarshalJSON implements custom unmarshaling
func (msg *LegacyMsgSend) UnmarshalJSON(data []byte) error {
	var err error

	// Try legacy format first
	legacyMsg := &LegacyMsgSend{}
	err = ModuleCdc.UnmarshalJSON(data, legacyMsg)
	if err == nil && legacyMsg.Type() == LegacyMsgSendType {
		*msg = *legacyMsg
		return nil
	}

	// Try new format
	return ModuleCdc.UnmarshalJSON(data, msg.MsgSend)
}

// RegisterLegacyAminoCodec modification
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Register legacy types with custom names
	legacy.RegisterAminoMsg(cdc, &LegacyMsgSend{}, "bank/MsgSend")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgMultiSend{}, "bank/MsgMultiSend")

	// Register new types
	banktypes.RegisterLegacyAminoCodec(cdc)

	// Register the concrete types
	//cdc.RegisterConcrete(&LegacySendAuthorization{}, "msgauth/SendAuthorization", nil)
}

// Initialize custom codec
var (
	CustomCdc = NewCustomAminoCodec()
	ModuleCdc = codec.NewAminoCodec(CustomCdc.LegacyAmino)
)

func init() {
	RegisterLegacyAminoCodec(CustomCdc.LegacyAmino)
	cryptocodec.RegisterCrypto(CustomCdc.LegacyAmino)
	CustomCdc.Seal()
}
