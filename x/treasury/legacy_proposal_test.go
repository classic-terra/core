package treasury_test

import (
	"testing"

	apptesting "github.com/classic-terra/core/v4/app/testing"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/treasury/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/stretchr/testify/suite"
)

// LegacyProposalTestSuite covers the removal of the treasury burn tax
// exemption proposals in v14_5.
type LegacyProposalTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestLegacyProposalTestSuite(t *testing.T) {
	suite.Run(t, new(LegacyProposalTestSuite))
}

func (s *LegacyProposalTestSuite) SetupTest() {
	s.Setup(s.T(), core.ColumbusChainID)
	params := govv1.DefaultParams()
	params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000))
	params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 5_000_000))
	s.Require().NoError(s.App.GovKeeper.Params.Set(s.Ctx, params))
	s.FundAcc(s.TestAccs[0], sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 100_000_000)))
}

func (s *LegacyProposalTestSuite) legacyContents() []govv1beta1.Content {
	addrs := []string{s.TestAccs[1].String()}
	return []govv1beta1.Content{
		&types.AddBurnTaxExemptionAddressProposal{Title: "t", Description: "d", Addresses: addrs},
		&types.RemoveBurnTaxExemptionAddressProposal{Title: "t", Description: "d", Addresses: addrs},
	}
}

// TestSubmissionRejected: the proposals are no longer routed by x/gov.
func (s *LegacyProposalTestSuite) TestSubmissionRejected() {
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	msgServer := govkeeper.NewMsgServerImpl(&s.App.GovKeeper)

	s.Require().False(s.App.GovKeeper.LegacyRouter().HasRoute(types.RouterKey))

	for _, content := range s.legacyContents() {
		legacy, err := govv1.NewLegacyContent(content, govAuthority)
		s.Require().NoError(err)
		submit, err := govv1.NewMsgSubmitProposal(
			[]sdk.Msg{legacy},
			sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000)),
			s.TestAccs[0].String(), "", "t", "d", false,
		)
		s.Require().NoError(err)

		_, err = msgServer.SubmitProposal(s.Ctx, submit)
		s.Require().ErrorIs(err, govtypes.ErrNoProposalHandlerExists, "%T", content)
	}
}

// TestHistoricalProposalsStillDecode: proposals stored before the removal
// (e.g. mainnet #12175) must still unpack, otherwise gov proposal queries and
// genesis export would fail.
func (s *LegacyProposalTestSuite) TestHistoricalProposalsStillDecode() {
	cdc := s.App.AppCodec()
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	for _, content := range s.legacyContents() {
		legacy, err := govv1.NewLegacyContent(content, govAuthority)
		s.Require().NoError(err)
		msgAny, err := codectypes.NewAnyWithValue(legacy)
		s.Require().NoError(err)
		stored := govv1.Proposal{Id: 12175, Messages: []*codectypes.Any{msgAny}}

		bz, err := cdc.Marshal(&stored)
		s.Require().NoError(err)

		var decoded govv1.Proposal
		s.Require().NoError(cdc.Unmarshal(bz, &decoded))
		msgs, err := decoded.GetMsgs()
		s.Require().NoError(err)
		s.Require().Len(msgs, 1)

		got, err := govv1.LegacyContentFromMessage(msgs[0].(*govv1.MsgExecLegacyContent))
		s.Require().NoError(err)
		s.Require().Equal(content, got)

		// JSON (REST/CLI query output) must work as well
		_, err = cdc.MarshalJSON(&decoded)
		s.Require().NoError(err)
	}
}
