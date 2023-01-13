package ante

import (
	treasury "github.com/classic-terra/classic/x/treasury/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

// TaxPowerUpgradeHeight is when taxes are allowed to go into effect
// This will still need a parameter change proposal, but can be activated
// anytime after this height
const (
	TaxPowerUpgradeHeight = 9346889
	TaxPowerSplitHeight   = 123456789
)

// BurnTaxFeeDecorator will immediately burn the collected Tax
type BurnTaxFeeDecorator struct {
	TreasuryKeeper     TreasuryKeeper
	DistributionKeeper distribution.Keeper
	BankKeeper         BankKeeper
}

// NewBurnTaxFeeDecorator returns new tax fee decorator instance
func NewBurnTaxFeeDecorator(treasuryKeeper TreasuryKeeper, bankKeeper BankKeeper, distributionKeeper distribution.Keeper) BurnTaxFeeDecorator {
	return BurnTaxFeeDecorator{
		TreasuryKeeper:     treasuryKeeper,
		DistributionKeeper: distributionKeeper,
		BankKeeper:         bankKeeper,
	}
}

// AnteHandle handles msg tax fee checking
func (btfd BurnTaxFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Do not proceed if you are below this block height
	currHeight := ctx.BlockHeight()
	if currHeight < TaxPowerUpgradeHeight {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	msgs := feeTx.GetMsgs()

	// At this point we have already run the DeductFees AnteHandler and taken the fees from the sending account
	// Now we remove the taxes from the gas reward and immediately burn it

	if !simulate {
		// Compute taxes again.
		taxes := FilterMsgAndComputeTax(ctx, btfd.TreasuryKeeper, msgs...)

		// Record tax proceeds
		if !taxes.IsZero() {
			if currHeight >= TaxPowerSplitHeight {
				feePool := btfd.DistributionKeeper.GetFeePool(ctx)

				for _, taxCoin := range taxes {
					splitCoin := sdk.NewCoin(taxCoin.Denom, taxCoin.Amount.Quo(sdk.NewInt(2)))
					taxCoin.Amount = taxCoin.Amount.Sub(splitCoin.Amount)

					if splitCoin.Amount.IsPositive() {
						feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinFromCoin(splitCoin))
					}
				}

				btfd.DistributionKeeper.SetFeePool(ctx, feePool)
			}

			err = btfd.BankKeeper.SendCoinsFromModuleToModule(ctx, types.FeeCollectorName, treasury.BurnModuleName, taxes)

			if err != nil {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
			}
		}
	}

	return next(ctx, tx, simulate)
}
