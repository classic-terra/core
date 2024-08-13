package v9

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV9UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// set default oracle split
		keepers.TreasuryKeeper.SetTaxRate(ctx, sdk.ZeroDec())

		tax2gasParams := tax2gastypes.DefaultParams()
		tax2gasParams.GasPrices = sdk.NewDecCoins(
			sdk.NewDecCoinFromDec("uluna", sdk.NewDecWithPrec(28325, 3)),
			sdk.NewDecCoinFromDec("uusd", sdk.NewDecWithPrec(75, 2)),
		)
		tax2gasParams.MaxTotalBypassMinFeeMsgGasUsage = 200000
		keepers.Tax2gasKeeper.SetParams(ctx, tax2gasParams)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
