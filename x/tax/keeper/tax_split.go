package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
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
	}

	return nil
}
