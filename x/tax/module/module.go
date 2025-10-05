package module

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	core "github.com/classic-terra/core/v3/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/classic-terra/core/v3/x/market/simulation"
	"github.com/classic-terra/core/v3/x/tax/client/cli"
	"github.com/classic-terra/core/v3/x/tax/keeper"
	"github.com/classic-terra/core/v3/x/tax/types"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

type AppModuleBasic struct {
	cdc codec.Codec
}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the tax module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// ---------------------------------------
// Interfaces.
func (b AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

func (b AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterInterfaces registers interfaces and implementations of the tax module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

type AppModule struct {
	AppModuleBasic

	k keeper.Keeper
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.k))
	// queryproto.RegisterQueryServer(cfg.QueryServer(), grpc.Querier{Q: module.NewQuerier(am.k)})
	types.RegisterQueryServer(cfg.QueryServer(), am.k)

	// Register no-op migration from version 1 to 2 (consensus version bump)
	if err := cfg.RegisterMigration(types.ModuleName, 1, func(ctx sdk.Context) error {
		return nil
	}); err != nil {
		panic(fmt.Sprintf("failed to register migration for x/%s from version 1 to 2: %v", types.ModuleName, err))
	}
}

func NewAppModule(cdc codec.Codec, taxKeeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc},
		k:              taxKeeper,
	}
}

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// GenerateGenesisState creates a randomized GenState of the dyncomm module.
// Simulation hooks intentionally omitted for SDK v0.50
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	taxGenesis := types.DefaultGenesisState()
	params := types.DefaultParams()
	params.BurnTaxRate = sdkmath.LegacyNewDecWithPrec(1, 2)
	params.GasPrices = sdk.NewDecCoins(sdk.NewDecCoinFromDec(core.MicroSDRDenom, sdkmath.LegacyZeroDec()))
	taxGenesis.Params = params
	_, _ = json.MarshalIndent(&taxGenesis, "", " ")
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(taxGenesis)
}

// QuerierRoute returns the tax module's querier route name.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// InitGenesis performs genesis initialization for the tax module.
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState

	cdc.MustUnmarshalJSON(gs, &genesisState)
	InitGenesis(ctx, am.k, &genesisState)
	return nil
}

// ExportGenesis returns the exported genesis state as raw bytes for the tax.
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.k.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

// RegisterStoreDecoder registers a decoder for dyncomm module's types
// IsAppModule implements the appmodule.AppModule marker.
func (AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType marker.
func (AppModule) IsOnePerModuleType() {}

func (AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

// BeginBlock performs TODO.
// BeginBlock deprecated in v0.50

// EndBlock performs TODO.
func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	return []abci.ValidatorUpdate{}, nil
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 2 }
