package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	treasury "github.com/classic-terra/core/v2/x/treasury/types"
)

// BurnTaxSplit splits
func (k Keeper) BurnTaxSplit(ctx sdk.Context, taxes sdk.Coins) (err error) {
	burnSplitRate := k.treasuryKeeper.GetBurnSplitRate(ctx)

	if burnSplitRate.IsPositive() {
		distributionDeltaCoins := sdk.NewCoins()

		for _, taxCoin := range taxes {
			splitcoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
			distributionDeltaCoins = distributionDeltaCoins.Add(sdk.NewCoin(taxCoin.Denom, splitcoinAmount))
		}

		taxes = taxes.Sub(distributionDeltaCoins...)
	}

	ctx.Logger().Info("BurnTaxSplit", "rate", burnSplitRate, "taxes", taxes)
	if !taxes.IsZero() {
		if err = k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.FeeCollectorName,
			treasury.BurnModuleName,
			taxes,
		); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	return nil
}
