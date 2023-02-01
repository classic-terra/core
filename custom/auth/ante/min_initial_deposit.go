package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	core "github.com/terra-money/core/types"
)

// MinInitialDeposit Decorator will check Initial Deposits for MsgSubmitProposal
type MinInitialDepositDecorator struct {
	govKeeper      GovKeeper
	treasuryKeeper TreasuryKeeper
}

// NewMinInitialDeposit returns new min initial deposit decorator instance
func NewMinInitialDepositDecorator(govKeeper GovKeeper, treasuryKeeper TreasuryKeeper) MinInitialDepositDecorator {
	return MinInitialDepositDecorator{
		govKeeper:      govKeeper,
		treasuryKeeper: treasuryKeeper,
	}
}

// IsMsgSubmitProposal checks whether the input msg is a MsgSubmitProposal
func IsMsgSubmitProposal(msg sdk.Msg) bool {
	_, ok := msg.(*govtypes.MsgSubmitProposal)
	return ok
}

// HandleCheckMinInitialDeposit
func HandleCheckMinInitialDeposit(ctx sdk.Context, msg sdk.Msg, govKeeper GovKeeper, treasuryKeeper TreasuryKeeper) (err error) {
	submitPropMsg, ok := msg.(*govtypes.MsgSubmitProposal)
	if !ok {
		return fmt.Errorf("could not dereference msg as MsgSubmitProposal")
	}

	minInitialDepositRatio := treasuryKeeper.GetParams(ctx).MinInitialDepositRatio

	minDeposit := govKeeper.GetDepositParams(ctx).MinDeposit
	requiredAmount := sdk.NewDecFromInt(minDeposit.AmountOf(core.MicroLunaDenom)).Mul(minInitialDepositRatio).TruncateInt()

	requiredDepositCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, requiredAmount),
	)
	initialDepositCoins := submitPropMsg.GetInitialDeposit()

	if !initialDepositCoins.IsAllGTE(requiredDepositCoins) {
		return fmt.Errorf("not enough initial deposit provided. Expected %q; got %q", requiredDepositCoins, initialDepositCoins)
	}

	return nil
}

// AnteHandle handles checking MsgSubmitProposal
func (btfd MinInitialDepositDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {

		if !IsMsgSubmitProposal(msg) {
			continue
		}

		err := HandleCheckMinInitialDeposit(ctx, msg, btfd.govKeeper, btfd.treasuryKeeper)
		if err != nil {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
		}

	}

	return next(ctx, tx, simulate)
}
