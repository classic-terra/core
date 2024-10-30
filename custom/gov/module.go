package gov

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/simulation"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	customcli "github.com/classic-terra/core/v3/custom/gov/client/cli"
	"github.com/classic-terra/core/v3/custom/gov/keeper"
	"github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	core "github.com/classic-terra/core/v3/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
)

const ConsensusVersion = 5

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the gov module.
type AppModuleBasic struct {
	gov.AppModuleBasic
	legacyProposalHandlers []govclient.ProposalHandler
}

// Name returns the gov module's name.
func (AppModuleBasic) Name() string {
	return govtypes.ModuleName
}

// NewAppModuleBasic creates a new AppModuleBasic object
func NewAppModuleBasic(proposalHandlers []govclient.ProposalHandler) AppModuleBasic {
	return AppModuleBasic{
		AppModuleBasic:         gov.NewAppModuleBasic(proposalHandlers),
		legacyProposalHandlers: proposalHandlers,
	}
}

// GetTxCmd returns the root tx command for the bank module.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	legacyProposalCLIHandlers := getProposalCLIHandlers(a.legacyProposalHandlers)
	return cli.NewTxCmd(legacyProposalCLIHandlers)
}

func getProposalCLIHandlers(handlers []govclient.ProposalHandler) []*cobra.Command {
	proposalCLIHandlers := make([]*cobra.Command, 0, len(handlers))
	for _, proposalHandler := range handlers {
		proposalCLIHandlers = append(proposalCLIHandlers, proposalHandler.CLIHandler())
	}
	return proposalCLIHandlers
}

// GetQueryCmd returns no root query command for the oracle module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return customcli.GetQueryCmd()
}

// RegisterLegacyAminoCodec registers the gov module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	v1beta1.RegisterLegacyAminoCodec(cdc)
	v2lunc1.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces implements InterfaceModule.RegisterInterfaces
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	v1beta1.RegisterInterfaces(registry)
	v2lunc1.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (a AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// customize to set default genesis state deposit denom to uluna
	defaultGenesisState := v2lunc1.DefaultGenesisState()
	defaultGenesisState.Params.MinDeposit[0].Denom = core.MicroLunaDenom

	return cdc.MustMarshalJSON(defaultGenesisState)
}

// ValidateGenesis performs genesis state validation for the gov module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data v2lunc1.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", govtypes.ModuleName, err)
	}

	return v2lunc1.ValidateGenesis(&data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the gov module.
func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := v1.RegisterQueryHandlerClient(context.Background(), mux, v1.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
	if err := v1beta1.RegisterQueryHandlerClient(context.Background(), mux, v1beta1.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// AppModule implements an application module for the gov module.
type AppModule struct {
	gov.AppModule
	accountKeeper govtypes.AccountKeeper
	bankKeeper    govtypes.BankKeeper
	oracleKeeper  markettypes.OracleKeeper
	keeper        keeper.Keeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace govtypes.ParamSubspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, accountKeeper govtypes.AccountKeeper, bankKeeper govtypes.BankKeeper, oracleKeeper markettypes.OracleKeeper, legacySubspace govtypes.ParamSubspace) AppModule {
	return AppModule{
		AppModule:      gov.NewAppModule(cdc, keeper.Keeper, accountKeeper, bankKeeper, legacySubspace),
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		oracleKeeper:   oracleKeeper,
		keeper:         keeper,
		legacySubspace: legacySubspace,
	}
}

// Name returns the gov module's name.
func (AppModule) Name() string {
	return govtypes.ModuleName
}

// RegisterInvariants registers module invariants
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	am.AppModule.RegisterInvariants(ir)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	msgServer := keeper.NewMsgServerImpl(&am.keeper)
	v1beta1.RegisterMsgServer(cfg.MsgServer(), keeper.NewLegacyMsgServerImpl(am.accountKeeper.GetModuleAddress(govtypes.ModuleName).String(), msgServer))
	v2lunc1.RegisterMsgServer(cfg.MsgServer(), msgServer)

	queryServer := keeper.NewQueryServerImpl(am.keeper)
	v2lunc1.RegisterQueryServer(cfg.QueryServer(), queryServer)

	legacyQueryServer := govkeeper.NewLegacyQueryServer(am.keeper.Keeper)
	v1beta1.RegisterQueryServer(cfg.QueryServer(), legacyQueryServer)
	v1.RegisterQueryServer(cfg.QueryServer(), am.keeper.Keeper)

	m := keeper.NewMigrator(&am.keeper, am.legacySubspace)
	err := cfg.RegisterMigration(govtypes.ModuleName, 1, m.Migrate1to2)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate x/gov from version 1 to 2: %v", err))
	}
	err = cfg.RegisterMigration(govtypes.ModuleName, 2, m.Migrate2to3)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate x/gov from version 2 to 3: %v", err))
	}
	err = cfg.RegisterMigration(govtypes.ModuleName, 3, m.Migrate3to4)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate x/gov from version 3 to 4: %v", err))
	}
	err = cfg.RegisterMigration(govtypes.ModuleName, 4, m.Migrate4to5)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate x/gov from version 4 to 5: %v", err))
	}
}

// InitGenesis performs genesis initialization for the gov module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState v2lunc1.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.accountKeeper, am.bankKeeper, &am.keeper, &genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the gov
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return am.AppModule.ExportGenesis(ctx, cdc)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// EndBlock returns the end blocker for the gov module. It returns no validator
// updates.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return am.AppModule.EndBlock(ctx, req)
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the gov module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalContents returns all the gov content functions used to
// simulate governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return simulation.ProposalContents()
}

// ProposalMsgs returns all the gov msgs used to simulate governance proposals.
func (AppModule) ProposalMsgs(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for gov module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	am.AppModule.RegisterStoreDecoder(sdr)
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(
		simState.AppParams, simState.Cdc,
		am.accountKeeper, am.bankKeeper, am.keeper.Keeper,
		simState.ProposalMsgs, simState.LegacyProposalContents,
	)
}
