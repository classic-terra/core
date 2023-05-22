package v2

import (
	"github.com/classic-terra/core/app/keepers"
	"github.com/classic-terra/core/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

//	feesharetypes "github.com/classic-terra/core/x/feeshare/types"
)

func CreateV2UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	appKeepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// treasury store migration
		return mm.RunMigrations(ctx, cfg, fromVM)

		// set new FeeShare params
		// This is NOT part of any release -
		// V2 upgrade before this was included
		// we never ran this on the store
		/*newFeeShareParams := feesharetypes.Params{
			EnableFeeShare:  true,
			DeveloperShares: sdk.NewDecWithPrec(50, 2), // = 50%
			AllowedDenoms:   []string{"uluna"},
		}
		appKeepers.FeeShareKeeper.SetParams(ctx, newFeeShareParams)*/

	}
}
