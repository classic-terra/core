package classictax_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	core "github.com/classic-terra/core/v2/types"
	"github.com/stretchr/testify/suite"

	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/CosmWasm/wasmd/x/wasm"
	terraapp "github.com/classic-terra/core/v2/app"
	"github.com/classic-terra/core/v2/x/classictax/ante"
	"github.com/classic-terra/core/v2/x/classictax/post"
	classictaxtypes "github.com/classic-terra/core/v2/x/classictax/types"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	suite.Suite

	app *terraapp.TerraApp
	// anteHandler sdk.AnteHandler
	ctx       sdk.Context
	clientCtx client.Context
	txBuilder client.TxBuilder
}

// returns context and app with params set on account keeper
func createTestApp(isCheckTx bool, tempDir string) (*terraapp.TerraApp, sdk.Context) {
	// TODO: we need to feed in custom binding here?
	var wasmOpts []wasm.Option
	app := terraapp.NewTerraApp(
		log.NewNopLogger(), dbm.NewMemDB(), nil, true, map[int64]bool{},
		tempDir, simapp.FlagPeriodValue, terraapp.MakeEncodingConfig(),
		simapp.EmptyAppOptions{}, wasmOpts,
	)
	ctx := app.BaseApp.NewContext(isCheckTx, tmproto.Header{})
	app.AccountKeeper.SetParams(ctx, authtypes.DefaultParams())
	app.TreasuryKeeper.SetParams(ctx, treasurytypes.DefaultParams())
	app.DistrKeeper.SetParams(ctx, distributiontypes.DefaultParams())
	app.DistrKeeper.SetFeePool(ctx, distributiontypes.InitialFeePool())
	stakingparms := stakingtypes.DefaultParams()
	stakingparms.BondDenom = core.MicroLunaDenom
	app.StakingKeeper.SetParams(ctx, stakingparms)
	app.ClassicTaxKeeper.SetParams(ctx, classictaxtypes.DefaultParams())
	totalSupply := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, math.Int(math.LegacyNewDec(1_000_000_000_000))), sdk.NewCoin(core.MicroUSDDenom, math.Int(math.LegacyNewDec(1_000_000_000_000))))
	err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, totalSupply)
	if err != nil {
		panic("mint should not have failed")
	}

	return app, ctx
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (suite *AnteTestSuite) SetupTest(isCheckTx bool) {
	tempDir := suite.T().TempDir()
	suite.app, suite.ctx = createTestApp(isCheckTx, tempDir)
	suite.ctx = suite.ctx.WithBlockHeight(1)

	// Set up TxConfig.
	encodingConfig := suite.SetupEncoding()

	suite.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig)
}

