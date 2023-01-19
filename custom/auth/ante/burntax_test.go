package ante_test

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/terra-money/core/custom/auth/ante"
	core "github.com/terra-money/core/types"
	treasury "github.com/terra-money/core/x/treasury/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// go test -v -run ^TestAnteTestSuite/TestEnsureBurnTaxModule$ github.com/terra-money/core/custom/auth/ante
func (suite *AnteTestSuite) TestEnsureBurnTaxModule() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.DistrKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()

	// msg and signatures
	sendAmount := int64(1000000)
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))
	msg := banktypes.NewMsgSend(addr1, addr1, sendCoins)

	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// set zero gas prices
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins())

	// Set IsCheckTx to true
	suite.ctx = suite.ctx.WithIsCheckTx(true)

	// Luna must pass without burn before the specified tax block height
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err, "Decorator should not have errored when block height is 1")

	// Set the blockheight past the tax height block
	suite.ctx = suite.ctx.WithBlockHeight(10000000)
	// antehandler errors with insufficient fees due to tax
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().Error(err, "Decorator should errored on low fee for local gasPrice + tax")

	tk := suite.app.TreasuryKeeper
	expectedTax := tk.GetTaxRate(suite.ctx).MulInt64(sendAmount).TruncateInt()
	if taxCap := tk.GetTaxCap(suite.ctx, core.MicroSDRDenom); expectedTax.GT(taxCap) {
		expectedTax = taxCap
	}

	taxes := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, expectedTax.Int64()))

	bk := suite.app.BankKeeper
	bk.MintCoins(suite.ctx, minttypes.ModuleName, sendCoins)

	// Populate the FeeCollector module with taxes
	bk.SendCoinsFromModuleToModule(suite.ctx, minttypes.ModuleName, types.FeeCollectorName, taxes)
	feeCollector := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.FeeCollectorName)

	amountFee := bk.GetAllBalances(suite.ctx, feeCollector.GetAddress())
	suite.Require().Equal(amountFee, taxes)
	totalSupply, _, err := bk.GetPaginatedTotalSupply(suite.ctx, &query.PageRequest{})

	// must pass with tax and burn
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err, "Decorator should not have errored on fee higher than local gasPrice")

	// Burn the taxes
	tk.BurnCoinsFromBurnAccount(suite.ctx)
	suite.Require().NoError(err)

	supplyAfterBurn, _, err := bk.GetPaginatedTotalSupply(suite.ctx, &query.PageRequest{})

	// Total supply should have decreased by the tax amount
	suite.Require().Equal(taxes, totalSupply.Sub(supplyAfterBurn))
}

// the following binance addresses should not be applied tax
// go test -v -run ^TestAnteTestSuite/TestFilterRecipient$ github.com/terra-money/core/custom/auth/ante
//
// N is set of all Binance whitelist addresses
// “Inputs” is set of all sender address
// “Outputs” is set of all recipient address
// There is one - to - one correspondence between the elements of the two sets
// (in, out) is a pair of element from two sets
//
// sender1 -> amount1 -> receiver1 (Inputs[0]-Outputs[0])
// sender2 -> amount2 -> receiver2 (Inputs[1]-Outputs[1])
// sender3 -> amount3 -> receiver3 (Inputs[2]-Outputs[2])
//
// case 1: ∀Inputs ∊ N, ∀Outputs ∊ N -> ∀(in, out) pass
// case 2: ∀Inputs ∊ N, ∃Outputs ∊ N -> ∀(in, out) pass
// case 3: ∃Inputs ∊ N, ∀Outputs ∊ N -> ∀(in, out) pass
// case 4: ∄Inputs ∊ N, ∀Outputs ∊ N -> ∀(in, out) pass
// case 5: ∀Inputs ∊ N, ∄Outputs ∊ N -> ∀(in, out) pass
// case 6: ∃Inputs ∊ N, ∃Outputs ∊ N -> ∃(in, out) ∊ N pass, ∃(in, out) ∉ N tax
// case 7: ∃Inputs ∊ N, ∄Outputs ∊ N -> ∃(in, out) ∊ N pass, ∃(in, out) ∉ N tax
// case 8: ∄Inputs ∊ N, ∃Outputs ∊ N -> ∃(in, out) ∊ N pass, ∃(in, out) ∉ N tax
// case 9: ∄Inputs ∊ N, ∄Outputs ∊ N -> ∀(in, out) tax
//
// E = ∀Inputs ∊ N || ∀Outputs ∊ N -> ∀(in, out) pass
// E1 = ∄Inputs ∊ N && ∄Outputs ∊ N -> ∀(in, out) tax
// Default: calculate tax cut for ∃(in, out) ∉ N

