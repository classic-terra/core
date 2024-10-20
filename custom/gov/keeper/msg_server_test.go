package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/stretchr/testify/require"

	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v2lunc1types "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	core "github.com/classic-terra/core/v3/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	baseDepositTestAmount = 100
)

var ()

func TestSubmitProposalReq(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr
	govMsgSvr := NewMsgServerImpl(input.GovKeeper)
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	initialDeposit := coins
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	cases := map[string]struct {
		preRun    func() (*v1.MsgSubmitProposal, error)
		expErr    bool
		expErrMsg string
	}{
		"metadata too long": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					initialDeposit,
					proposer.String(),
					strings.Repeat("1", 300),
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"many signers": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(govAcct, addr)},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "expected gov account as only signer for proposal message",
		},
		"signer isn't gov account": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(addr)},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "expected gov account as only signer for proposal message",
		},
		"invalid msg handler": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(govAcct)},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "proposal message not recognized by router",
		},
		"all good": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr: false,
		},
		"all good with min deposit": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Fund the account
			FundAccount(input, addr, coins)
			msg, err := tc.preRun()
			if err != nil {
				t.Fatalf("preRun error: %v", err)
			}
			res, err := govMsgSvr.SubmitProposal(ctx, msg)
			if tc.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.expErrMsg) {
					t.Errorf("expected error message to contain %q but got %q", tc.expErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
				if res.ProposalId == 0 {
					t.Errorf("expected non-nil ProposalId but got %v", res.ProposalId)
				}
			}
		})
	}
}

func TestVoteReq(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	govKeeper := input.GovKeeper
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr
	govMsgSvr := NewMsgServerImpl(input.GovKeeper)
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		coins,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)
	// Fund the account
	FundAccount(input, addr, coins)
	require.NoError(t, err)
	res, err := govMsgSvr.SubmitProposal(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res.ProposalId)
	proposalId := res.ProposalId
	requiredAmount := govKeeper.GetDepositLimitBaseUusd(ctx, proposalId).TruncateInt()

	cases := map[string]struct {
		preRun    func() uint64
		expErr    bool
		expErrMsg string
		option    v1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)
				FundAccount(input, addr, coins)
				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				return res.ProposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"metadata too long": {
			preRun: func() uint64 {
				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, proposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  strings.Repeat("a", 300),
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, requiredAmount)),
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)
				FundAccount(input, addr, coins)
				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)

				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, res.ProposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return res.ProposalId
			},
			option:   v1.VoteOption_VOTE_OPTION_YES,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			pId := tc.preRun()
			FundAccount(input, addr, coins)
			voteReq := v1.NewMsgVote(tc.voter, pId, tc.option, tc.metadata)
			_, err := govMsgSvr.Vote(ctx, voteReq)
			if tc.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.expErrMsg) {
					t.Errorf("expected error message to contain %q but got %q", tc.expErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
				if res.ProposalId == 0 {
					t.Errorf("expected non-nil ProposalId but got %v", res.ProposalId)
				}
			}
		})
	}
}

func TestVoteWeightedReq(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	govKeeper := input.GovKeeper
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr
	govMsgSvr := NewMsgServerImpl(input.GovKeeper)
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		coins,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)
	FundAccount(input, addr, coins)
	res, err := govMsgSvr.SubmitProposal(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res.ProposalId)
	proposalId := res.ProposalId
	// requiredAmount := suite.govKeeper.GetDepositLimitBaseUusd(suite.ctx, proposalId).TruncateInt()

	cases := map[string]struct {
		preRun    func() uint64
		vote      *v1.MsgVote
		expErr    bool
		expErrMsg string
		option    v1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)
				FundAccount(input, addr, coins)
				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				return res.ProposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"metadata too long": {
			preRun: func() uint64 {
				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, proposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  strings.Repeat("a", 300),
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)
				FundAccount(input, addr, coins)
				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, res.ProposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return res.ProposalId
			},
			option:   v1.VoteOption_VOTE_OPTION_YES,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			pId := tc.preRun()
			voteReq := v1.NewMsgVoteWeighted(tc.voter, pId, v1.NewNonSplitVoteOption(tc.option), tc.metadata)
			_, err := govMsgSvr.VoteWeighted(ctx, voteReq)
			if tc.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.expErrMsg) {
					t.Errorf("expected error message to contain %q but got %q", tc.expErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
				if res.ProposalId == 0 {
					t.Errorf("expected non-nil ProposalId but got %v", res.ProposalId)
				}
			}
		})
	}
}

