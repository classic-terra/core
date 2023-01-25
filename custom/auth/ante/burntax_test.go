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

func (suite *AnteTestSuite) TestEnsureBurnTaxModule() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.DistrKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()

	// msg and signatures
	sendAmount := int64(1_000_000)
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
	suite.ctx = suite.ctx.WithBlockHeight(10_000_000)

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

func (suite *AnteTestSuite) TestEnsureIBCUntaxed() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()

	// msg and signatures
	sendAmount := int64(1_000_000)
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.OsmoIbcDenom, sendAmount))
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
	
	// Set the blockheight past the tax height block
	suite.ctx = suite.ctx.WithBlockHeight(10_000_000)

	// IBC must pass without burn before the specified tax block height
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().Error(err, "Decorator should have errored on IBC denoms when block height is 10,000,000")

	// Set the blockheight past the tax height block
	suite.ctx = suite.ctx.WithBlockHeight(12_000_000)

	// IBC must pass without burn after the specified tax block height
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err, "Decorator should not have errored on IBC denoms when block height is 12,000,000")

}

// the following binance addresses should not be applied tax
// go test -v -run ^TestAnteTestSuite/TestFilterRecipient$ github.com/terra-money/core/custom/auth/ante
func (suite *AnteTestSuite) TestFilterRecipient() {

	// keys and addresses
	var privs []cryptotypes.PrivKey
	var addrs []sdk.AccAddress

	// 0, 1: binance
	// 2, 3: normal
	for i := 0; i < 4; i++ {
		priv, _, addr := testdata.KeyTestPubAddr()
		privs = append(privs, priv)
		addrs = append(addrs, addr)
	}

	ante.BurnTaxAddressWhitelist = []string{
		addrs[0].String(),
		addrs[1].String(),
	}

	// set send amount
	sendAmt := int64(1000000)
	sendCoin := sdk.NewInt64Coin(core.MicroSDRDenom, sendAmt)
	feeAmt := int64(1000)

	cases := []struct {
		name       string
		msgSigner  cryptotypes.PrivKey
		msgCreator func() []sdk.Msg
		burnAmount int64
		feeAmount  int64
	}{
		{
			name:      "MsgSend(binance -> binance)",
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// skip this one hence burn amount is 0
			burnAmount: 0,
			feeAmount:  feeAmt,
		}, {
			name:      "MsgSend(normal -> normal)",
			msgSigner: privs[2],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[2], addrs[3], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			burnAmount: feeAmt,
			feeAmount:  feeAmt,
		}, {
			name:      "MsgSend(binance -> normal), MsgSend(binance -> binance)",
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[2], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg2)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			burnAmount: feeAmt * 2,
			feeAmount:  feeAmt * 2,
		}, {
			name:      "MsgSend(binance -> binance), MsgMultiSend(binance -> normal, binance -> binance)",
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgMultiSend(
					[]banktypes.Input{
						{
							Address: addrs[0].String(),
							Coins:   sdk.NewCoins(sendCoin),
						},
						{
							Address: addrs[0].String(),
							Coins:   sdk.NewCoins(sendCoin),
						},
					},
					[]banktypes.Output{
						{
							Address: addrs[2].String(),
							Coins:   sdk.NewCoins(sendCoin),
						},
						{
							Address: addrs[1].String(),
							Coins:   sdk.NewCoins(sendCoin),
						},
					},
				)
				msgs = append(msgs, msg2)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			burnAmount: feeAmt * 3,
			feeAmount:  feeAmt * 3,
		},
	}

	// there should be no coin in burn module
	for _, c := range cases {
		suite.SetupTest(true) // setup
		fmt.Printf("CASE = %s \n", c.name)
		suite.ctx = suite.ctx.WithBlockHeight(ante.WhitelistHeight)
		suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

		mfd := ante.NewBurnTaxFeeDecorator(suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.DistrKeeper)
		antehandler := sdk.ChainAnteDecorators(
			cosmosante.NewDeductFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper),
			mfd,
		)

		for i := 0; i < 4; i++ {
			fundCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1000000000))
			acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addrs[i])
			suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, fundCoins)
			suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, addrs[i], fundCoins)
		}

		// msg and signatures
		feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, c.feeAmount))
		gasLimit := testdata.NewTestGasLimit()
		suite.Require().NoError(suite.txBuilder.SetMsgs(c.msgCreator()...))
		suite.txBuilder.SetFeeAmount(feeAmount)
		suite.txBuilder.SetGasLimit(gasLimit)

		privs, accNums, accSeqs := []cryptotypes.PrivKey{c.msgSigner}, []uint64{0}, []uint64{0}
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
		fmt.Printf("amount fee after = %v \n", amountFee)
		amountBurn := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
		fmt.Printf("amount burn after = %v \n", amountBurn)

		if c.burnAmount > 0 {
			suite.Require().Equal(amountBurnBefore.Amount.Add(sdk.NewInt(c.burnAmount)), amountBurn.Amount)
			suite.Require().Equal(amountFeeBefore, amountFee)
		} else {
			suite.Require().Equal(amountBurnBefore, amountBurn)
			suite.Require().Equal(amountFeeBefore.Amount.Add(sdk.NewInt(c.feeAmount)), amountFee.Amount)
		}
}
