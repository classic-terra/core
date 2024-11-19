package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// Wrapper type for backward compatibility
type LegacyStdTx struct {
	legacytx.StdTx
}

// Wrapper type for backward compatibility
type LegacyBaseAccount struct {
	types.BaseAccount
}

// Wrapper type for backward compatibility
type LegacyModuleAccount struct {
	types.ModuleAccount
}

func (l LegacyStdTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.StdTx)
}

func (l *LegacyStdTx) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.StdTx)
}

func (l LegacyBaseAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.BaseAccount)
}

func (l *LegacyBaseAccount) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.BaseAccount)
}

func (l LegacyModuleAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ModuleAccount)
}

func (l *LegacyModuleAccount) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.ModuleAccount)
}

// RegisterLegacyAminoCodec registers the account interfaces and concrete types on the
// provided LegacyAmino codec. These types are used for Amino JSON serialization
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	cdc.RegisterConcrete(&LegacyBaseAccount{}, "core/Account", nil)
	cdc.RegisterConcrete(&LegacyModuleAccount{}, "core/ModuleAccount", nil)
	cdc.RegisterConcrete(LegacyStdTx{}, "core/StdTx", nil)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/oracle module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding as Amino is
	// still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/staking and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
}
