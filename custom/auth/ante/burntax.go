package ante

import (
	treasury "github.com/classic-terra/core/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// TaxPowerUpgradeHeight is when taxes are allowed to go into effect
// This will still need a parameter change proposal, but can be activated
// anytime after this height
const TaxPowerUpgradeHeight = 9346889

// BurnTaxFeeDecorator will immediately burn the collected Tax
type BurnTaxFeeDecorator struct {
	accountKeeper  cosmosante.AccountKeeper
	treasuryKeeper TreasuryKeeper
	bankKeeper     BankKeeper
	distrKeeper    DistrKeeper
}

// NewBurnTaxFeeDecorator returns new tax fee decorator instance
func NewBurnTaxFeeDecorator(accountKeeper cosmosante.AccountKeeper, treasuryKeeper TreasuryKeeper, bankKeeper BankKeeper, distrKeeper DistrKeeper) BurnTaxFeeDecorator {
	return BurnTaxFeeDecorator{
		accountKeeper:  accountKeeper,
		treasuryKeeper: treasuryKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
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
		taxes := FilterMsgAndComputeTax(ctx, btfd.treasuryKeeper, msgs...)

		// Record tax proceeds
		if !taxes.IsZero() {
			tainted := false

			// Iterate over messages
			for _, msg := range msgs {
				var whitelistedRecipients []string
				var whitelistedSenders []string
				senderWhitelistCount := 0
				recipientWhitelistCount := 0

				// Fetch recipients
				switch v := msg.(type) {
				case *banktypes.MsgSend:
					whitelistedRecipients = append(whitelistedRecipients, v.ToAddress)
					whitelistedSenders = append(whitelistedSenders, v.FromAddress)
				case *banktypes.MsgMultiSend:
					for _, output := range v.Outputs {
						whitelistedRecipients = append(whitelistedRecipients, output.Address)
					}

					for _, input := range v.Inputs {
						whitelistedSenders = append(whitelistedSenders, input.Address)
					}
				default:
					// TODO: We might want to return an error if we cannot match the msg types, but as such I think that means we also need to cover MsgSetSendEnabled & MsgUpdateParams
					// return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidType, "Unsupported message type")
				}

				// Match signers vs whitelist
				for _, sender := range whitelistedSenders {
					if btfd.treasuryKeeper.HasWhitelistAddress(ctx, sender) {
						senderWhitelistCount++
					}
				}

				// If all signers are not matched apply burn tax
				if len(whitelistedSenders) > senderWhitelistCount {
					tainted = true

					break
				}

				// Check recipients
				for _, recipient := range whitelistedRecipients {
					if btfd.treasuryKeeper.HasWhitelistAddress(ctx, recipient) {
						recipientWhitelistCount++
					}
				}

				// If all recipients are not matched apply burn tax
				if len(whitelistedRecipients) > recipientWhitelistCount {
					tainted = true

					break
				}
			}

			if !tainted {
				return next(ctx, tx, simulate)
			}

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

				if err = btfd.distrKeeper.FundCommunityPool(
					ctx,
					communityDeltaCoins,
					btfd.accountKeeper.GetModuleAddress(types.FeeCollectorName),
				); err != nil {
					return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
				}
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
