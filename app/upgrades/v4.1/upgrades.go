package v4_1

import (
	"github.com/classic-terra/core/v2/app/keepers"
	"github.com/classic-terra/core/v2/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	wasm "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func CreateV4_1UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// to run staking store migration
		wasmkeeper := keepers.WasmKeeper
		wasmMigrator := wasm.NewMigrator(wasmkeeper)
		wasmMigrator.Migrate2to2(ctx)

		// to run wasm store migration
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
