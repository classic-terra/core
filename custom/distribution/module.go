package distribution

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	customtypes "github.com/classic-terra/core/v3/custom/distribution/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// AppModuleBasic defines the basic application module used by the distribution module.
type AppModuleBasic struct {
	distribution.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the distribution module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
	*distrtypes.ModuleCdc = *customtypes.ModuleCdc
}

// AppModule implements an application module for the distribution module.
type AppModule struct {
	distribution.AppModule

	keeper         keeper.Keeper
	legacySubspace paramtypes.Subspace
	upgradeHeight  int64
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper, accountKeeper distrtypes.AccountKeeper, bankKeeper distrtypes.BankKeeper, stakingKeeper distrtypes.StakingKeeper, ss paramtypes.Subspace, upgradeHeight int64) AppModule {
	return AppModule{
		AppModule:      distribution.NewAppModule(cdc, k, accountKeeper, bankKeeper, stakingKeeper, ss),
		keeper:         k,
		legacySubspace: ss,
		upgradeHeight:  upgradeHeight,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	distrQuerySrv := keeper.NewQuerier(am.keeper)
	distrtypes.RegisterQueryServer(
		cfg.QueryServer(),
		NewLegacyQueryServer(distrQuerySrv, am.legacySubspace, &am.keeper, am.upgradeHeight),
	)

	distrtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	m := keeper.NewMigrator(am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(distrtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/distribution from version 1 to 2: %v", err))
	}
	if err := cfg.RegisterMigration(distrtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/distribution from version 2 to 3: %v", err))
	}
}
