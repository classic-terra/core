//nolint:revive
package v8_1

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV81UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		// set default oracle split
		keepers.TreasuryKeeper.SetOracleSplitRate(sdkCtx, treasurytypes.DefaultOracleSplit)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
