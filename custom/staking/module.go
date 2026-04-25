package staking

import (
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	customtypes "github.com/classic-terra/core/v4/custom/staking/types"
	core "github.com/classic-terra/core/v4/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// AppModuleBasic defines the basic application module used by the staking module.
type AppModuleBasic struct {
	staking.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the staking module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (am AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// customize to set default genesis state deposit denom to uluna
	defaultGenesisState := stakingtypes.DefaultGenesisState()
	defaultGenesisState.Params.BondDenom = core.MicroLunaDenom

	return cdc.MustMarshalJSON(defaultGenesisState)
}

// AppModule implements an application module for the staking module.
type AppModule struct {
	staking.AppModule

	cdc          codec.Codec
	keeper       *keeper.Keeper
	paramsKeeper paramskeeper.Keeper
	ss           paramtypes.Subspace
	storeKey     storetypes.StoreKey
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec,
	keeper *keeper.Keeper,
	ak stakingtypes.AccountKeeper,
	bk stakingtypes.BankKeeper,
	pk paramskeeper.Keeper,
	ss paramtypes.Subspace,
	storeKey storetypes.StoreKey,
) AppModule {
	return AppModule{
		AppModule:    staking.NewAppModule(cdc, keeper, ak, bk, ss),
		cdc:          cdc,
		keeper:       keeper,
		paramsKeeper: pk,
		ss:           ss,
		storeKey:     storeKey,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	stakingtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	querier := keeper.Querier{Keeper: am.keeper}
	stakingtypes.RegisterQueryServer(
		cfg.QueryServer(),
		NewLegacyQueryServer(querier, am.ss, am.keeper, am.cdc, am.storeKey),
	)

	m := keeper.NewMigrator(am.keeper, am.ss)
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 4, m.Migrate4to5); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 4 to 5: %v", stakingtypes.ModuleName, err))
	}
}
