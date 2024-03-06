package ante

import (
	"fmt"

	core "github.com/classic-terra/core/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// MinInitialDeposit Decorator will check Initial Deposits for MsgSubmitProposal
type MinInitialDepositDecorator struct {
	govKeeper      govkeeper.Keeper
	treasuryKeeper TreasuryKeeper
}

// NewMinInitialDeposit returns new min initial deposit decorator instance
func NewMinInitialDepositDecorator(govKeeper govkeeper.Keeper, treasuryKeeper TreasuryKeeper) MinInitialDepositDecorator {
	return MinInitialDepositDecorator{
		govKeeper:      govKeeper,
		treasuryKeeper: treasuryKeeper,
	}
}

// IsMsgSubmitProposal checks whether the input msg is a MsgSubmitProposal
func IsMsgSubmitProposal(msg sdk.Msg) bool {
	_, okv1beta1 := msg.(*govv1beta1.MsgSubmitProposal)
	_, okv1 := msg.(*govv1.MsgSubmitProposal)
	return okv1beta1 || okv1
}

// HandleCheckMinInitialDeposit
func HandleCheckMinInitialDeposit(ctx sdk.Context, msg sdk.Msg, govKeeper govkeeper.Keeper, treasuryKeeper TreasuryKeeper) (err error) {
	submitPropMsgV1Beta1, okV1Beta1 := msg.(*govv1beta1.MsgSubmitProposal)
	submitPropMsgV1, okV1 := msg.(*govv1.MsgSubmitProposal)

	if !(okV1Beta1 || okV1) {
		return fmt.Errorf("could not dereference msg as MsgSubmitProposal")
	}

	var initialDepositCoins sdk.Coins

	if okV1Beta1 {
		initialDepositCoins = submitPropMsgV1Beta1.GetInitialDeposit()
	} else if okV1 {
		initialDepositCoins = submitPropMsgV1.GetInitialDeposit()
	}

	minDeposit := govKeeper.GetDepositParams(ctx).MinDeposit
	requiredAmount := sdk.NewDecFromInt(minDeposit[0].Amount).Mul(treasuryKeeper.GetMinInitialDepositRatio(ctx)).TruncateInt()

	requiredDepositCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, requiredAmount),
	)

	if !initialDepositCoins.IsAllGTE(requiredDepositCoins) {
		return fmt.Errorf("not enough initial deposit provided. Expected %q; got %q", requiredDepositCoins, initialDepositCoins)
	}

	return nil
}

// AnteHandle handles checking MsgSubmitProposal
func (midd MinInitialDepositDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		if !IsMsgSubmitProposal(msg) {
			continue
		}

		err := HandleCheckMinInitialDeposit(ctx, msg, midd.govKeeper, midd.treasuryKeeper)
		if err != nil {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
		}
	}

	return next(ctx, tx, simulate)
}