func (suite *AnteTestSuite) SetupEncoding() simappparams.EncodingConfig {
	encodingConfig := simapp.MakeTestEncodingConfig()
	// We're using TestMsg encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	return encodingConfig
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (suite *AnteTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  suite.clientCtx.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err := suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			suite.clientCtx.TxConfig.SignModeHandler().DefaultMode(), signerData,
			suite.txBuilder, priv, suite.clientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func (suite *AnteTestSuite) CreateValidator(tokens int64) (cryptotypes.PrivKey, cryptotypes.PubKey, stakingtypes.Validator) {
	priv, pub, addr := testdata.KeyTestPubAddr()
	valAddr := sdk.ValAddress(addr)

	desc := stakingtypes.NewDescription("moniker", "", "", "", "")
	validator, err := stakingtypes.NewValidator(valAddr, pub, desc)
	suite.Require().NoError(err)

	commission := stakingtypes.NewCommissionWithTime(
		sdk.NewDecWithPrec(1, 2), sdk.NewDecWithPrec(1, 0),
		sdk.NewDecWithPrec(1, 0), suite.ctx.BlockHeader().Time,
	)

	validator, err = validator.SetInitialCommission(commission)
	suite.Require().NoError(err)

	validator.MinSelfDelegation = math.NewInt(1)
	suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
	suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	suite.app.StakingKeeper.SetNewValidatorByPowerIndex(suite.ctx, validator)

	err = suite.app.StakingKeeper.AfterValidatorCreated(suite.ctx, validator.GetOperator())
	suite.Require().NoError(err)

	// move coins to the validator account for self-delegation
	sendCoins := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2*tokens)))
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, addr, sendCoins)
	suite.Require().NoError(err)

	_, err = suite.app.StakingKeeper.Delegate(suite.ctx, addr, math.NewInt(tokens), stakingtypes.Unbonded, validator, true)
	suite.Require().NoError(err)
	err = suite.app.StakingKeeper.AfterDelegationModified(suite.ctx, addr, valAddr)
	suite.Require().NoError(err)

	// turn block for validator updates
	suite.ctx = suite.ctx.WithBlockTime(time.Now())
	staking.EndBlocker(suite.ctx, suite.app.StakingKeeper)

	retval, found := suite.app.StakingKeeper.GetValidator(suite.ctx, valAddr)
	suite.Require().Equal(true, found)
	return priv, pub, retval
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (suite *AnteTestSuite) TestAnte_GetTaxCoins() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"))
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1573yjmczqf5l95277as5qupn8v3ug7d88v0skv"))
	suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"), sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000)), sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(1_000_000_000))))

	priv1, _, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	mfd := ante.NewClassicTaxFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, suite.app.TreasuryKeeper, suite.app.OracleKeeper, suite.app.ClassicTaxKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)
	ph := post.NewClassicTaxPostDecorator(suite.app.ClassicTaxKeeper, suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.OracleKeeper)
	postHandler := sdk.ChainAnteDecorators(ph)

	// configure tx Builder
	suite.txBuilder.SetGasLimit(400_000)
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(25_000_000))))

	// invalid tx fails
	editmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))),
	)
	err := suite.txBuilder.SetMsgs(editmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	_, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	tax, tax2, _, err := suite.app.ClassicTaxKeeper.GetTaxCoins(suite.ctx, tx.GetMsgs()...)
	suite.Require().NoError(err)

	// print out values
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(5_000_000))), tax)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(5_000_000)), tax2)

	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(25_000_000))))
	// calculate for sending uusd
	sendmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(1_000_000_000))),
	)
	err = suite.txBuilder.SetMsgs(sendmsg)
	suite.Require().NoError(err)
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	_, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	tax, tax2, _, err = suite.app.ClassicTaxKeeper.GetTaxCoins(suite.ctx, tx.GetMsgs()...)
	suite.Require().NoError(err)

	// print out values
	// exchange rate is (from gas fees) 0.75 (uusd) / 28.325 (uluna) = 0.026478376
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(5_000_000))), tax)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(188_833_333)), tax2)

}

func (suite *AnteTestSuite) TestAnte_UnderpayTax() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"))
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1573yjmczqf5l95277as5qupn8v3ug7d88v0skv"))
	suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"), sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))))

	priv1, _, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	burnModule := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, treasurytypes.BurnModuleName)
	burnBefore := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroLunaDenom)
	fcModule := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, authtypes.FeeCollectorName)
	feeBefore := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroLunaDenom)

	mfd := ante.NewClassicTaxFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, suite.app.TreasuryKeeper, suite.app.OracleKeeper, suite.app.ClassicTaxKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)
	ph := post.NewClassicTaxPostDecorator(suite.app.ClassicTaxKeeper, suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.OracleKeeper)
	postHandler := sdk.ChainAnteDecorators(ph)

	// configure tx Builder
	suite.txBuilder.SetGasLimit(400_000)
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(22_000_000))))

	// invalid tx fails
	sendmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))),
	)

	err := suite.txBuilder.SetMsgs(sendmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().Error(err)

	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(30_000_000))))
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	suite.ctx, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	burnAfter := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroLunaDenom)
	feeAfter := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroLunaDenom)

	// nothing burned before
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), burnBefore)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(4_500_000)), burnAfter)

	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), feeBefore)
	suite.Require().Less(sdk.NewInt(10_500_000).Int64(), feeAfter.Amount.Int64())
}

