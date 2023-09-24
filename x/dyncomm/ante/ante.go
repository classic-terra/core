package ante

import (
	"fmt"

	dyncommkeeper "github.com/classic-terra/core/v2/x/dyncomm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// DyncommDecorator checks for EditValidator and rejects
// edits that do not conform with dyncomm
type DyncommDecorator struct {
	dyncommKeeper dyncommkeeper.Keeper
	stakingKeeper stakingkeeper.Keeper
}

func NewDyncommDecorator(dk dyncommkeeper.Keeper, sk stakingkeeper.Keeper) DyncommDecorator {
	return DyncommDecorator{
		dyncommKeeper: dk,
		stakingKeeper: sk,
	}
}

// IsMsgSubmitProposal checks whether the input msg is a MsgSubmitProposal
func IsMsgEditValidator(msg sdk.Msg) bool {
	_, ok := msg.(*stakingtypes.MsgEditValidator)
	return ok
}

func (dd DyncommDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	err := dd.FilterMsgsAndCheckEditValidator(ctx, msgs...)

	if err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)

}

func (dd DyncommDecorator) FilterMsgsAndCheckEditValidator(ctx sdk.Context, msgs ...sdk.Msg) (err error) {

	for _, msg := range msgs {
		if !IsMsgEditValidator(msg) {
			continue
		}

		err := dd.CheckEditValidator(ctx, msg)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
		}
	}
	return nil

}

func (dd DyncommDecorator) CheckEditValidator(ctx sdk.Context, msg sdk.Msg) (err error) {

	msgEditValidator := msg.(*stakingtypes.MsgEditValidator)

	// no update of CommissionRate provided
	if msgEditValidator.CommissionRate == nil {
		return nil
	}

	operator := msgEditValidator.ValidatorAddress
	newIntendedRate := msgEditValidator.CommissionRate
	dynMinRate := dd.dyncommKeeper.GetDynCommissionRate(ctx, operator)

	if newIntendedRate.LT(dynMinRate) {
		return fmt.Errorf("commission for %s must be at least %f", operator, dynMinRate.MustFloat64())
	}

	return nil

}
