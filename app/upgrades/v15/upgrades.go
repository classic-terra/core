package v15

import (
	"context"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
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

		// Note: allowedSwapDenoms is deliberately not touched here. It is code-fixed to
		// {uusd} by marketkeeper.NewKeeper and cannot be usefully changed at this point

		// Initialize the market params added in this upgrade.
		marketParams := k.MarketKeeper.GetParams(sdkCtx)
		marketParams.EpochLengthBlocks = markettypes.DefaultEpochLengthBlocks
		marketParams.SwapFeeBurnRate = markettypes.DefaultSwapFeeBurnRate
		marketParams.SwapFeeCommunityRate = markettypes.DefaultSwapFeeCommunityRate
		marketParams.MaxOracleAgeSeconds = markettypes.DefaultMaxOracleAgeSeconds
		marketParams.TwapLookbackWindow = markettypes.DefaultTWAPLookbackWindow
		marketParams.MaxTwapDeviation = markettypes.DefaultMaxTWAPDeviation
		marketParams.DailyCapFactor = markettypes.DefaultDailyCapFactor
		k.MarketKeeper.SetParams(sdkCtx, marketParams)

		// Initialize TaxRedirectRate. It is a new treasury param added in this upgrade.
		k.TreasuryKeeper.SetTaxRedirectRate(sdkCtx, treasurytypes.DefaultTaxRedirectRate)

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
