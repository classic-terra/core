package bank

import (
	"fmt"

	"github.com/classic-terra/core/v3/x/tax/handlers"

	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

// ConsensusVersion defines the current x/bank module consensus version.
const ConsensusVersion = 4

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the bank module.
type AppModuleBasic struct {
	*bank.AppModuleBasic
}

// AppModule implements an application module for the bank module.
type AppModule struct {
	*bank.AppModule

	keeper         keeper.Keeper
	accountKeeper  types.AccountKeeper
	treasuryKeeper treasurykeeper.Keeper
	taxKeeper      taxkeeper.Keeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, accountKeeper types.AccountKeeper, treasuryKeeper treasurykeeper.Keeper, ss exported.Subspace, taxKeeper taxkeeper.Keeper) AppModule {
	bm := bank.NewAppModule(cdc, keeper, accountKeeper, ss)
	return AppModule{
		AppModule:      &bm,
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		treasuryKeeper: treasuryKeeper,
		legacySubspace: ss,
		taxKeeper:      taxKeeper,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	origMsgServer := keeper.NewMsgServerImpl(am.keeper)
	types.RegisterMsgServer(cfg.MsgServer(), handlers.NewBankMsgServer(am.keeper, am.treasuryKeeper, am.taxKeeper, origMsgServer))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	m := keeper.NewMigrator(am.keeper.(keeper.BaseKeeper), am.legacySubspace)
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 1 to 2: %v", err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 2 to 3: %v", err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 3 to 4: %v", err))
	}
}
