package taxexemption_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/api/amino"
	sdkmath "cosmossdk.io/math"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/taxexemption/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtestutil "github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/suite"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// GovTestSuite exercises the taxexemption messages through the real x/gov
// pipeline (submit -> vote -> tally -> execute) on a fully wired TerraApp.
// It guards against regressions where the messages are routable on the
// MsgServiceRouter but cannot be carried by a governance proposal, e.g.
// because the cosmos.msg.v1.signer annotation is missing.
type GovTestSuite struct {
	apptesting.KeeperTestHelper

	govAuthority string
	voter        sdk.AccAddress
}

func TestGovTestSuite(t *testing.T) {
	suite.Run(t, new(GovTestSuite))
}

func (s *GovTestSuite) SetupTest() {
	s.Setup(s.T(), core.ColumbusChainID)
	s.govAuthority = authtypes.NewModuleAddress(govtypes.ModuleName).String()

	params := govv1.DefaultParams()
	params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000))
	params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 5_000_000))
	s.Require().NoError(s.App.GovKeeper.Params.Set(s.Ctx, params))

	// Create a single bonded validator that holds all voting power; its
	// operator account is the voter.
	bondAmt := sdkmath.NewInt(10_000_000)
	valAddr := sdk.ValAddress(s.TestAccs[2])
	validator := stakingtestutil.NewValidator(s.T(), valAddr, simtestutil.CreateTestPubKeys(1)[0])
	validator, _ = validator.AddTokensFromDel(bondAmt)
	// back the tokens with coins; TestingUpdateValidator moves them from the
	// not-bonded to the bonded pool so the quorum check sees them
	s.FundAcc(s.TestAccs[2], sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, bondAmt)))
	s.Require().NoError(s.App.BankKeeper.SendCoinsFromAccountToModule(
		s.Ctx, s.TestAccs[2], stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, bondAmt))))
	stakingkeeper.TestingUpdateValidator(s.App.StakingKeeper, s.Ctx, validator, true)
	s.voter = s.TestAccs[2]

	s.FundAcc(s.TestAccs[0], sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 100_000_000)))
}

// passProposal submits msgs as a gov v1 proposal, votes yes with all voting
// power, advances past the voting period and runs the gov EndBlocker. It
// asserts that the proposal passed and its messages executed successfully.
func (s *GovTestSuite) passProposal(msgs ...sdk.Msg) {
	s.T().Helper()

	proposer := s.TestAccs[0]
	msgServer := govkeeper.NewMsgServerImpl(&s.App.GovKeeper)

	submit, err := govv1.NewMsgSubmitProposal(
		msgs,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000)),
		proposer.String(), "", "tax exemption", "tax exemption change", false,
	)
	s.Require().NoError(err)

	res, err := msgServer.SubmitProposal(s.Ctx, submit)
	s.Require().NoError(err, "proposal submission must succeed")

	_, err = msgServer.Vote(s.Ctx, govv1.NewMsgVote(s.voter, res.ProposalId, govv1.OptionYes, ""))
	s.Require().NoError(err)

	proposal, err := s.App.GovKeeper.Proposals.Get(s.Ctx, res.ProposalId)
	s.Require().NoError(err)
	s.Require().Equal(govv1.StatusVotingPeriod, proposal.Status)

	s.Ctx = s.Ctx.WithBlockTime(proposal.VotingEndTime.Add(time.Second))
	s.Require().NoError(gov.EndBlocker(s.Ctx, &s.App.GovKeeper))

	proposal, err = s.App.GovKeeper.Proposals.Get(s.Ctx, res.ProposalId)
	s.Require().NoError(err)
	s.Require().Equal(govv1.StatusPassed, proposal.Status, "failed reason: %s", proposal.FailedReason)
}

func (s *GovTestSuite) allMsgs() []sdk.Msg {
	addr := s.TestAccs[1].String()
	return []sdk.Msg{
		&types.MsgAddTaxExemptionZone{Zone: "z", Addresses: []string{addr}, Authority: s.govAuthority},
		&types.MsgModifyTaxExemptionZone{Zone: "z", Authority: s.govAuthority},
		&types.MsgAddTaxExemptionAddress{Zone: "z", Addresses: []string{addr}, Authority: s.govAuthority},
		&types.MsgRemoveTaxExemptionAddress{Zone: "z", Addresses: []string{addr}, Authority: s.govAuthority},
		&types.MsgRemoveTaxExemptionZone{Zone: "z", Authority: s.govAuthority},
	}
}

// TestMsgSignersResolveToAuthority checks that the SDK signing context can
// derive the signer of every taxexemption message from its authority field.
func (s *GovTestSuite) TestMsgSignersResolveToAuthority() {
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	for _, msg := range s.allMsgs() {
		signers, _, err := s.App.AppCodec().GetMsgV1Signers(msg)
		s.Require().NoError(err, "%T", msg)
		s.Require().Len(signers, 1, "%T", msg)
		s.Require().Equal([]byte(govAddr), signers[0], "%T", msg)
	}
}

