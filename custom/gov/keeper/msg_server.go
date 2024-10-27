package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	v2lunc1 "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

type msgServer struct {
	*Keeper
	v1MsgServer govv1.MsgServer
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) v2lunc1.MsgServer {
	return &msgServer{
		Keeper:      keeper,
		v1MsgServer: govkeeper.NewMsgServerImpl(keeper.Keeper),
	}
}

var _ v2lunc1.MsgServer = msgServer{}

// SubmitProposal implements the MsgServer.SubmitProposal method.
func (k msgServer) SubmitProposal(goCtx context.Context, msg *govv1.MsgSubmitProposal) (*govv1.MsgSubmitProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	initialDeposit := msg.GetInitialDeposit()

	if err := k.validateInitialDeposit(ctx, initialDeposit); err != nil {
		return nil, err
	}

	proposalMsgs, err := msg.GetMsgs()
	if err != nil {
		return nil, err
	}

	proposer, err := sdk.AccAddressFromBech32(msg.GetProposer())
	if err != nil {
		return nil, err
	}

	proposal, err := k.Keeper.SubmitProposal(ctx, proposalMsgs, msg.Metadata, msg.Title, msg.Summary, proposer)
	if err != nil {
		return nil, err
	}

	bytes, err := proposal.Marshal()
	if err != nil {
		return nil, err
	}

	// ref: https://github.com/cosmos/cosmos-sdk/issues/9683
	ctx.GasMeter().ConsumeGas(
		3*ctx.KVGasConfig().WriteCostPerByte*uint64(len(bytes)),
		"submit proposal",
	)

	votingStarted, err := k.Keeper.AddDeposit(ctx, proposal.Id, proposer, msg.GetInitialDeposit())
	if err != nil {
		return nil, err
	}

	if votingStarted {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(govtypes.EventTypeSubmitProposal,
				sdk.NewAttribute(govtypes.AttributeKeyVotingPeriodStart, fmt.Sprintf("%d", proposal.Id)),
			),
		)
	}

	return &govv1.MsgSubmitProposalResponse{
		ProposalId: proposal.Id,
	}, nil
}

// Deposit implements the MsgServer.Deposit method.
func (k msgServer) Deposit(goCtx context.Context, msg *govv1.MsgDeposit) (*govv1.MsgDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, err
	}
	votingStarted, err := k.Keeper.AddDeposit(ctx, msg.ProposalId, accAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	if votingStarted {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				govtypes.EventTypeProposalDeposit,
				sdk.NewAttribute(govtypes.AttributeKeyVotingPeriodStart, fmt.Sprintf("%d", msg.ProposalId)),
			),
		)
	}

	return &govv1.MsgDepositResponse{}, nil
}

func (k msgServer) ExecLegacyContent(goCtx context.Context, msg *govv1.MsgExecLegacyContent) (*govv1.MsgExecLegacyContentResponse, error) {
	return k.v1MsgServer.ExecLegacyContent(goCtx, msg)
}

func (k msgServer) Vote(goCtx context.Context, msg *govv1.MsgVote) (*govv1.MsgVoteResponse, error) {
	return k.v1MsgServer.Vote(goCtx, msg)
}

func (k msgServer) VoteWeighted(goCtx context.Context, msg *govv1.MsgVoteWeighted) (*govv1.MsgVoteWeightedResponse, error) {
	return k.v1MsgServer.VoteWeighted(goCtx, msg)
}

func (k msgServer) UpdateParams(goCtx context.Context, msg *v2lunc1.MsgUpdateParams) (*v2lunc1.MsgUpdateParamsResponse, error) {
	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &v2lunc1.MsgUpdateParamsResponse{}, nil
}

type legacyMsgServer struct {
	govAcct string
	server  v2lunc1.MsgServer
}

// NewLegacyMsgServerImpl returns an implementation of the v1beta1 legacy MsgServer interface. It wraps around
// the current MsgServer
func NewLegacyMsgServerImpl(govAcct string, v2lunc1Server v2lunc1.MsgServer) govv1beta1.MsgServer {
	return &legacyMsgServer{govAcct: govAcct, server: v2lunc1Server}
}

var _ govv1beta1.MsgServer = legacyMsgServer{}

func (k legacyMsgServer) SubmitProposal(goCtx context.Context, msg *govv1beta1.MsgSubmitProposal) (*govv1beta1.MsgSubmitProposalResponse, error) {
	contentMsg, err := govv1.NewLegacyContent(msg.GetContent(), k.govAcct)
	if err != nil {
		return nil, fmt.Errorf("error converting legacy content into proposal message: %w", err)
	}

	proposal, err := govv1.NewMsgSubmitProposal(
		[]sdk.Msg{contentMsg},
		msg.InitialDeposit,
		msg.Proposer,
		"",
		msg.GetContent().GetTitle(),
		msg.GetContent().GetDescription(),
	)
	if err != nil {
		return nil, err
	}

	resp, err := k.server.SubmitProposal(goCtx, proposal)
	if err != nil {
		return nil, err
	}

	return &govv1beta1.MsgSubmitProposalResponse{ProposalId: resp.ProposalId}, nil
}

func (k legacyMsgServer) Vote(goCtx context.Context, msg *govv1beta1.MsgVote) (*govv1beta1.MsgVoteResponse, error) {
	_, err := k.server.Vote(goCtx, &govv1.MsgVote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     govv1.VoteOption(msg.Option),
	})
	if err != nil {
		return nil, err
	}
	return &govv1beta1.MsgVoteResponse{}, nil
}

func (k legacyMsgServer) VoteWeighted(goCtx context.Context, msg *govv1beta1.MsgVoteWeighted) (*govv1beta1.MsgVoteWeightedResponse, error) {
	opts := make([]*govv1.WeightedVoteOption, len(msg.Options))
	for idx, opt := range msg.Options {
		opts[idx] = &govv1.WeightedVoteOption{
			Option: govv1.VoteOption(opt.Option),
			Weight: opt.Weight.String(),
		}
	}

	_, err := k.server.VoteWeighted(goCtx, &govv1.MsgVoteWeighted{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Options:    opts,
	})
	if err != nil {
		return nil, err
	}
	return &govv1beta1.MsgVoteWeightedResponse{}, nil
}

func (k legacyMsgServer) Deposit(goCtx context.Context, msg *govv1beta1.MsgDeposit) (*govv1beta1.MsgDepositResponse, error) {
	_, err := k.server.Deposit(goCtx, &govv1.MsgDeposit{
		ProposalId: msg.ProposalId,
		Depositor:  msg.Depositor,
		Amount:     msg.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &govv1beta1.MsgDepositResponse{}, nil
}
