package post

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	oracle "github.com/classic-terra/core/v3/x/oracle/types"
	treasury "github.com/classic-terra/core/v3/x/treasury/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// BurnTaxSplit splits
func (fd Tax2gasPostDecorator) BurnTaxSplit(ctx sdk.Context, taxes sdk.Coins) (err error) {
	burnSplitRate := fd.treasuryKeeper.GetBurnSplitRate(ctx)
	oracleSplitRate := fd.treasuryKeeper.GetOracleSplitRate(ctx)
	communityTax := fd.distrKeeper.GetCommunityTax(ctx)
	distributionDeltaCoins := sdk.NewCoins()
	oracleSplitCoins := sdk.NewCoins()
	communityTaxCoins := sdk.NewCoins()

	if burnSplitRate.IsPositive() {

		for _, taxCoin := range taxes {
			splitcoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
			distributionDeltaCoins = distributionDeltaCoins.Add(sdk.NewCoin(taxCoin.Denom, splitcoinAmount))
		}

		taxes = taxes.Sub(distributionDeltaCoins...)
	}

	if communityTax.IsPositive() {

		// we need to apply a reduced community tax here as the community tax is applied again during distribution
		// in the distribution module and we don't want to calculate the tax twice
		// the reduction depends on the oracle split rate as well as on the community tax itself
		// the formula can be applied even with a zero oracle split rate
		applyCommunityTax := communityTax.Mul(oracleSplitRate.Quo(communityTax.Mul(oracleSplitRate).Add(sdk.OneDec()).Sub(communityTax)))

		for _, distrCoin := range distributionDeltaCoins {
			communityTaxAmount := applyCommunityTax.MulInt(distrCoin.Amount).RoundInt()
			communityTaxCoins = communityTaxCoins.Add(sdk.NewCoin(distrCoin.Denom, communityTaxAmount))
		}

		distributionDeltaCoins = distributionDeltaCoins.Sub(communityTaxCoins...)
	}

	if !communityTaxCoins.IsZero() {
		if err = fd.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.FeeCollectorName,
			distributiontypes.ModuleName,
			communityTaxCoins,
		); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}

		feePool := fd.distrKeeper.GetFeePool(ctx)
		feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(communityTaxCoins...)...)
		fd.distrKeeper.SetFeePool(ctx, feePool)
	}

	if oracleSplitRate.IsPositive() {
		for _, distrCoin := range distributionDeltaCoins {
			oracleCoinAmnt := oracleSplitRate.MulInt(distrCoin.Amount).RoundInt()
			oracleSplitCoins = oracleSplitCoins.Add(sdk.NewCoin(distrCoin.Denom, oracleCoinAmnt))
		}
	}

	if !oracleSplitCoins.IsZero() {
		if err = fd.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.FeeCollectorName,
			oracle.ModuleName,
			oracleSplitCoins,
		); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	if !taxes.IsZero() {
		if err = fd.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.FeeCollectorName,
			treasury.BurnModuleName,
			taxes,
		); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	// Record tax proceeds
	fd.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)
	return nil
}