func TestDepositReq(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	govMsgSvr := NewMsgServerImpl(input.GovKeeper)

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		coins,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)
	FundAccount(input, addr, coins)
	res, err := govMsgSvr.SubmitProposal(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res.ProposalId)
	pId := res.ProposalId

	cases := map[string]struct {
		preRun     func() uint64
		expErr     bool
		proposalId uint64
		depositor  sdk.AccAddress
		deposit    sdk.Coins
		options    v1.WeightedVoteOptions
	}{
		"wrong proposal id": {
			preRun: func() uint64 {
				return 0
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    true,
			options:   v1.NewNonSplitVoteOption(v1.OptionYes),
		},
		"all good": {
			preRun: func() uint64 {
				return pId
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    false,
			options:   v1.NewNonSplitVoteOption(v1.OptionYes),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			proposalId := tc.preRun()
			FundAccount(input, addr, coins)
			depositReq := v1.NewMsgDeposit(tc.depositor, proposalId, tc.deposit)
			_, err := govMsgSvr.Deposit(ctx, depositReq)
			if tc.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
				if res.ProposalId == 0 {
					t.Errorf("expected non-nil ProposalId but got %v", res.ProposalId)
				}
			}
		})
	}
}

func TestMsgUpdateParams(t *testing.T) {
	input := CreateTestInput(t)
	ctx := input.Ctx
	govKeeper := input.GovKeeper
	authority := govKeeper.GetAuthority()
	params := v2lunc1types.DefaultParams()
	govMsgSvr := NewMsgServerImpl(input.GovKeeper)
	testCases := []struct {
		name      string
		input     func() *v2lunc1types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid",
			input: func() *v2lunc1types.MsgUpdateParams {
				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params,
				}
			},
			expErr: false,
		},
		{
			name: "invalid authority",
			input: func() *v2lunc1types.MsgUpdateParams {
				return &v2lunc1types.MsgUpdateParams{
					Authority: "authority",
					Params:    params,
				}
			},
			expErr:    true,
			expErrMsg: "invalid authority address",
		},
		{
			name: "invalid min deposit",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.MinDeposit = nil

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid minimum deposit",
		},
		{
			name: "negative deposit",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.MinDeposit = sdk.Coins{{
					Denom:  sdk.DefaultBondDenom,
					Amount: sdk.NewInt(-100),
				}}

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid minimum deposit",
		},
		{
			name: "invalid max deposit period",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.MaxDepositPeriod = nil

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "maximum deposit period must not be nil",
		},
		{
			name: "zero max deposit period",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				duration := time.Duration(0)
				params1.MaxDepositPeriod = &duration

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "maximum deposit period must be positive",
		},
		{
			name: "invalid quorum",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Quorum = "abc"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid quorum string",
		},
		{
			name: "negative quorum",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Quorum = "-0.1"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorom cannot be negative",
		},
		{
			name: "quorum > 1",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Quorum = "2"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorom too large",
		},
		{
			name: "invalid threshold",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Threshold = "abc"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid threshold string",
		},
		{
			name: "negative threshold",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Threshold = "-0.1"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "vote threshold must be positive",
		},
		{
			name: "threshold > 1",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.Threshold = "2"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "vote threshold too large",
		},
		{
			name: "invalid veto threshold",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.VetoThreshold = "abc"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid vetoThreshold string",
		},
		{
			name: "negative veto threshold",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.VetoThreshold = "-0.1"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "veto threshold must be positive",
		},
		{
			name: "veto threshold > 1",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.VetoThreshold = "2"

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "veto threshold too large",
		},
		{
			name: "invalid voting period",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				params1.VotingPeriod = nil

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "voting period must not be nil",
		},
		{
			name: "zero voting period",
			input: func() *v2lunc1types.MsgUpdateParams {
				params1 := params
				duration := time.Duration(0)
				params1.VotingPeriod = &duration

				return &v2lunc1types.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "voting period must be positive",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.input()
			exec := func(updateParams *v2lunc1types.MsgUpdateParams) error {
				if err := msg.ValidateBasic(); err != nil {
					return err
				}

				if _, err := govMsgSvr.UpdateParams(ctx, updateParams); err != nil {
					return err
				}
				return nil
			}

			err := exec(msg)
			if tc.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.expErrMsg) {
					t.Errorf("expected error message to contain %q but got %q", tc.expErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
			}
		})
	}
}

