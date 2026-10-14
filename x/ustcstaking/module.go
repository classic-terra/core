package ustcstaking

import (
	"context"
	"encoding/json"

	ustcstakingcli "github.com/classic-terra/core/v4/x/ustcstaking/client/cli"
	"github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	"github.com/classic-terra/core/v4/x/ustcstaking/simulation"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

type AppModuleBasic struct{ cdc codec.Codec }

type AppModule struct {
	AppModuleBasic
	keeper            keeper.Keeper
	simulationKeepers simulation.Keepers
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper, dependencies ...simulation.Keepers) AppModule {
	result := AppModule{AppModuleBasic: AppModuleBasic{cdc: cdc}, keeper: k}
	if len(dependencies) > 0 {
		result.simulationKeepers = dependencies[0]
	}
	return result
}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, data json.RawMessage) error {
	var genesis types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesis)
	return ValidateGenesis(&genesis)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	_ = types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

func (AppModuleBasic) GetTxCmd() *cobra.Command    { return ustcstakingcli.GetTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return ustcstakingcli.GetQueryCmd() }

func (AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(ExportGenesis(ctx, am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesis types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesis)
	InitGenesis(ctx, am.keeper, &genesis)
	return nil
}

func (AppModule) QuerierRoute() string { return types.QuerierRoute }

func (am AppModule) RegisterInvariants(registry sdk.InvariantRegistry) {
	keeper.RegisterInvariants(registry, am.keeper)
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
}

func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenesis(simState)
}

func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

func (am AppModule) WeightedOperations(state module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(state, am.keeper, am.simulationKeepers)
}
func (AppModule) ProposalMsgs(module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}
func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}

func (AppModule) EndBlock(context.Context) ([]abci.ValidatorUpdate, error) { return nil, nil }
