package authz

import (
	customcli "github.com/classic-terra/core/v4/custom/authz/client/cli"
	customtypes "github.com/classic-terra/core/v4/custom/authz/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/spf13/cobra"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the distribution module.
type AppModuleBasic struct {
	authz.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the bank module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}

// GetTxCmd returns the root tx command for the bank module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return customcli.GetTxCmd()
}
