package v13

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
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

		// Ensure UST meta denom (oracle-only) is present in oracle vote targets.
		// Existing chains won't pick up DefaultParams changes automatically, so patch params here.
		params := k.OracleKeeper.GetParams(ctx)
		hasMeta := false
		for _, d := range params.Whitelist {
			if d.Name == oracletypes.MetaUSDDenom {
				hasMeta = true
				break
			}
		}
		if !hasMeta {
			params.Whitelist = append(params.Whitelist, oracletypes.Denom{
				Name:     oracletypes.MetaUSDDenom,
				TobinTax: sdk.ZeroDec(),
			})
			k.OracleKeeper.SetParams(ctx, params)
			// Set TobinTax immediately so it becomes a vote target without waiting a full period
			k.OracleKeeper.SetTobinTax(ctx, oracletypes.MetaUSDDenom, sdk.ZeroDec())
		}

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
