package wasm

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/spf13/cobra"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/simulation"
	types "github.com/CosmWasm/wasmd/x/wasm/types"

	customcli "github.com/classic-terra/core/v3/custom/wasm/client/cli"
	customtypes "github.com/classic-terra/core/v3/custom/wasm/types/legacy"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the wasm module.
type AppModuleBasic struct {
	wasm.AppModuleBasic
}

// RegisterInterfaces implements InterfaceModule
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// register canonical wasm types
	types.RegisterInterfaces(registry)
	customtypes.RegisterInterfaces(registry)
}

// GetTxCmd returns the root tx command for the wasm module.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return customcli.GetTxCmd()
}

type AppModule struct {
	wasm.AppModule
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	keeper *keeper.Keeper,
	validatorSetSource keeper.ValidatorSetSource,
	ak types.AccountKeeper,
	bk simulation.BankKeeper,
	router *baseapp.MsgServiceRouter,
	ss paramtypes.Subspace,
) AppModule {
	return AppModule{
		AppModule:      wasm.NewAppModule(cdc, keeper, validatorSetSource, ak, bk, router, ss),
		keeper:         keeper,
		legacySubspace: ss.WithKeyTable(ParamKeyTable()),
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	// Register the query service
	originalQueryServer := keeper.Querier(am.keeper)
	types.RegisterQueryServer(
		cfg.QueryServer(),
		NewLegacyQueryServer(
			originalQueryServer,
			am.legacySubspace,
			am.keeper,
		),
	)

	// For wasm module, we need to dereference the keeper pointer
	k := *am.keeper
	m := keeper.NewMigrator(k, am.legacySubspace)
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/wasm from version 1 to 2: %v", err))
	}
}
