package wasm

import (
	"github.com/classic-terra/core/v3/x/tax/handlers"

	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/exported"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the bank module.
type AppModuleBasic struct {
	*wasm.AppModuleBasic
}

// AppModule implements an application module for the bank module.
type AppModule struct {
	*wasm.AppModule

	keeper         *keeper.Keeper
	accountKeeper  types.AccountKeeper
	treasuryKeeper treasurykeeper.Keeper
	taxKeeper      taxkeeper.Keeper
	bankKeeper     bankkeeper.Keeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	keeper *keeper.Keeper,
	validatorSetSource keeper.ValidatorSetSource,
	accountKeeper types.AccountKeeper,
	treasuryKeeper treasurykeeper.Keeper,
	taxKeeper taxkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	router *baseapp.MsgServiceRouter,
	ss exported.Subspace,
) AppModule {
	bm := wasm.NewAppModule(cdc, keeper, validatorSetSource, accountKeeper, bankKeeper, router, ss)
	return AppModule{
		AppModule:      &bm,
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		treasuryKeeper: treasuryKeeper,
		taxKeeper:      taxKeeper,
		bankKeeper:     bankKeeper,
		legacySubspace: ss,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	origMsgServer := keeper.NewMsgServerImpl(am.keeper)
	types.RegisterMsgServer(cfg.MsgServer(), handlers.NewWasmMsgServer(*am.keeper, am.treasuryKeeper, am.taxKeeper, am.bankKeeper, origMsgServer))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.Querier(am.keeper))

	m := keeper.NewMigrator(*am.keeper, am.legacySubspace)
	err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2)
	if err != nil {
		panic(err)
	}
	err = cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3)
	if err != nil {
		panic(err)
	}
	err = cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4)
	if err != nil {
		panic(err)
	}
}
