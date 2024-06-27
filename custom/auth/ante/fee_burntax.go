package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	oracle "github.com/classic-terra/core/v3/x/oracle/types"
	treasury "github.com/classic-terra/core/v3/x/treasury/types"
)

// BurnTaxSplit splits
func (fd FeeDecorator) BurnTaxSplit(ctx sdk.Context, taxes sdk.Coins) (err error) {
	burnSplitRate := fd.treasuryKeeper.GetBurnSplitRate(ctx)
	oracleSplitRate := fd.treasuryKeeper.GetOracleSplitRate(ctx)
	distributionDeltaCoins := sdk.NewCoins()
	oracleSplitCoins := sdk.NewCoins()

	if burnSplitRate.IsPositive() {

		for _, taxCoin := range taxes {
			splitcoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
			distributionDeltaCoins = distributionDeltaCoins.Add(sdk.NewCoin(taxCoin.Denom, splitcoinAmount))
		}

		taxes = taxes.Sub(distributionDeltaCoins...)
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

	return nil
}
