package v13

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CreateV13UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Initialize/ensure allowed swap denoms for market: restrict to uusd by default.
		k.MarketKeeper.SetAllowedSwapDenoms([]string{"uusd"})

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