// TestZoneLifecycleViaGovernance manages a zone end-to-end through proposals
// and checks the effect on IsExemptedFromTax after each step.
func (s *GovTestSuite) TestZoneLifecycleViaGovernance() {
	k := s.App.TaxExemptionKeeper
	member := s.TestAccs[1].String()
	member2 := s.TestAccs[2].String()
	outsider := sdk.AccAddress([]byte("outsider_address____")).String()

	s.Require().False(k.IsExemptedFromTax(s.Ctx, member, outsider))

	// 1. add zone with outgoing exemption
	s.passProposal(&types.MsgAddTaxExemptionZone{
		Zone: "zone1", Outgoing: true, Addresses: []string{member}, Authority: s.govAuthority,
	})
	zone, err := k.GetTaxExemptionZone(s.Ctx, "zone1")
	s.Require().NoError(err)
	s.Require().True(zone.Outgoing)
	s.Require().True(k.IsExemptedFromTax(s.Ctx, member, outsider))
	s.Require().False(k.IsExemptedFromTax(s.Ctx, outsider, member))

	// 2. modify zone to incoming-only
	s.passProposal(&types.MsgModifyTaxExemptionZone{
		Zone: "zone1", Incoming: true, Authority: s.govAuthority,
	})
	s.Require().False(k.IsExemptedFromTax(s.Ctx, member, outsider))
	s.Require().True(k.IsExemptedFromTax(s.Ctx, outsider, member))

	// 3. add a second address, 4. remove it again (bundled in one proposal
	// to also cover multi-message proposals)
	s.passProposal(&types.MsgAddTaxExemptionAddress{
		Zone: "zone1", Addresses: []string{member2}, Authority: s.govAuthority,
	})
	s.Require().True(k.IsExemptedFromTax(s.Ctx, outsider, member2))

	s.passProposal(
		&types.MsgRemoveTaxExemptionAddress{Zone: "zone1", Addresses: []string{member2}, Authority: s.govAuthority},
		&types.MsgModifyTaxExemptionZone{Zone: "zone1", Incoming: true, Outgoing: true, Authority: s.govAuthority},
	)
	s.Require().False(k.IsExemptedFromTax(s.Ctx, outsider, member2))
	s.Require().True(k.IsExemptedFromTax(s.Ctx, member, outsider))

	// 5. remove zone
	s.passProposal(&types.MsgRemoveTaxExemptionZone{Zone: "zone1", Authority: s.govAuthority})
	_, err = k.GetTaxExemptionZone(s.Ctx, "zone1")
	s.Require().Error(err)
	s.Require().False(k.IsExemptedFromTax(s.Ctx, member, outsider))
	s.Require().False(k.IsExemptedFromTax(s.Ctx, outsider, member))
}

// TestProposalRejectsForeignAuthority makes sure a proposal cannot carry a
// taxexemption message whose authority is not the gov module account.
func (s *GovTestSuite) TestProposalRejectsForeignAuthority() {
	msgServer := govkeeper.NewMsgServerImpl(&s.App.GovKeeper)
	submit, err := govv1.NewMsgSubmitProposal(
		[]sdk.Msg{&types.MsgAddTaxExemptionZone{
			Zone: "evil", Addresses: []string{s.TestAccs[1].String()}, Authority: s.TestAccs[0].String(),
		}},
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1_000_000))),
		s.TestAccs[0].String(), "", "t", "s", false,
	)
	s.Require().NoError(err)

	_, err = msgServer.SubmitProposal(s.Ctx, submit)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}

// TestAminoNamesMatchLegacyCodec makes sure the amino.name proto option used
// by the x/tx LEGACY_AMINO_JSON sign mode matches the name registered on the
// legacy amino codec, so Ledger/amino-signed txs stay consistent.
func (s *GovTestSuite) TestAminoNamesMatchLegacyCodec() {
	for _, msg := range s.allMsgs() {
		fullName := protoreflect.FullName(sdk.MsgTypeURL(msg)[1:])
		desc, err := gogoproto.HybridResolver.FindDescriptorByName(fullName)
		s.Require().NoError(err, fullName)
		protoName, ok := protov2.GetExtension(desc.Options(), amino.E_Name).(string)
		s.Require().True(ok, fullName)

		bz, err := types.ModuleCdc.MarshalJSON(msg)
		s.Require().NoError(err)
		var wrapped struct {
			Type string `json:"type"`
		}
		s.Require().NoError(json.Unmarshal(bz, &wrapped))
		s.Require().Equal(wrapped.Type, protoName, fullName)
	}
}
