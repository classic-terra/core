package v10

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV10UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// set default oracle split
		keepers.TreasuryKeeper.SetTaxRate(ctx, sdk.ZeroDec())

		tax2gasParams := taxtypes.DefaultParams()
		keepers.TaxKeeper.SetParams(ctx, tax2gasParams)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
