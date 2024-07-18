package post

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	treasury "github.com/classic-terra/core/v3/x/treasury/types"
)

// BurnTaxSplit splits
func (tdd Tax2gasPostDecorator) BurnTaxSplit(ctx sdk.Context, taxes sdk.Coins) (err error) {
	burnSplitRate := tdd.treasuryKeeper.GetBurnSplitRate(ctx)

	if burnSplitRate.IsPositive() {
		distributionDeltaCoins := sdk.NewCoins()

		for _, taxCoin := range taxes {
			splitcoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
			distributionDeltaCoins = distributionDeltaCoins.Add(sdk.NewCoin(taxCoin.Denom, splitcoinAmount))
		}

		taxes = taxes.Sub(distributionDeltaCoins...)
	}

	if !taxes.IsZero() {
		if err = tdd.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.FeeCollectorName,
			treasury.BurnModuleName,
			taxes,
		); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	// Record tax proceeds
	tdd.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)
	return nil
}