func TestSubmitProposal_InitialDeposit(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govKeeper := input.GovKeeper
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.NewDecWithPrec(2, 1))
	_, _, addr := testdata.KeyTestPubAddr()

	// setup desposit value when test
	const meetsDepositValue = baseDepositTestAmount * baseDepositTestPercent / 100
	baseDepositRatioDec := sdk.NewDec(baseDepositTestPercent).Quo(sdk.NewDec(100))

	testcases := map[string]struct {
		minDeposit             sdk.Coins
		minInitialDepositRatio sdk.Dec
		initialDeposit         sdk.Coins
		accountBalance         sdk.Coins

		expectError bool
	}{
		"meets initial deposit, enough balance - success": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
		},
		"does not meet initial deposit, enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),

			expectError: true,
		},
		"meets initial deposit, not enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),

			expectError: true,
		},
		"does not meet initial deposit and not enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),

			expectError: true,
		},
	}

	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	msgs := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))),
		legacyProposalMsg,
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			// Fund the proposer's account
			FundAccount(input, addr, tc.minDeposit)
			govMsgSvr := NewMsgServerImpl(input.GovKeeper)

			params := v2lunc1types.DefaultParams()
			params.MinUusdDeposit = tc.minDeposit[0]
			params.MinInitialDepositRatio = tc.minInitialDepositRatio.String()
			govKeeper.SetParams(ctx, params)

			msg, err := v1.NewMsgSubmitProposal(msgs, tc.initialDeposit, addr.String(), "test", "Proposal", "description of proposal")
			require.NoError(t, err)

			// System under test
			_, err = govMsgSvr.SubmitProposal(sdk.WrapSDKContext(ctx), msg)

			// Assertions
			if tc.expectError {
				require.NoError(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// legacy msg server tests
func TestLegacyMsgSubmitProposal(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 UUSD
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	initialDeposit := coins

	cases := map[string]struct {
		preRun func() (*v1beta1.MsgSubmitProposal, error)
		expErr bool
	}{
		"all good": {
			preRun: func() (*v1beta1.MsgSubmitProposal, error) {
				return v1beta1.NewMsgSubmitProposal(
					v1beta1.NewTextProposal("test", "I am test"),
					initialDeposit,
					proposer,
				)
			},
			expErr: false,
		},
		"all good with min deposit": {
			preRun: func() (*v1beta1.MsgSubmitProposal, error) {
				return v1beta1.NewMsgSubmitProposal(
					v1beta1.NewTextProposal("test", "I am test"),
					coins,
					proposer,
				)
			},
			expErr: false,
		},
	}

	legacyMsgSrvr := NewLegacyMsgServerImpl(govAcct.String(), NewMsgServerImpl(input.GovKeeper))

	for name, c := range cases {

		t.Run(name, func(t *testing.T) {
			msg, err := c.preRun()

			if err != nil {
				t.Fatalf("preRun error: %v", err)
			}

			FundAccount(input, addr, coins)
			res, err := legacyMsgSrvr.SubmitProposal(ctx, msg)

			if c.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("SubmitProposal error: %v", err)
				}
				if res.ProposalId == 0 {
					t.Errorf("expected non-nil ProposalId but got %v", res.ProposalId)
				}
			}
		})
	}
}

