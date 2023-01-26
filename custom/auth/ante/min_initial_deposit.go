package ante

import (
	"fmt"
	//"strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	core "github.com/terra-money/core/types"
)

const (
	minInitialDepositRatio      int64 = 2
	minInitialDepositRatioPrec int64 = 2
)

var minInitialDepositEnableBlockHeight = map[string]int64{
	"columbus-5": 11_440_000,
	"rebel-2":    12_690_000,
}

// MinInitialDeposit Decorator will check Initial Deposits for MsgSubmitProposal
type MinInitialDepositDecorator struct {
	govKeeper GovKeeper
}

// NewMinInitialDeposit returns new min initial deposit decorator instance
func NewMinInitialDepositDecorator(govKeeper GovKeeper) MinInitialDepositDecorator {
	return MinInitialDepositDecorator{
		govKeeper: govKeeper,
	}
}

// IsMsgSubmitProposal checks wheter the input msg is a MsgSubmitProposal
func IsMsgSubmitProposal(msg sdk.Msg) bool {
	_, ok := msg.(*govtypes.MsgSubmitProposal)
	return ok
}

// HandleCheckMinInitialDeposit
func HandleCheckMinInitialDeposit(ctx sdk.Context, msg sdk.Msg, govKeeper GovKeeper) (err error) {
	submitPropMsg, ok := msg.(*govtypes.MsgSubmitProposal)
	if !ok {
		return fmt.Errorf("Could not dereference msg as MsgSubmitProposal")
	}

	minDeposit := govKeeper.GetDepositParams(ctx).MinDeposit
	requiredAmount := sdk.NewDecFromInt(minDeposit.AmountOf(core.MicroLunaDenom)).Mul(sdk.NewDecWithPrec(minInitialDepositRatio, minInitialDepositRatioPrec)).TruncateInt()

	requiredDepositCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, requiredAmount),
	)
	initialDepositCoins := submitPropMsg.GetInitialDeposit()

	if !initialDepositCoins.IsAllGTE(requiredDepositCoins) {
		return fmt.Errorf("Not enough initial deposit provided. Expected %q; got %q", requiredDepositCoins, initialDepositCoins)
	}

	return nil
}

// AnteHandle handles checking MsgSubmitProposal
func (btfd MinInitialDepositDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	enableHeight, ok := minInitialDepositEnableBlockHeight[ctx.ChainID()]

	if ok && (ctx.BlockHeight() < enableHeight) {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {

		if !IsMsgSubmitProposal(msg) {
			continue
		}

		err := HandleCheckMinInitialDeposit(ctx, msg, btfd.govKeeper)
		if err != nil {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
		}

	}

	return next(ctx, tx, simulate)
}