func (suite *AnteTestSuite) TestFilterRecipient() {

	// keys and addresse
	// 1 and 2 is binance whitelist address
	//3 and 4 is normal address
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	priv2, _, addr2 := testdata.KeyTestPubAddr()
	priv3, _, addr3 := testdata.KeyTestPubAddr()
	_, _, addr4 := testdata.KeyTestPubAddr()
	ante.BurnTaxAddressWhitelist = map[string]byte{
		addr1.String(): 1,
		addr2.String(): 1,
	}

	sendAmount := int64(1000000)
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))

	cases := []struct {
		name            string
		inputAddresses  []banktypes.Input
		senderPriv      cryptotypes.PrivKey
		outputAddresses []banktypes.Output
		shouldTax       int64
		blockHeight     int64
	}{
		{
			name: "send token from one of binance address",
			inputAddresses: []banktypes.Input{
				{
					Address: addr1.String(),
					Coins:   sendCoins,
				}, {
					Address: addr3.String(),
					Coins:   sendCoins,
				}},
			senderPriv: priv1,
			outputAddresses: []banktypes.Output{
				{
					Address: addr4.String(),
					Coins:   sendCoins,
				}, {
					Address: addr4.String(),
					Coins:   sendCoins,
				}},
			shouldTax:   1000,
			blockHeight: ante.WhitelistHeight,
		}, {
			name: "token received by one of binance address",
			inputAddresses: []banktypes.Input{
				{
					Address: addr3.String(),
					Coins:   sendCoins,
				}, {
					Address: addr4.String(),
					Coins:   sendCoins,
				}},
			senderPriv: priv3,
			outputAddresses: []banktypes.Output{
				{
					Address: addr1.String(),
					Coins:   sendCoins,
				}, {
					Address: addr3.String(),
					Coins:   sendCoins,
				}},
			shouldTax:   1000,
			blockHeight: ante.WhitelistHeight,
		}, {
			name: "normal tax cut for two normal address",
			inputAddresses: []banktypes.Input{
				{
					Address: addr3.String(),
					Coins:   sendCoins,
				}, {
					Address: addr4.String(),
					Coins:   sendCoins,
				}},
			senderPriv: priv3,
			outputAddresses: []banktypes.Output{
				{
					Address: addr4.String(),
					Coins:   sendCoins,
				}, {
					Address: addr3.String(),
					Coins:   sendCoins,
				}},
			shouldTax:   2000,
			blockHeight: ante.WhitelistHeight,
		}, {
			name: "send token from one of binance address to one of binance address",
			inputAddresses: []banktypes.Input{
				{
					Address: addr2.String(),
					Coins:   sendCoins,
				}, {
					Address: addr3.String(),
					Coins:   sendCoins,
				}},
			senderPriv: priv2,
			outputAddresses: []banktypes.Output{
				{
					Address: addr4.String(),
					Coins:   sendCoins,
				}, {
					Address: addr1.String(),
					Coins:   sendCoins,
				}},
			shouldTax:   0,
			blockHeight: ante.WhitelistHeight,
		},
	}

	// there should be no coin in burn module
	for _, c := range cases {
		fmt.Printf("CASE = %s \n", c.name)
		suite.SetupTest(true) // setup
		suite.ctx = suite.ctx.WithBlockHeight(c.blockHeight)
		suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

		for _, input := range c.inputAddresses {
			fundCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 2000000))
			addr := sdk.MustAccAddressFromBech32(input.Address)
			acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
			suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, fundCoins)
			suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, addr, fundCoins)
		}

		mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.DistrKeeper)
		antehandler := sdk.ChainAnteDecorators(
			cosmosante.NewDeductFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper),
			mfd,
		)

		// msg and signatures
		msg := banktypes.NewMsgMultiSend(c.inputAddresses, c.outputAddresses)

		feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, c.shouldTax))
		if c.shouldTax == 0 {
			feeAmount = sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1000))
		}
		gasLimit := testdata.NewTestGasLimit()
		suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
		suite.txBuilder.SetFeeAmount(feeAmount)
		suite.txBuilder.SetGasLimit(gasLimit)

		privs, accNums, accSeqs := []cryptotypes.PrivKey{c.senderPriv}, []uint64{0}, []uint64{0}
		tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
		suite.Require().NoError(err)

		// check fee decorator and burn module amount before ante handler
		feeCollector := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.FeeCollectorName)
		burnModule := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, treasury.BurnModuleName)

		amountFeeBefore := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
		fmt.Printf("amount fee before = %v \n", amountFeeBefore)
		amountBurnBefore := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
		fmt.Printf("amount burn before = %v \n", amountBurnBefore)

		_, err = antehandler(suite.ctx, tx, false)
		suite.Require().NoError(err)

		// check fee decorator
		amountFee := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
		fmt.Printf("amount fee = %v \n", amountFee)
		amountBurn := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
		fmt.Printf("amount burn before = %v \n", amountBurn)

		if c.shouldTax > 0 {
			// if should tax, send tax from fee collector to burn module
			suite.Require().Equal(amountBurnBefore.Amount.Add(sdk.NewInt(c.shouldTax)), amountBurn.Amount)
			suite.Require().Equal(amountFeeBefore, amountFee)
		} else {
			// if no tax, tax will remain in fee collector
			suite.Require().Equal(amountBurnBefore, amountBurn)
			suite.Require().Equal(amountFeeBefore.Amount.Add(sdk.NewInt(1000)), amountFee.Amount)
		}
	}
}
