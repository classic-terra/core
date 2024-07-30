//nolint:revive
package v8_2

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV82UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// set default oracle split
		keepers.TreasuryKeeper.SetTaxRate(ctx, sdk.ZeroDec())
		keepers.Tax2gasKeeper.SetParams(ctx, tax2gastypes.DefaultParams())
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
