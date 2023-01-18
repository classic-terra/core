package ante_test

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	treasury "github.com/terra-money/core/x/treasury/types"

	"github.com/terra-money/core/custom/auth/ante"
	core "github.com/terra-money/core/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

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
func (suite *AnteTestSuite) TestFilterRecipient() {

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	priv2, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()
	ante.BurnTaxAddressWhitelist = []string{
		addr1.String(),
	}

	cases := []struct {
		name             string
		senderAddress    sdk.AccAddress
		senderPriv       cryptotypes.PrivKey
		recipientAddress sdk.AccAddress
		burnShouldWork   bool
	}{
		{
			name:             "send token from binance address",
			senderAddress:    addr1,
			senderPriv:       priv1,
			recipientAddress: addr2,
			burnShouldWork:   false,
		}, {
			name:             "token received by binance address",
			senderAddress:    addr2,
			senderPriv:       priv2,
			recipientAddress: addr1,
			burnShouldWork:   false,
		}, {
			name:             "normal tax cut",
			senderAddress:    addr2,
			senderPriv:       priv2,
			recipientAddress: addr3,
			burnShouldWork:   true,
		},
	}

	// there should be no coin in burn module
	for _, c := range cases {
		fmt.Printf("CASE = %s \n", c.name)
		suite.SetupTest(true) // setup
		suite.ctx = suite.ctx.WithBlockHeight(ante.TaxPowerSplitHeight)
		suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

		sendAmount := int64(1000000)
		sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))
		fundCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 2000000))
		acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, c.senderAddress)
		suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
		suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, fundCoins)
		suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, c.senderAddress, fundCoins)

		mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.DistrKeeper)
		antehandler := sdk.ChainAnteDecorators(
			cosmosante.NewDeductFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper),
			mfd,
		)

		// msg and signatures
		msg := banktypes.NewMsgSend(c.senderAddress, c.recipientAddress, sendCoins)

		feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1000))
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

		if c.burnShouldWork {
			suite.Require().Equal(amountBurnBefore.Amount.Add(sdk.NewInt(1000)), amountBurn.Amount)
			suite.Require().Equal(amountFeeBefore, amountFee)
		} else {
			suite.Require().Equal(amountBurnBefore, amountBurn)
			suite.Require().Equal(amountFeeBefore.Amount.Add(sdk.NewInt(1000)), amountFee.Amount)
		}
	}
}
