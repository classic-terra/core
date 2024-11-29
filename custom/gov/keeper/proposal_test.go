package keeper

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	core "github.com/classic-terra/core/v3/types"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func TestGetSetProposal(t *testing.T) {
	input := CreateTestInput(t)
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()

	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	tp := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))),
		legacyProposalMsg,
	}
	govKeeper := input.GovKeeper
	oracleKeeper := input.OracleKeeper
	ctx := input.Ctx

	lunaPriceInUSD := sdk.MustNewDecFromStr("0.10008905")
	oracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, lunaPriceInUSD)

	totalLuncMinDeposit, err := govKeeper.GetMinimumDepositBaseUstc(ctx)
	require.NoError(t, err)

	proposal, err := govKeeper.SubmitProposal(ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	require.NoError(t, err)
	proposalID := proposal.Id
	govKeeper.SetProposal(ctx, proposal)

	gotProposal, ok := govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.Equal(t, proposal, gotProposal)

	// Get min luna amount by uusd
	minLunaAmount := govKeeper.GetDepositLimitBaseUstc(ctx, proposalID)
	fmt.Printf("minLunaAmount %s\n", minLunaAmount)
	require.Equal(t, totalLuncMinDeposit, minLunaAmount)
}

func TestDeleteProposal(t *testing.T) {
	input := CreateTestInput(t)
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()

	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	tp := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))),
		legacyProposalMsg,
	}

	govKeeper := input.GovKeeper
	oracleKeeper := input.OracleKeeper
	ctx := input.Ctx

	oracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	// delete non-existing proposal
	require.PanicsWithValue(t, fmt.Sprintf("couldn't find proposal with id#%d", 10),
		func() {
			govKeeper.DeleteProposal(ctx, 10)
		},
	)
	proposal, err := govKeeper.SubmitProposal(ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	require.NoError(t, err)
	proposalID := proposal.Id
	govKeeper.SetProposal(ctx, proposal)
	require.NotPanics(t, func() {
		govKeeper.DeleteProposal(ctx, proposalID)
	}, "")
}

func TestActivateVotingPeriod(t *testing.T) {
	input := CreateTestInput(t)
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()

	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	tp := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))),
		legacyProposalMsg,
	}

	govKeeper := input.GovKeeper
	oracleKeeper := input.OracleKeeper
	ctx := input.Ctx

	oracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	proposal, err := govKeeper.SubmitProposal(ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	require.NoError(t, err)

	require.Nil(t, proposal.VotingStartTime)

	govKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, ok := govKeeper.GetProposal(ctx, proposal.Id)
	require.True(t, ok)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockHeader().Time))

	activeIterator := govKeeper.ActiveProposalQueueIterator(ctx, *proposal.VotingEndTime)
	require.True(t, activeIterator.Valid())

	proposalID := types.GetProposalIDFromBytes(activeIterator.Value())
	require.Equal(t, proposalID, proposal.Id)
	activeIterator.Close()

	// delete the proposal to avoid issues with other tests
	require.NotPanics(t, func() {
		govKeeper.DeleteProposal(ctx, proposalID)
	}, "")
}

func TestDeleteProposalInVotingPeriod(t *testing.T) {
	input := CreateTestInput(t)
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()

	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	tp := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))),
		legacyProposalMsg,
	}

	govKeeper := input.GovKeeper
	oracleKeeper := input.OracleKeeper
	ctx := input.Ctx

	oracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	proposal, err := govKeeper.SubmitProposal(ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	require.NoError(t, err)
	require.Nil(t, proposal.VotingStartTime)

	govKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, ok := govKeeper.GetProposal(ctx, proposal.Id)
	require.True(t, ok)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockHeader().Time))

	activeIterator := govKeeper.ActiveProposalQueueIterator(ctx, *proposal.VotingEndTime)
	require.True(t, activeIterator.Valid())

	proposalID := types.GetProposalIDFromBytes(activeIterator.Value())
	require.Equal(t, proposalID, proposal.Id)
	activeIterator.Close()

	// add vote
	voteOptions := []*v1.WeightedVoteOption{{Option: v1.OptionYes, Weight: "1.0"}}
	err = govKeeper.AddVote(ctx, proposal.Id, sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"), voteOptions, "")
	require.NoError(t, err)

	require.NotPanics(t, func() {
		govKeeper.DeleteProposal(ctx, proposalID)
	}, "")

	// add vote but proposal is deleted along with its VotingPeriodProposalKey
	err = govKeeper.AddVote(ctx, proposal.Id, sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"), voteOptions, "")
	require.ErrorContains(t, err, ": inactive proposal")
}

type invalidProposalRoute struct{ v1beta1.TextProposal }

func (invalidProposalRoute) ProposalRoute() string { return "nonexistingroute" }

func TestSubmitProposal(t *testing.T) {
	input := CreateTestInput(t)
	govKeeper := input.GovKeeper
	ctx := input.Ctx

	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, sdk.OneDec())

	tp := v1beta1.TextProposal{Title: "title", Description: "description"}
	govAcct := govKeeper.GetGovernanceAccount(ctx).GetAddress().String()
	_, _, randomAddr := testdata.KeyTestPubAddr()

	testCases := []struct {
		content     v1beta1.Content
		authority   string
		metadata    string
		expectedErr error
	}{
		{&tp, govAcct, "", nil},
		// Keeper does not check the validity of title and description, no error
		{&v1beta1.TextProposal{Title: "", Description: "description"}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: strings.Repeat("1234567890", 100), Description: "description"}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: "title", Description: ""}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: "title", Description: strings.Repeat("1234567890", 1000)}, govAcct, "", nil},
		// error when metadata is too long (>10000)
		{&tp, govAcct, strings.Repeat("a", 100001), types.ErrMetadataTooLong},
		// error when signer is not gov acct
		{&tp, randomAddr.String(), "", types.ErrInvalidSigner},
		// error only when invalid route
		{&invalidProposalRoute{}, govAcct, "", types.ErrNoProposalHandlerExists},
	}

	for i, tc := range testCases {
		prop, err := v1.NewLegacyContent(tc.content, tc.authority)
		require.NoError(t, err)
		_, err = govKeeper.SubmitProposal(ctx, []sdk.Msg{prop}, tc.metadata, "title", "", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
		require.True(t, errors.Is(tc.expectedErr, err), "tc #%d; got: %v, expected: %v", i, err, tc.expectedErr)
	}
}

func TestMigrateProposalMessages(t *testing.T) {
	content := v1beta1.NewTextProposal("Test", "description")
	contentMsg, err := v1.NewLegacyContent(content, sdk.AccAddress("test1").String())
	require.NoError(t, err)
	content, err = v1.LegacyContentFromMessage(contentMsg)
	require.NoError(t, err)
	require.Equal(t, "Test", content.GetTitle())
	require.Equal(t, "description", content.GetDescription())
}
