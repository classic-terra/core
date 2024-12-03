package market

import (
	"github.com/classic-terra/core/v3/x/tax/handlers"

	"github.com/classic-terra/core/v3/x/market"
	"github.com/classic-terra/core/v3/x/market/keeper"
	"github.com/classic-terra/core/v3/x/market/types"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the bank module.
type AppModuleBasic struct {
	*market.AppModuleBasic
}

// AppModule implements an application module for the bank module.
type AppModule struct {
	*market.AppModule

	keeper         keeper.Keeper
	accountKeeper  types.AccountKeeper
	treasuryKeeper treasurykeeper.Keeper
	taxKeeper      taxkeeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, accountKeeper types.AccountKeeper, treasuryKeeper treasurykeeper.Keeper, bankKeeper types.BankKeeper, oracleKeeper types.OracleKeeper, taxKeeper taxkeeper.Keeper) AppModule {
	bm := market.NewAppModule(cdc, keeper, accountKeeper, bankKeeper, oracleKeeper)
	return AppModule{
		AppModule:      &bm,
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		treasuryKeeper: treasuryKeeper,
		taxKeeper:      taxKeeper,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	origMsgServer := keeper.NewMsgServerImpl(am.keeper)
	types.RegisterMsgServer(cfg.MsgServer(), handlers.NewMarketMsgServer(am.keeper, am.treasuryKeeper, am.taxKeeper, origMsgServer))
	querier := keeper.NewQuerier(am.keeper)
	types.RegisterQueryServer(cfg.QueryServer(), querier)
}
