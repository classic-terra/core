package slashing

import (
	"github.com/classic-terra/core/v4/custom/slashing/client/cli"
	customtypes "github.com/classic-terra/core/v4/custom/slashing/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/spf13/cobra"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the slashing module.
type AppModuleBasic struct {
	slashing.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the slashing module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}

// GetQueryCmd returns a manual query command so slashing queries use PrintProto
// output instead of AutoCLI aminojson output, which omits zero-value fields.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}