func (suite *AnteTestSuite) TestAnte_TaxPaymentDenoms() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	// set stability tax to zero
	suite.app.TreasuryKeeper.SetTaxRate(suite.ctx, sdk.ZeroDec())
	suite.app.OracleKeeper.SetLunaExchangeRate(suite.ctx, core.MicroUSDDenom, sdk.NewDecWithPrec(5, 5))
	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"))
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1573yjmczqf5l95277as5qupn8v3ug7d88v0skv"))
	suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"), sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000)), sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(10_000_000))))

	priv1, _, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	burnModule := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, treasurytypes.BurnModuleName)
	burnBefore := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroLunaDenom)
	burnBeforeUSD := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroUSDDenom)
	fcModule := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, authtypes.FeeCollectorName)
	feeBefore := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroLunaDenom)
	feeBeforeUSD := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroUSDDenom)

	mfd := ante.NewClassicTaxFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, suite.app.TreasuryKeeper, suite.app.OracleKeeper, suite.app.ClassicTaxKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)
	ph := post.NewClassicTaxPostDecorator(suite.app.ClassicTaxKeeper, suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.OracleKeeper)
	postHandler := sdk.ChainAnteDecorators(ph)

	// configure tx Builder
	suite.txBuilder.SetGasLimit(400_000)
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2_000))))

	// invalid tx fails
	sendmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(100_000_000))),
	)

	err := suite.txBuilder.SetMsgs(sendmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.ctx, tx, false)
	suite.Require().Error(err)

	// first test with paying in coin denom
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(1_200_000))))
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	_, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	burnAfter := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroLunaDenom)
	feeAfter := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroLunaDenom)
	burnAfterUSD := suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroUSDDenom)
	feeAfterUSD := suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroUSDDenom)

	// nothing burned before
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), burnBefore)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), burnAfter)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.ZeroInt()), burnBeforeUSD)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(450_000)), burnAfterUSD)

	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), feeBefore)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt()), feeAfter)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.ZeroInt()), feeBeforeUSD)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(750_000)), feeAfterUSD)

	// now pay all in uluna
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(300_000_000))))
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	_, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	// tax split in tests is 90/10
	// tax is 500_000 uusd -> 18_883_333 uluna
	// 90% of that is 16_995_000
	// 10% of that is 1_888_333
	exchangeRate, err := suite.app.ClassicTaxKeeper.GetLunaExchangeRate(suite.ctx, core.MicroUSDDenom)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecWithPrec(264783759929391, 16), exchangeRate)

	taxInUluna, err := suite.app.ClassicTaxKeeper.CoinToMicroLuna(suite.ctx, sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(500_000)))
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(18_883_333)), taxInUluna)

	burnAfter = suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroLunaDenom)
	feeAfter = suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroLunaDenom)
	burnAfterUSD = suite.app.BankKeeper.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroUSDDenom)
	feeAfterUSD = suite.app.BankKeeper.GetBalance(suite.ctx, fcModule.GetAddress(), core.MicroUSDDenom)

	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(16_995_000)), burnAfter)
	suite.Require().Equal(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(225_581_295)), feeAfter)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(450_000)), burnAfterUSD)
	suite.Require().Equal(sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(750_000)), feeAfterUSD)
}

