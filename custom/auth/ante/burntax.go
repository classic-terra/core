package ante

import (
	treasury "github.com/classic-terra/core/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// BurnTaxFeeDecorator will immediately burn the collected Tax
type BurnTaxFeeDecorator struct {
	treasuryKeeper TreasuryKeeper
	bankKeeper     BankKeeper
	distrKeeper    DistrKeeper
}

// NewBurnTaxFeeDecorator returns new tax fee decorator instance
func NewBurnTaxFeeDecorator(treasuryKeeper TreasuryKeeper, bankKeeper BankKeeper, distrKeeper DistrKeeper) BurnTaxFeeDecorator {
	return BurnTaxFeeDecorator{
		treasuryKeeper: treasuryKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
	}
}

// AnteHandle handles msg tax fee checking
func (btfd BurnTaxFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	msgs := feeTx.GetMsgs()

	// At this point we have already run the DeductFees AnteHandler and taken the fees from the sending account
	// Now we remove the taxes from the gas reward and immediately burn it
	if !simulate {
		// Compute taxes again.
		taxes := FilterMsgAndComputeTax(ctx, btfd.treasuryKeeper, msgs...)

		// Record tax proceeds
		if !taxes.IsZero() {
			burnSplitRate := btfd.treasuryKeeper.GetBurnSplitRate(ctx)

			if burnSplitRate.IsPositive() {
				communityDeltaCoins := sdk.NewCoins()

				for i, taxCoin := range taxes {
					splitcoinAmount := burnSplitRate.MulInt(taxCoin.Amount).RoundInt()
					splitCoin := sdk.NewCoin(taxCoin.Denom, splitcoinAmount)
					taxes[i] = taxCoin.Sub(splitCoin)

					if splitcoinAmount.IsPositive() {
						communityDeltaCoins = communityDeltaCoins.Add(splitCoin)
					}
				}

				if err = btfd.bankKeeper.SendCoinsFromModuleToModule(
					ctx,
					types.FeeCollectorName,
					distrtypes.ModuleName,
					communityDeltaCoins,
				); err != nil {
					return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}

				feePool := btfd.distrKeeper.GetFeePool(ctx)
				feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(communityDeltaCoins...)...)
				btfd.distrKeeper.SetFeePool(ctx, feePool)
			}

			if !taxes.IsZero() {
				if err = btfd.bankKeeper.SendCoinsFromModuleToModule(
					ctx,
					types.FeeCollectorName,
					treasury.BurnModuleName,
					taxes,
				); err != nil {
					return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}
			}
		}
	}

	return next(ctx, tx, simulate)
}
