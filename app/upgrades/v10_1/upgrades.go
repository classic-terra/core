//nolint:revive
package v10_1

import (
	"context"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV101UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		keepers.TreasuryKeeper.SetTaxRate(sdkCtx, sdkmath.LegacyZeroDec())
		params := keepers.TreasuryKeeper.GetParams(sdkCtx)
		params.TaxPolicy.RateMax = sdkmath.LegacyZeroDec()
		params.TaxPolicy.RateMin = sdkmath.LegacyZeroDec()
		keepers.TreasuryKeeper.SetParams(sdkCtx, params)

		tax2gasParams := taxtypes.DefaultParams()
		keepers.TaxKeeper.SetParams(sdkCtx, tax2gasParams)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
