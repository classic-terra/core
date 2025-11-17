package v15

import (
	"context"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV15UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Initialize/ensure allowed swap denoms for market: restrict to uusd by default.
		k.MarketKeeper.SetAllowedSwapDenoms([]string{"uusd"})

		// Ensure UST meta denom (oracle-only) is present in oracle vote targets.
		// Existing chains won't pick up DefaultParams changes automatically, so patch params here.
		params := k.OracleKeeper.GetParams(sdkCtx)
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
				TobinTax: sdkmath.LegacyZeroDec(),
			})
			k.OracleKeeper.SetParams(sdkCtx, params)
			// Set TobinTax immediately so it becomes a vote target without waiting a full period
			k.OracleKeeper.SetTobinTax(sdkCtx, oracletypes.MetaUSDDenom, sdkmath.LegacyZeroDec())
		}

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
