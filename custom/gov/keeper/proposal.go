package keeper

import (
	"errors"
	"fmt"

	v2lunc1types "github.com/classic-terra/core/v3/custom/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// SubmitProposal creates a new proposal given an array of messages
func (keeper Keeper) SubmitProposal(ctx sdk.Context, messages []sdk.Msg, metadata, title, summary string, proposer sdk.AccAddress) (v1.Proposal, error) {
	err := keeper.assertMetadataLength(metadata)

	if err != nil {
		return v1.Proposal{}, err
	}

	// assert summary is no longer than predefined max length of metadata
	err = keeper.assertMetadataLength(summary)
	if err != nil {
		return v1.Proposal{}, err
	}

	// assert title is no longer than predefined max length of metadata
	err = keeper.assertMetadataLength(title)
	if err != nil {
		return v1.Proposal{}, err
	}

	// Will hold a comma-separated string of all Msg type URLs.
	msgsStr := ""

	// Loop through all messages and confirm that each has a handler and the gov module account
	// as the only signer
	for _, msg := range messages {
		msgsStr += fmt.Sprintf(",%s", sdk.MsgTypeURL(msg))

		// perform a basic validation of the message
		if err := msg.ValidateBasic(); err != nil {
			return v1.Proposal{}, sdkerrors.Wrap(types.ErrInvalidProposalMsg, err.Error())
		}

		signers := msg.GetSigners()
		if len(signers) != 1 {
			return v1.Proposal{}, types.ErrInvalidSigner
		}

		// assert that the governance module account is the only signer of the messages
		if !signers[0].Equals(keeper.GetGovernanceAccount(ctx).GetAddress()) {
			return v1.Proposal{}, sdkerrors.Wrapf(types.ErrInvalidSigner, signers[0].String())
		}

		// use the msg service router to see that there is a valid route for that message.
		handler := keeper.router.Handler(msg)
		if handler == nil {
			return v1.Proposal{}, sdkerrors.Wrap(types.ErrUnroutableProposalMsg, sdk.MsgTypeURL(msg))
		}

		// Only if it's a MsgExecLegacyContent do we try to execute the
		// proposal in a cached context.
		// For other Msgs, we do not verify the proposal messages any further.
		// They may fail upon execution.
		// ref: https://github.com/cosmos/cosmos-sdk/pull/10868#discussion_r784872842
		if msg, ok := msg.(*v1.MsgExecLegacyContent); ok {
			cacheCtx, _ := ctx.CacheContext()
			if _, err := handler(cacheCtx, msg); err != nil {
				if errors.Is(types.ErrNoProposalHandlerExists, err) {
					return v1.Proposal{}, err
				}
				return v1.Proposal{}, sdkerrors.Wrap(types.ErrInvalidProposalContent, err.Error())
			}
		}

	}

	proposalID, err := keeper.GetProposalID(ctx)
	if err != nil {
		return v1.Proposal{}, err
	}

	submitTime := ctx.BlockHeader().Time
	depositPeriod := keeper.GetParams(ctx).MaxDepositPeriod

	proposal, err := v1.NewProposal(messages, proposalID, submitTime, submitTime.Add(*depositPeriod), metadata, title, summary, proposer)
	if err != nil {
		return v1.Proposal{}, err
	}

	keeper.SetProposal(ctx, proposal)
	keeper.InsertInactiveProposalQueue(ctx, proposalID, *proposal.DepositEndTime)
	keeper.SetProposalID(ctx, proposalID+1)

	totalLuncDeposit, err := keeper.GetMinimumDepositBaseUusd(ctx)
	if err != nil {
		return v1.Proposal{}, sdkerrors.Wrap(v2lunc1types.ErrQueryExchangeRateUusdFail, err.Error())
	}

	err = keeper.SetDepositLimitBaseUusd(ctx, proposalID, totalLuncDeposit)
	if err != nil {
		return v1.Proposal{}, sdkerrors.Wrap(v2lunc1types.ErrQueryExchangeRateUusdFail, err.Error())
	}

	// called right after a proposal is submitted
	keeper.Hooks().AfterProposalSubmission(ctx, proposalID)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSubmitProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(types.AttributeKeyProposalMessages, msgsStr),
		),
	)

	return proposal, nil
}