func (suite *AnteTestSuite) TestAnte_OverpayTax() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	curParams := suite.app.ClassicTaxKeeper.GetParams(suite.ctx)
	curParams.GasPrices = []sdk.DecCoin{sdk.NewDecCoinFromDec(core.MicroLunaDenom, sdk.NewDecWithPrec(10, 0))}
	suite.app.ClassicTaxKeeper.SetParams(suite.ctx, curParams)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"))
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1573yjmczqf5l95277as5qupn8v3ug7d88v0skv"))
	suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"), sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))))

	priv1, _, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	mfd := ante.NewClassicTaxFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, suite.app.TreasuryKeeper, suite.app.OracleKeeper, suite.app.ClassicTaxKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)
	ph := post.NewClassicTaxPostDecorator(suite.app.ClassicTaxKeeper, suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.OracleKeeper)
	postHandler := sdk.ChainAnteDecorators(ph)

	// configure tx Builder
	suite.txBuilder.SetGasLimit(400_000)
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100_000_000))))

	// invalid tx fails
	sendmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(500_000_000))),
	)
	err := suite.txBuilder.SetMsgs(sendmsg)
	suite.Require().NoError(err)

	// check tax gas
	tax, _, _, err := suite.app.ClassicTaxKeeper.GetTaxCoins(suite.ctx, sendmsg)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2_500_000))), tax)
	taxGas, _ := suite.app.ClassicTaxKeeper.CalculateTaxGas(suite.ctx, tax, curParams.GasPrices)
	suite.Require().Equal(sdk.NewInt(250_000).Int64(), int64(taxGas))

	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	suite.ctx, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)

	// check that balance for gas was deducted nonetheless
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100_000_000))), tx.GetFee())

	balance := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc.GetAddress())
	suite.Require().Less(sdk.NewInt(885_000_000).Int64(), balance.AmountOf(core.MicroLunaDenom).Int64())
	suite.Require().Greater(sdk.NewInt(950_000_000).Int64(), balance.AmountOf(core.MicroLunaDenom).Int64())

	value := suite.ctx.Value(classictaxtypes.CtxFeeKey)
	suite.Require().Less(sdk.NewInt(5_000_000).Int64(), value.(sdk.Coins).AmountOf(core.MicroLunaDenom).Int64())
}

func (suite *AnteTestSuite) TestAnte_RefundTax() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	curParams := suite.app.ClassicTaxKeeper.GetParams(suite.ctx)
	curParams.GasPrices = []sdk.DecCoin{sdk.NewDecCoinFromDec(core.MicroLunaDenom, sdk.NewDecWithPrec(10, 0))}
	suite.app.ClassicTaxKeeper.SetParams(suite.ctx, curParams)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"))
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress("terra1573yjmczqf5l95277as5qupn8v3ug7d88v0skv"))
	suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, sdk.AccAddress("terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"), sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))))

	priv1, _, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	mfd := ante.NewClassicTaxFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, suite.app.TreasuryKeeper, suite.app.OracleKeeper, suite.app.ClassicTaxKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)
	ph := post.NewClassicTaxPostDecorator(suite.app.ClassicTaxKeeper, suite.app.TreasuryKeeper, suite.app.BankKeeper, suite.app.OracleKeeper)
	postHandler := sdk.ChainAnteDecorators(ph)

	// configure tx Builder
	suite.txBuilder.SetGasLimit(400_000)
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100_000_000))))

	// invalid tx fails
	sendmsg := banktypes.NewMsgSend(
		acc.GetAddress(),
		acc2.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(500_000_000))),
	)
	err := suite.txBuilder.SetMsgs(sendmsg)
	suite.Require().NoError(err)

	// check tax gas
	tax, _, _, err := suite.app.ClassicTaxKeeper.GetTaxCoins(suite.ctx, sendmsg)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2_500_000))), tax)
	taxGas, _ := suite.app.ClassicTaxKeeper.CalculateTaxGas(suite.ctx, tax, curParams.GasPrices)
	suite.Require().Equal(sdk.NewInt(250_000).Int64(), int64(taxGas))

	// set gas prices to 0
	curParams.GasPrices = []sdk.DecCoin{sdk.NewDecCoinFromDec(core.MicroLunaDenom, sdk.ZeroDec())}
	suite.app.ClassicTaxKeeper.SetParams(suite.ctx, curParams)

	// send too much fees
	suite.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100_000_000))))

	balanceBefore := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc.GetAddress())
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
	suite.Require().NoError(err)
	suite.ctx, err = antehandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	suite.ctx, err = postHandler(suite.ctx, tx, false)
	suite.Require().NoError(err)
	balanceAfter := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc.GetAddress())

	// check that balance for gas was deducted nonetheless
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1_000_000_000))), balanceBefore)
	suite.Require().Equal(balanceBefore.Sub(tax...), balanceAfter)
}