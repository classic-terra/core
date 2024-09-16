package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	core "github.com/classic-terra/core/v3/types"
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
	switch msg.(type) {
	case *govv1beta1.MsgSubmitProposal, *govv1.MsgSubmitProposal:
		return true
	default:
		return false
	}
}

// HandleCheckMinInitialDeposit
func HandleCheckMinInitialDeposit(ctx sdk.Context, msg sdk.Msg, govKeeper govkeeper.Keeper, treasuryKeeper TreasuryKeeper) (err error) {
	var initialDepositCoins sdk.Coins

	switch submitPropMsg := msg.(type) {
	case *govv1beta1.MsgSubmitProposal:
		initialDepositCoins = submitPropMsg.GetInitialDeposit()
	case *govv1.MsgSubmitProposal:
		initialDepositCoins = submitPropMsg.GetInitialDeposit()
	default:
		return fmt.Errorf("could not dereference msg as MsgSubmitProposal")
	}
	minDeposit := govKeeper.GetParams(ctx).MinDeposit
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
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		if !IsMsgSubmitProposal(msg) {
			continue
		}

		err := HandleCheckMinInitialDeposit(ctx, msg, midd.govKeeper, midd.treasuryKeeper)
		if err != nil {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
		}
	}

	return next(ctx, tx, simulate)
}
