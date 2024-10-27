package keeper

import (
	"fmt"
	"testing"

	"github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	core "github.com/classic-terra/core/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	baseUSDDepositTestAmount = 100
	baseDepositTestPercent   = 25
)

func TestAddDeposits(t *testing.T) {
	input := CreateTestInput(t)
	bankKeeper := input.BankKeeper
	govKeeper := input.GovKeeper
	oracleKeeper := input.OracleKeeper
	ctx := input.Ctx

	oracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.OneDec())
	lunaCoin := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10_000_000_000)))

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	err := FundAccount(input, addr1, lunaCoin)
	require.NoError(t, err)
	err1 := FundAccount(input, addr2, lunaCoin)
	require.NoError(t, err1)

	addr1Initial := bankKeeper.GetAllBalances(ctx, addr1)
	addr2Initial := bankKeeper.GetAllBalances(ctx, addr2)

	govAcct := authtypes.NewModuleAddress(types.ModuleName)
	_, _, addr := testdata.KeyTestPubAddr()

	legacyProposalMsg, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Title", "description"), authtypes.NewModuleAddress(types.ModuleName).String())
	if err != nil {
		panic(err)
	}

	tp := []sdk.Msg{
		banktypes.NewMsgSend(govAcct, addr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1000)))),
		legacyProposalMsg,
	}
	proposal, err := govKeeper.SubmitProposal(ctx, tp, "", "title", "description", addr1)
	require.NoError(t, err)
	proposalID := proposal.Id

	amountDeposit1 := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100_000_000)))
	amountDeposit2 := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(500_000_000)))

	require.True(t, sdk.NewCoins(proposal.TotalDeposit...).IsEqual(sdk.NewCoins()))

	// Check no deposits at beginning
	deposit, found := govKeeper.GetDeposit(ctx, proposalID, addr1)
	require.False(t, found)
	proposal, ok := govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.Nil(t, proposal.VotingStartTime)

	// Check first deposit
	votingStarted, err := govKeeper.AddDeposit(ctx, proposalID, addr1, amountDeposit1)
	require.NoError(t, err)
	require.False(t, votingStarted)
	deposit, found = govKeeper.GetDeposit(ctx, proposalID, addr1)
	require.True(t, found)
	require.Equal(t, amountDeposit1, sdk.NewCoins(deposit.Amount...))
	require.Equal(t, addr1.String(), deposit.Depositor)
	proposal, ok = govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.Equal(t, amountDeposit1, sdk.NewCoins(proposal.TotalDeposit...))
	require.Equal(t, addr1Initial.Sub(amountDeposit1...), bankKeeper.GetAllBalances(ctx, addr1))

	// Check a second deposit from same address
	votingStarted, err = govKeeper.AddDeposit(ctx, proposalID, addr1, amountDeposit2)
	require.NoError(t, err)
	require.True(t, votingStarted)
	deposit, found = govKeeper.GetDeposit(ctx, proposalID, addr1)
	require.True(t, found)
	require.Equal(t, amountDeposit1.Add(amountDeposit2...), sdk.NewCoins(deposit.Amount...))
	require.Equal(t, addr1.String(), deposit.Depositor)
	proposal, ok = govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.Equal(t, amountDeposit1.Add(amountDeposit2...), sdk.NewCoins(proposal.TotalDeposit...))
	require.Equal(t, addr1Initial.Sub(amountDeposit1...).Sub(amountDeposit2...), bankKeeper.GetAllBalances(ctx, addr1))

	// Check third deposit from a new address
	votingStarted, err = govKeeper.AddDeposit(ctx, proposalID, addr2, amountDeposit1)
	require.NoError(t, err)
	require.False(t, votingStarted)
	deposit, found = govKeeper.GetDeposit(ctx, proposalID, addr2)
	require.True(t, found)
	require.Equal(t, addr2.String(), deposit.Depositor)
	require.Equal(t, amountDeposit1, sdk.NewCoins(deposit.Amount...))
	proposal, ok = govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.Equal(t, amountDeposit1.Add(amountDeposit2...).Add(amountDeposit1...), sdk.NewCoins(proposal.TotalDeposit...))
	require.Equal(t, addr2Initial.Sub(amountDeposit1...), bankKeeper.GetAllBalances(ctx, addr2))

	// Check that proposal moved to voting period
	proposal, ok = govKeeper.GetProposal(ctx, proposalID)
	require.True(t, ok)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockHeader().Time))

	// Test deposit iterator
	// NOTE order of deposits is determined by the addresses
	deposits := govKeeper.GetAllDeposits(ctx)
	fmt.Printf("deposits: %v\n", deposits)
	require.Len(t, deposits, 2)
	require.Equal(t, deposits, govKeeper.GetDeposits(ctx, proposalID))
	require.Equal(t, addr1.String(), deposits[0].Depositor)
	require.Equal(t, amountDeposit1.Add(amountDeposit2...), sdk.NewCoins(deposits[0].Amount...))
	require.Equal(t, addr2.String(), deposits[1].Depositor)
	require.Equal(t, amountDeposit1, sdk.NewCoins(deposits[1].Amount...))

	// Test Refund Deposits
	deposit, found = govKeeper.GetDeposit(ctx, proposalID, addr2)
	require.True(t, found)
	require.Equal(t, amountDeposit1, sdk.NewCoins(deposit.Amount...))
	govKeeper.RefundAndDeleteDeposits(ctx, proposalID)
	deposit, found = govKeeper.GetDeposit(ctx, proposalID, addr2)
	require.False(t, found)
	require.Equal(t, addr1Initial, bankKeeper.GetAllBalances(ctx, addr1))
	require.Equal(t, addr2Initial, bankKeeper.GetAllBalances(ctx, addr2))

	// Test delete and burn deposits
	proposal, err = govKeeper.SubmitProposal(ctx, tp, "", "title", "description", addr1)
	require.NoError(t, err)
	proposalID = proposal.Id
	_, err = govKeeper.AddDeposit(ctx, proposalID, addr1, amountDeposit1)
	require.NoError(t, err)
	govKeeper.DeleteAndBurnDeposits(ctx, proposalID)
	deposits = govKeeper.GetDeposits(ctx, proposalID)
	require.Len(t, deposits, 0)
	require.Equal(t, addr1Initial.Sub(amountDeposit1...), bankKeeper.GetAllBalances(ctx, addr1))
}

