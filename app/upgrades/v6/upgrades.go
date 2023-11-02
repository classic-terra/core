package v6

import (
	"github.com/classic-terra/core/v2/app/keepers"
	"github.com/classic-terra/core/v2/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV6UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// as the stability tax rate is no longer used for the burn tax,
		// we set it to 0 to allow dApps/clients to run without changes and
		// without being double-taxed
		k.TreasuryKeeper.SetTaxRate(ctx, sdk.NewDec(0))

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
