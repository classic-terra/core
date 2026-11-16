//nolint:revive
package v14_5

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV145UpgradeHandler wires module migrations for v14_5 (core v4.2.0).
// The handler is intentionally empty for now: it only runs module migrations
// and exists to coordinate a chain-wide halt and binary switch via governance.
// The binary itself contains state-machine-breaking changes (e.g. taxexemption
// msgs become submittable via gov), so it must not be rolled out without it.
func CreateV145UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
