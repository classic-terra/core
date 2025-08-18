package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func (k Keeper) ProcessTaxSplits(ctx sdk.Context, taxes sdk.Coins) error {
	burnSplitRate := k.treasuryKeeper.GetBurnSplitRate(ctx)
	oracleSplitRate := k.treasuryKeeper.GetOracleSplitRate(ctx)
	communityTax := k.distributionKeeper.GetCommunityTax(ctx)
	distributionDeltaCoins := sdk.NewCoins()
	oracleSplitCoins := sdk.NewCoins()
	communityTaxCoins := sdk.NewCoins()
	// Redirect rate configured in treasury params (co-located with oracle split)
	marketRedirectRate := k.treasuryKeeper.GetTaxRedirectRate(ctx)

	// 1) Redirect to market accumulator FROM FULL TAX first
	marketSplitCoins := sdk.NewCoins()
	if marketRedirectRate.IsPositive() && !taxes.IsZero() {
		for _, taxCoin := range taxes {
			redirectAmt := marketRedirectRate.MulInt(taxCoin.Amount).RoundInt()
			if redirectAmt.IsPositive() {
				marketSplitCoins = marketSplitCoins.Add(sdk.NewCoin(taxCoin.Denom, redirectAmt))
			}
		}
		// Deduct redirected portion from taxes before any other splits
		taxes = taxes.Sub(marketSplitCoins...)
	}

	// Calculate distribution delta coins (amount to be split between burn, oracle, etc.)
	if burnSplitRate.IsPositive() {
		for _, taxCoin := range taxes {
			splitCoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
			distributionDeltaCoins = distributionDeltaCoins.Add(sdk.NewCoin(taxCoin.Denom, splitCoinAmount))
		}
		taxes = taxes.Sub(distributionDeltaCoins...)
	}

	// Calculate community tax coins
	if communityTax.IsPositive() {
		// Adjust community tax to avoid double taxation
		applyCommunityTax := communityTax.Mul(oracleSplitRate.Quo(communityTax.Mul(oracleSplitRate).Add(sdk.OneDec()).Sub(communityTax)))

		for _, distrCoin := range distributionDeltaCoins {
			communityTaxAmount := applyCommunityTax.MulInt(distrCoin.Amount).RoundInt()
			communityTaxCoins = communityTaxCoins.Add(sdk.NewCoin(distrCoin.Denom, communityTaxAmount))
		}

		distributionDeltaCoins = distributionDeltaCoins.Sub(communityTaxCoins...)
	}

	// Calculate oracle split coins
	if oracleSplitRate.IsPositive() {
		for _, distrCoin := range distributionDeltaCoins {
			oracleCoinAmount := oracleSplitRate.MulInt(distrCoin.Amount).RoundInt()
			oracleSplitCoins = oracleSplitCoins.Add(sdk.NewCoin(distrCoin.Denom, oracleCoinAmount))
		}
	}

	// Handle community tax coins
	if !communityTaxCoins.IsZero() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			authtypes.FeeCollectorName,
			distributiontypes.ModuleName,
			communityTaxCoins,
		); err != nil {
			return err
		}

		// Add to community pool
		feePool := k.distributionKeeper.GetFeePool(ctx)
		feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(communityTaxCoins...)...)
		k.distributionKeeper.SetFeePool(ctx, feePool)

		// Emit event for community tax transfer
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				taxtypes.EventTaxCommunity,
				sdk.NewAttribute(sdk.AttributeKeyModule, "tax"),
				sdk.NewAttribute(taxtypes.AttributeKeyFromModule, authtypes.FeeCollectorName),
				sdk.NewAttribute(taxtypes.AttributeKeyToModule, distributiontypes.ModuleName),
				sdk.NewAttribute(taxtypes.AttributeKeyAmount, communityTaxCoins.String()),
				sdk.NewAttribute(taxtypes.AttributeKeyHeight, sdk.NewInt(ctx.BlockHeight()).String()),
			),
		)
	}

	// Handle oracle split coins
	if !oracleSplitCoins.IsZero() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			authtypes.FeeCollectorName,
			oracletypes.ModuleName,
			oracleSplitCoins,
		); err != nil {
			return err
		}

		// Emit event for oracle split transfer
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				taxtypes.EventTaxOracle,
				sdk.NewAttribute(sdk.AttributeKeyModule, "tax"),
				sdk.NewAttribute(taxtypes.AttributeKeyFromModule, authtypes.FeeCollectorName),
				sdk.NewAttribute(taxtypes.AttributeKeyToModule, oracletypes.ModuleName),
				sdk.NewAttribute(taxtypes.AttributeKeyAmount, oracleSplitCoins.String()),
				sdk.NewAttribute(taxtypes.AttributeKeyHeight, sdk.NewInt(ctx.BlockHeight()).String()),
			),
		)
	}

	// Handle market split coins (redirected first from full taxes) to market accumulator
	if !marketSplitCoins.IsZero() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			authtypes.FeeCollectorName,
			markettypes.AccumulatorModuleName,
			marketSplitCoins,
		); err != nil {
			return err
		}

		// Emit event for market redirect transfer
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				taxtypes.EventTaxMarketRedirect,
				sdk.NewAttribute(sdk.AttributeKeyModule, "tax"),
				sdk.NewAttribute(taxtypes.AttributeKeyFromModule, authtypes.FeeCollectorName),
				sdk.NewAttribute(taxtypes.AttributeKeyToModule, markettypes.AccumulatorModuleName),
				sdk.NewAttribute(taxtypes.AttributeKeyAmount, marketSplitCoins.String()),
				sdk.NewAttribute(taxtypes.AttributeKeyHeight, sdk.NewInt(ctx.BlockHeight()).String()),
			),
		)
	}

	// Handle remaining taxes (burn)
	if !taxes.IsZero() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			authtypes.FeeCollectorName,
			treasurytypes.BurnModuleName,
			taxes,
		); err != nil {
			return err
		}

		// Emit event for burn of remaining taxes
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				taxtypes.EventTaxBurn,
				sdk.NewAttribute(sdk.AttributeKeyModule, "tax"),
				sdk.NewAttribute(taxtypes.AttributeKeyFromModule, authtypes.FeeCollectorName),
				sdk.NewAttribute(taxtypes.AttributeKeyToModule, treasurytypes.BurnModuleName),
				sdk.NewAttribute(taxtypes.AttributeKeyAmount, taxes.String()),
				sdk.NewAttribute(taxtypes.AttributeKeyHeight, sdk.NewInt(ctx.BlockHeight()).String()),
			),
		)
	}

	return nil
}
