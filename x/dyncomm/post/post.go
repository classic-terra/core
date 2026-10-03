package post

import (
	dyncommkeeper "github.com/classic-terra/core/v4/x/dyncomm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// DyncommDecorator does post runMsg store
// modifications for dyncomm module
type DyncommDecorator struct {
	dyncommKeeper dyncommkeeper.Keeper
}

func NewDyncommPostDecorator(dk dyncommkeeper.Keeper) DyncommDecorator {
	return DyncommDecorator{
		dyncommKeeper: dk,
	}
}

func (dd DyncommDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	// Only run in the modes that execute msgs. In CheckTx/ReCheckTx and in
	// PrepareProposal/ProcessProposal the msgs are not executed, so there is
	// nothing to record, and charging gas there made PrepareProposal reject
	// txs that CheckTx accepted (#648). Simulate must match Finalize so that
	// gas estimation (--gas auto) includes this handler; its writes are
	// discarded.
	if mode := ctx.ExecMode(); mode != sdk.ExecModeFinalize && mode != sdk.ExecModeSimulate {
		return next(ctx, tx, simulate, success)
	}

	msgs := tx.GetMsgs()
	dd.FilterMsgsAndProcessMsgs(ctx, msgs...)

	return next(ctx, tx, simulate, success)
}

func (dd DyncommDecorator) FilterMsgsAndProcessMsgs(ctx sdk.Context, msgs ...sdk.Msg) {
	for _, msg := range msgs {
		switch msg.(type) {
		case *stakingtypes.MsgEditValidator:
			dd.ProcessEditValidator(ctx, msg)
		case *stakingtypes.MsgCreateValidator:
			dd.ProcessCreateValidator(ctx, msg)
		default:
			continue
		}
	}
}

func (dd DyncommDecorator) ProcessEditValidator(ctx sdk.Context, msg sdk.Msg) {
	msgEditValidator := msg.(*stakingtypes.MsgEditValidator)

	// no update of CommissionRate provided
	if msgEditValidator.CommissionRate == nil {
		return
	}

	// post handler runs after successfully
	// calling runMsgs -> we can set state changes here!
	newIntendedRate := msgEditValidator.CommissionRate
	dd.dyncommKeeper.SetTargetCommissionRate(ctx, msgEditValidator.ValidatorAddress, *newIntendedRate)
}

func (dd DyncommDecorator) ProcessCreateValidator(ctx sdk.Context, msg sdk.Msg) {
	// post handler runs after successfully
	// calling runMsgs -> we can set state changes here!
	msgCreateValidator := msg.(*stakingtypes.MsgCreateValidator)
	newIntendedRate := msgCreateValidator.Commission.Rate
	dd.dyncommKeeper.SetTargetCommissionRate(ctx, msgCreateValidator.ValidatorAddress, newIntendedRate)
}
