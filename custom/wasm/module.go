package wasm

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/simulation"
	types "github.com/CosmWasm/wasmd/x/wasm/types"
	customcli "github.com/classic-terra/core/v4/custom/wasm/client/cli"
	v4 "github.com/classic-terra/core/v4/custom/wasm/migrations/v4"
	customtypes "github.com/classic-terra/core/v4/custom/wasm/types/legacy"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/spf13/cobra"
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
	cdc            codec.Codec
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
	storeKey       storetypes.StoreKey
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
	storeKey storetypes.StoreKey,
) AppModule {
	return AppModule{
		AppModule:      wasm.NewAppModule(cdc, keeper, validatorSetSource, ak, bk, router, ss),
		cdc:            cdc,
		keeper:         keeper,
		legacySubspace: ss,
		storeKey:       storeKey,
	}
}

// ConsensusVersion returns the consensus-breaking version for the wasm module.
// Bumped to 5 to account for the ContractInfo field order migration (v0.61.4 -> v0.61.5).
func (AppModule) ConsensusVersion() uint64 { return 5 }

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	// Register the query service
	originalQueryServer := keeper.Querier(am.keeper)
	types.RegisterQueryServer(
		cfg.QueryServer(),
		NewLegacyQueryServer(
			originalQueryServer,
			am.keeper,
			am.storeKey,
		),
	)

	// For wasm module, we need to dereference the keeper pointer
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

	// Register v4→v5 migration for ContractInfo field order fix (wasmd v0.61.4 -> v0.61.5)
	storeService := runtime.NewKVStoreService(am.storeKey.(*storetypes.KVStoreKey))
	v4Migrator := v4.NewMigrator(func(ctx context.Context, contractAddress sdk.AccAddress, contractInfo *types.ContractInfo) {
		store := storeService.OpenKVStore(ctx)
		if err := store.Set(types.GetContractAddressKey(contractAddress), am.cdc.MustMarshal(contractInfo)); err != nil {
			panic(err)
		}
	})
	err = cfg.RegisterMigration(types.ModuleName, 4, func(ctx sdk.Context) error {
		return v4Migrator.Migrate4to5(ctx, storeService, am.cdc)
	})
	if err != nil {
		panic(err)
	}
}