func TestValidateInitialDeposit(t *testing.T) {
	input := CreateTestInput(t)
	govKeeper := input.GovKeeper
	ctx := input.Ctx
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, sdk.NewDec(1))
	minLuncDeposit, err := input.GovKeeper.GetMinimumDepositBaseUusd(ctx)
	require.NoError(t, err)

	// setup deposit value when test
	meetsDepositValue := minLuncDeposit.Mul(sdk.NewInt(baseDepositTestPercent)).Quo(sdk.NewInt(100))

	testcases := map[string]struct {
		minDeposit               sdk.Coins
		minInitialDepositPercent int64
		initialDeposit           sdk.Coins

		expectError bool
	}{
		"min deposit * initial percent == initial deposit: success": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, meetsDepositValue)),
		},
		"min deposit * initial percent < initial deposit: success": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, meetsDepositValue.Add(sdk.NewInt(1)))),
		},
		"min deposit * initial percent > initial deposit: error": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, meetsDepositValue.Sub(sdk.NewInt(1)))),

			expectError: true,
		},
		"min deposit * initial percent == initial deposit (non-base values and denom): success": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(56912))),
			minInitialDepositPercent: 50,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, minLuncDeposit.Mul(sdk.NewInt(50)).Quo(sdk.NewInt(100)).Add(sdk.NewInt(10)))),
		},
		"min deposit * initial percent == initial deposit but different denoms: error": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin("uosmo", minLuncDeposit)),
			expectError:              true,
		},
		"min deposit * initial percent == initial deposit (multiple coins): success": {
			minDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount)),
				sdk.NewCoin("uosmo", sdk.NewInt(baseUSDDepositTestAmount*2))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, minLuncDeposit),
				sdk.NewCoin("uosmo", sdk.NewInt(baseUSDDepositTestAmount*2*baseDepositTestPercent/100)),
			),
		},
		"min deposit * initial percent > initial deposit (multiple coins): error": {
			minDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount)),
				sdk.NewCoin("uosmo", sdk.NewInt(baseUSDDepositTestAmount*2))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, minLuncDeposit),
				sdk.NewCoin("uosmo", sdk.NewInt(baseUSDDepositTestAmount*2*baseDepositTestPercent/100-1)),
			),

			expectError: true,
		},
		"min deposit * initial percent < initial deposit (multiple coins - coin not required by min deposit): success": {
			minDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: baseDepositTestPercent,
			initialDeposit: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, minLuncDeposit),
				sdk.NewCoin("uosmo", sdk.NewInt(baseUSDDepositTestAmount*baseDepositTestPercent/100-1)),
			),
		},
		"0 initial percent: success": {
			minDeposit:               sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseUSDDepositTestAmount))),
			minInitialDepositPercent: 0,
			initialDeposit:           sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, minLuncDeposit)),
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {

			params := v2lunc1.DefaultParams()
			params.MinDeposit = tc.minDeposit
			params.MinInitialDepositRatio = sdk.NewDec(tc.minInitialDepositPercent).Quo(sdk.NewDec(100)).String()
			govKeeper.SetParams(ctx, params)

			err := govKeeper.validateInitialDeposit(ctx, tc.initialDeposit)

			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
