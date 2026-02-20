package distribution

import (
	customtypes "github.com/classic-terra/core/v4/custom/distribution/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/distribution"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the distribution module.
type AppModuleBasic struct {
	distribution.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the distribution module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}
