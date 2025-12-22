package upgrade

import (
	"cosmossdk.io/x/upgrade"
	customtypes "github.com/classic-terra/core/v4/custom/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the upgrade module.
type AppModuleBasic struct {
	upgrade.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the upgrade module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}
