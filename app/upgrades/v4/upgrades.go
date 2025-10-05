package v4

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV4UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Migrate13to14 migrates from version v0.45.13 to v0.45.14.
		// Only for this particular version, which do not use the version of module.
		// stakingMigrator.Migrate13to14(ctx)

		// to run wasm store migration
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
