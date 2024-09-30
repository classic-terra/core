package gov

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/spf13/cobra"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govcodec "github.com/cosmos/cosmos-sdk/x/gov/codec"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customcli "github.com/classic-terra/core/v3/custom/gov/client/cli"
	customtypes "github.com/classic-terra/core/v3/custom/gov/types"
	core "github.com/classic-terra/core/v3/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the gov module.
type AppModuleBasic struct {
	gov.AppModuleBasic
}

// AppModule implements an application module for the gov module.
type AppModule struct {
	gov.AppModule
	accountKeeper govtypes.AccountKeeper
	bankKeeper    govtypes.BankKeeper
	oracleKeeper  markettypes.OracleKeeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace govtypes.ParamSubspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper, accountKeeper govtypes.AccountKeeper, bankKeeper govtypes.BankKeeper, oracleKeeper markettypes.OracleKeeper, legacySubspace govtypes.ParamSubspace) AppModule {
	return AppModule{
		AppModule:      gov.NewAppModule(cdc, keeper, accountKeeper, bankKeeper, legacySubspace),
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		oracleKeeper:   oracleKeeper,
		legacySubspace: legacySubspace,
	}
}

// NewAppModuleBasic creates a new AppModuleBasic object
func NewAppModuleBasic(proposalHandlers []govclient.ProposalHandler) AppModuleBasic {
	return AppModuleBasic{gov.NewAppModuleBasic(proposalHandlers)}
}

// GetTxCmd returns the root tx command for the bank module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return customcli.GetTxCmd()
}

// RegisterLegacyAminoCodec registers the gov module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
	*govcodec.ModuleCdc = *customtypes.ModuleCdc
	v1.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (am AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// customize to set default genesis state deposit denom to uluna
	defaultGenesisState := v1.DefaultGenesisState()
	defaultGenesisState.Params.MinDeposit[0].Denom = core.MicroLunaDenom

	return cdc.MustMarshalJSON(defaultGenesisState)
}
