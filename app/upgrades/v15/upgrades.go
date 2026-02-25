package v15

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV15UpgradeHandler creates the upgrade handler for v15.
// This upgrade triggers the wasm module migration from consensus version 4 to 5,
// which fixes the ContractInfo protobuf field order swap between wasmd v0.61.4 and v0.61.5.
func CreateV15UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
