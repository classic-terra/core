package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	v2luncv1types "github.com/classic-terra/core/v3/custom/gov/types"
	core "github.com/classic-terra/core/v3/types"

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
		if !signers[0].Equals(keeper.baseKeeper.GetGovernanceAccount(ctx).GetAddress()) {
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

	proposalID, err := keeper.baseKeeper.GetProposalID(ctx)
	if err != nil {
		return v1.Proposal{}, err
	}

	submitTime := ctx.BlockHeader().Time
	depositPeriod := keeper.baseKeeper.GetParams(ctx).MaxDepositPeriod

	proposal, err := v1.NewProposal(messages, proposalID, submitTime, submitTime.Add(*depositPeriod), metadata, title, summary, proposer)
	if err != nil {
		return v1.Proposal{}, err
	}

	keeper.baseKeeper.SetProposal(ctx, proposal)
	keeper.baseKeeper.InsertInactiveProposalQueue(ctx, proposalID, *proposal.DepositEndTime)
	keeper.baseKeeper.SetProposalID(ctx, proposalID+1)

	// Get exchange rate betweent Lunc/uusd from oracle
	// save it to store
	price, err := keeper.oracleKeeper.GetLunaExchangeRate(ctx, core.MicroUSDDenom)
	if err != nil {
		return v1.Proposal{}, sdkerrors.Wrap(v2luncv1types.ErrQueryExchangeRateUusdFail, err.Error())
	}
	minUusdDeposit := keeper.GetParams(ctx).MinUusdDeposit
	totalLuncDeposit := sdk.NewDecFromInt(minUusdDeposit.Amount).Quo(price).TruncateInt()

	if err != nil {
		return v1.Proposal{}, sdkerrors.Wrap(v2luncv1types.ErrQueryExchangeRateUusdFail, err.Error())
	}
	keeper.SetPriceLuncBaseUusd(ctx, proposalID, math.LegacyDec(totalLuncDeposit))

	// called right after a proposal is submitted
	keeper.baseKeeper.Hooks().AfterProposalSubmission(ctx, proposalID)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSubmitProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(types.AttributeKeyProposalMessages, msgsStr),
		),
	)

	return proposal, nil
}

// SetPriceLuncBaseUusd sets a price Lunc base on Uusd to store.
func (keeper Keeper) SetPriceLuncBaseUusd(ctx sdk.Context, proposalID uint64, amount sdk.Dec) {
	store := ctx.KVStore(keeper.storeKey)
	key := v2luncv1types.TotalDepositKey(proposalID)

	bz, err := amount.Marshal()
	if err == nil {
		store.Set(key, bz)
	}
}

// GetDepositLimitBaseUUSD gets the deposit limit (Lunc) for a specific proposal
func (keeper Keeper) GetDepositLimitBaseUusd(ctx sdk.Context, proposalID uint64) (depositLimit sdk.Dec, err error) {
	store := ctx.KVStore(keeper.storeKey)
	key := v2luncv1types.TotalDepositKey(proposalID)
	bz := store.Get(key)
	if bz == nil {
		return sdk.ZeroDec(), err
	}
	err = depositLimit.Unmarshal(bz)
	if err == nil {
		return sdk.ZeroDec(), err
	}

	return depositLimit, nil
}
