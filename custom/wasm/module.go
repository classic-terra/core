package wasm

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/simulation"
	"github.com/CosmWasm/wasmd/x/wasm/types"

	customcli "github.com/classic-terra/core/custom/wasm/client/cli"
	customrest "github.com/classic-terra/core/custom/wasm/client/rest"
	customsim "github.com/classic-terra/core/custom/wasm/simulation"
	customtypes "github.com/classic-terra/core/custom/wasm/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// Module init related flags
const (
	flagWasmMemoryCacheSize    = "wasm.memory_cache_size"
	flagWasmQueryGasLimit      = "wasm.query_gas_limit"
	flagWasmSimulationGasLimit = "wasm.simulation_gas_limit"
)

// AppModuleBasic defines the basic application module used by the wasm module.
type AppModuleBasic struct {
	wasm.AppModuleBasic
}

func (b AppModuleBasic) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) { //nolint:staticcheck
	customtypes.RegisterLegacyAminoCodec(amino)
}

// RegisterRESTRoutes registers the REST routes for the wasm module.
func (AppModuleBasic) RegisterRESTRoutes(cliCtx client.Context, rtr *mux.Router) {
	customrest.RegisterRoutes(cliCtx, rtr)
}

// GetTxCmd returns the root tx command for the wasm module.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return customcli.GetTxCmd()
}

type AppModule struct {
	wasm.AppModule
	appModuleBasic     AppModuleBasic
	cdc                codec.Codec
	keeper             *wasm.Keeper
	validatorSetSource keeper.ValidatorSetSource
	accountKeeper      types.AccountKeeper // for simulation
	bankKeeper         simulation.BankKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	keeper *wasm.Keeper,
	validatorSetSource keeper.ValidatorSetSource,
	ak types.AccountKeeper,
	bk simulation.BankKeeper,
) AppModule {
	return AppModule{
		appModuleBasic:     AppModuleBasic{},
		cdc:                cdc,
		keeper:             keeper,
		validatorSetSource: validatorSetSource,
		accountKeeper:      ak,
		bankKeeper:         bk,
	}
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return customsim.WeightedOperations(&simState, am.accountKeeper, am.bankKeeper, am.keeper)
}