func TestLegacyMsgVote(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govKeeper := input.GovKeeper
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 UUSD
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	FundAccount(input, addr, coins)

	// Create a proposal first
	proposal, err := input.GovKeeper.SubmitProposal(ctx, []sdk.Msg{}, "", "Test Proposal", "This is a test proposal", proposer)
	require.NoError(t, err)

	if err != nil {
		t.Fatalf("preRun error: %v", err)
	}
	proposalID := proposal.Id

	proposal, ok := input.GovKeeper.GetProposal(ctx, proposal.Id)
	require.True(t, ok)

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	govMsgSvr := NewMsgServerImpl(input.GovKeeper)

	cases := map[string]struct {
		preRun    func() uint64
		expErr    bool
		expErrMsg string
		option    v1beta1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)

				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				return res.ProposalId
			},
			option:    v1beta1.OptionYes,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalID
			},
			option:    v1beta1.OptionYes,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)
				FundAccount(input, addr, coins)
				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, res.ProposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return res.ProposalId
			},
			option:   v1beta1.OptionYes,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	legacyMsgSrvr := NewLegacyMsgServerImpl(govAcct.String(), govMsgSvr)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pId := tc.preRun()
			voteReq := v1beta1.NewMsgVote(tc.voter, pId, tc.option)
			_, err := legacyMsgSrvr.Vote(ctx, voteReq)
			proposal.Status = v1.StatusVotingPeriod
			input.GovKeeper.SetProposal(ctx, proposal)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLegacyVoteWeighted(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	govKeeper := input.GovKeeper
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	FundAccount(input, addr, coins)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, []sdk.Msg{}, "", "Test Proposal", "This is a test proposal", proposer)

	if err != nil {
		t.Fatalf("preRun error: %v", err)
	}
	proposalID := proposal.Id

	proposal, ok := input.GovKeeper.GetProposal(ctx, proposal.Id)
	require.True(t, ok)

	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	govMsgSvr := NewMsgServerImpl(input.GovKeeper)

	cases := map[string]struct {
		preRun    func() uint64
		vote      *v1beta1.MsgVote
		expErr    bool
		expErrMsg string
		option    v1beta1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)

				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				return res.ProposalId
			},
			option:    v1beta1.OptionYes,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalID
			},
			option:    v1beta1.OptionYes,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				require.NoError(t, err)

				res, err := govMsgSvr.SubmitProposal(ctx, msg)
				require.NoError(t, err)
				require.NotNil(t, res.ProposalId)
				// set proposal to status activedVoting
				proposal, ok := govKeeper.GetProposal(ctx, res.ProposalId)
				require.True(t, ok)
				proposal.Status = v1.StatusVotingPeriod
				govKeeper.SetProposal(ctx, proposal)
				return res.ProposalId
			},
			option:   v1beta1.OptionYes,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	legacyMsgSrvr := NewLegacyMsgServerImpl(govAcct.String(), govMsgSvr)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pId := tc.preRun()

			voteReq := v1beta1.NewMsgVoteWeighted(tc.voter, pId, v1beta1.NewNonSplitVoteOption(v1beta1.VoteOption(tc.option)))
			proposal.Status = v1.StatusVotingPeriod
			input.GovKeeper.SetProposal(ctx, proposal)
			_, err := legacyMsgSrvr.VoteWeighted(ctx, voteReq)
			FundAccount(input, addr, coins)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLegacyMsgDeposit(t *testing.T) {
	// Set up the necessary dependencies and context
	input := CreateTestInput(t)
	ctx := input.Ctx
	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()
	proposer := addr
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500000000))) //  500 Default Bond Denom
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	FundAccount(input, addr, coins)

	govMsgSvr := NewMsgServerImpl(input.GovKeeper)
	proposal, err := input.GovKeeper.SubmitProposal(ctx, []sdk.Msg{}, "", "Test Proposal", "This is a test proposal", proposer)
	require.NoError(t, err)
	proposalID := proposal.Id

	cases := map[string]struct {
		preRun     func() uint64
		expErr     bool
		proposalId uint64
		depositor  sdk.AccAddress
		deposit    sdk.Coins
		options    v1beta1.WeightedVoteOptions
	}{
		"wrong proposal id": {
			preRun: func() uint64 {
				return 0
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    true,
			options:   v1beta1.NewNonSplitVoteOption(v1beta1.OptionYes),
		},
		"all good": {
			preRun: func() uint64 {
				return proposalID
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    false,
			options:   v1beta1.NewNonSplitVoteOption(v1beta1.OptionYes),
		},
	}

	legacyMsgSrvr := NewLegacyMsgServerImpl(govAcct.String(), govMsgSvr)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			proposalId := tc.preRun()
			depositReq := v1beta1.NewMsgDeposit(tc.depositor, proposalId, tc.deposit)
			_, err := legacyMsgSrvr.Deposit(ctx, depositReq)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
