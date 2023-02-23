package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	feesharekeeper "github.com/classic-terra/core/x/feeshare/keeper"
	feesharetypes "github.com/classic-terra/core/x/feeshare/types"
)

func CreateV2UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	fskeeper *feesharekeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// treasury store migration

		// set new FeeShare params
		newFeeShareParams := feesharetypes.Params{
			EnableFeeShare:  true,
			DeveloperShares: sdk.NewDecWithPrec(50, 2), // = 50%
			AllowedDenoms:   []string{"uluna"},
		}
		fskeeper.SetParams(ctx, newFeeShareParams)

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
