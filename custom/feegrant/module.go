package feegrant

import (
	feegrant "cosmossdk.io/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	customtypes "github.com/classic-terra/core/v3/custom/feegrant/types"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the distribution module.
type AppModuleBasic struct {
	feegrant.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the bank module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}
