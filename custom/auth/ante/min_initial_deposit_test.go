package ante_test

import (
	"fmt"

	"github.com/classic-terra/core/v3/custom/auth/ante"
	govv2lunc1 "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	core "github.com/classic-terra/core/v3/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func (suite *AnteTestSuite) TestMinInitialDepositRatioDefault() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	midd := ante.NewMinInitialDepositDecorator(suite.app.GovKeeper, suite.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(midd)

	lunaPriceInUSD := sdk.MustNewDecFromStr("0.10008905")
	fmt.Printf("\n lunaPriceInUSD %s", lunaPriceInUSD.String())
	suite.app.OracleKeeper.SetLunaExchangeRate(suite.ctx, core.MicroUSDDenom, lunaPriceInUSD)

	// set required deposit to uluna
	suite.app.GovKeeper.SetParams(suite.ctx, govv2lunc1.DefaultParams())
	govparams := suite.app.GovKeeper.GetParams(suite.ctx)
	govparams.MinUusdDeposit = sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(500_000_000))
	suite.app.GovKeeper.SetParams(suite.ctx, govparams)

	price, _ := suite.app.GovKeeper.GetMinimumDepositBaseUusd(suite.ctx)
	fmt.Printf("\n GetMinimumDepositBaseUusd %s", price.String())

	// set initial deposit ratio to 0.0
	ratio := sdk.ZeroDec()
	suite.app.TreasuryKeeper.SetMinInitialDepositRatio(suite.ctx, ratio)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	prop1 := govv1beta1.NewTextProposal("prop1", "prop1")
	depositCoins1 := sdk.NewCoins()

	// create prop tx
	msg, _ := govv1beta1.NewMsgSubmitProposal(prop1, depositCoins1, addr1)
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// antehandler should not error
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err, "error: Proposal whithout initial deposit should have gone through")

	// create v1 proposal
	msgv1, _ := govv1.NewMsgSubmitProposal([]sdk.Msg{}, depositCoins1, addr1.String(), "metadata", "title", "summary")
	feeAmountv1 := testdata.NewTestFeeAmount()
	gasLimitv1 := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msgv1))
	suite.txBuilder.SetFeeAmount(feeAmountv1)
	suite.txBuilder.SetGasLimit(gasLimitv1)
	privs, accNums, accSeqs = []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	txv1, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// ante handler should not error for v1 proposal with sufficient deposit
	_, err = antehandler(suite.ctx, txv1, false)
	suite.Require().NoError(err, "error: v1 proposal whithout initial deposit should have gone through")
}

func (suite *AnteTestSuite) TestMinInitialDepositRatioWithSufficientDeposit() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	midd := ante.NewMinInitialDepositDecorator(suite.app.GovKeeper, suite.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(midd)

	lunaPriceInUSD := sdk.MustNewDecFromStr("0.0001")
	fmt.Printf("\n lunaPriceInUSD %s", lunaPriceInUSD.String())
	suite.app.OracleKeeper.SetLunaExchangeRate(suite.ctx, core.MicroUSDDenom, lunaPriceInUSD)

	// set required deposit to uluna
	suite.app.GovKeeper.SetParams(suite.ctx, govv2lunc1.DefaultParams())
	govparams := suite.app.GovKeeper.GetParams(suite.ctx)
	govparams.MinUusdDeposit = sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(500_000_000))
	suite.app.GovKeeper.SetParams(suite.ctx, govparams)

	price, _ := suite.app.GovKeeper.GetMinimumDepositBaseUusd(suite.ctx)
	fmt.Printf("\n GetMinimumDepositBaseUusd %s", price.String())

	// set initial deposit ratio to 0.2
	ratio := sdk.NewDecWithPrec(2, 1)
	suite.app.TreasuryKeeper.SetMinInitialDepositRatio(suite.ctx, ratio)

	// keys and addresses

	initDeposit, _ := sdk.NewIntFromString("1000000000000")
	fmt.Printf("\n initDeposit %s", initDeposit.String())

	priv1, _, addr1 := testdata.KeyTestPubAddr()
	prop1 := govv1beta1.NewTextProposal("prop1", "prop1")
	depositCoins1 := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, initDeposit),
	)

	// create prop tx
	msg, _ := govv1beta1.NewMsgSubmitProposal(prop1, depositCoins1, addr1)
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// antehandler should not error
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err, "error: Proposal with sufficient initial deposit should have gone through")

	// create v1 proposal
	msgv1, _ := govv1.NewMsgSubmitProposal([]sdk.Msg{}, depositCoins1, addr1.String(), "metadata", "title", "summary")
	feeAmountv1 := testdata.NewTestFeeAmount()
	gasLimitv1 := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msgv1))
	suite.txBuilder.SetFeeAmount(feeAmountv1)
	suite.txBuilder.SetGasLimit(gasLimitv1)
	privs, accNums, accSeqs = []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	txv1, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// ante handler should not error for v1 proposal with sufficient deposit
	_, err = antehandler(suite.ctx, txv1, false)
	suite.Require().NoError(err, "error: v1 proposal with sufficient initial deposit should have gone through")
}

func (suite *AnteTestSuite) TestMinInitialDepositRatioWithInsufficientDeposit() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	midd := ante.NewMinInitialDepositDecorator(suite.app.GovKeeper, suite.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(midd)

	lunaPriceInUSD := sdk.MustNewDecFromStr("0.00008905")
	fmt.Printf("\n lunaPriceInUSD %s", lunaPriceInUSD.String())
	suite.app.OracleKeeper.SetLunaExchangeRate(suite.ctx, core.MicroUSDDenom, lunaPriceInUSD)

	// set required deposit to uluna
	suite.app.GovKeeper.SetParams(suite.ctx, govv2lunc1.DefaultParams())
	govparams := suite.app.GovKeeper.GetParams(suite.ctx)
	govparams.MinUusdDeposit = sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(500_000_000))
	suite.app.GovKeeper.SetParams(suite.ctx, govparams)

	price, _ := suite.app.GovKeeper.GetMinimumDepositBaseUusd(suite.ctx)
	fmt.Printf("\n GetMinimumDepositBaseUusd %s", price.String())

	// set initial deposit ratio to 0.2
	ratio := sdk.NewDecWithPrec(2, 1)
	suite.app.TreasuryKeeper.SetMinInitialDepositRatio(suite.ctx, ratio)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	prop1 := govv1beta1.NewTextProposal("prop1", "prop1")
	depositCoins1 := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100_000)),
	)

	// create prop tx
	msg, _ := govv1beta1.NewMsgSubmitProposal(prop1, depositCoins1, addr1)
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// antehandler should error with insufficient deposit
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().Error(err, "error: Proposal with insufficient initial deposit should have failed")

	// create v1 proposal
	msgv1, _ := govv1.NewMsgSubmitProposal([]sdk.Msg{}, depositCoins1, addr1.String(), "metadata", "title", "summary")
	feeAmountv1 := testdata.NewTestFeeAmount()
	gasLimitv1 := testdata.NewTestGasLimit()
	suite.Require().NoError(suite.txBuilder.SetMsgs(msgv1))
	suite.txBuilder.SetFeeAmount(feeAmountv1)
	suite.txBuilder.SetGasLimit(gasLimitv1)
	privs, accNums, accSeqs = []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	txv1, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// // ante handler should error for v1 proposal with insufficient deposit
	_, err = antehandler(suite.ctx, txv1, false)
	suite.Require().Error(err, "error: v1 proposal with insufficient initial deposit should have failed")
}
