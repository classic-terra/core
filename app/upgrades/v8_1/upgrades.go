//nolint:revive
package v8_1

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV81UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// set default oracle split
	        keepers.TreasuryKeeper.SetOracleSplitRate(ctx, treasurytypes.DefaultOracleSplit)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
