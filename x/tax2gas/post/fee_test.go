package post_test

import (
	oracle "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/classic-terra/core/v3/x/tax2gas/ante"
	"github.com/classic-terra/core/v3/x/tax2gas/post"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	"github.com/classic-terra/core/v3/x/tax2gas/utils"
	"github.com/classic-terra/core/v3/x/treasury/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (s *PostTestSuite) TestDeductFeeDecorator() {
	s.SetupTest(true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := post.NewTax2GasPostDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, s.app.TreasuryKeeper, s.app.DistrKeeper, s.app.Tax2gasKeeper)
	posthandler := sdk.ChainPostDecorators(mfd)

	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(10000000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	anteConsumedFee := int64(207566)
	postConsumedFee := int64(1368041)

	testCases := []struct {
		name           string
		simulation     bool
		checkTx        bool
		setupFunc      func()
		expectedOracle sdk.Coins
		expectedCp     sdk.Coins
		expectedBurn   sdk.Coins
		expFail        bool
		expErrMsg      string
	}{
		{
			name: "happy case",
			setupFunc: func() {
				s.setupTestCase(addr1, tx, 100000000, 100000, anteConsumedFee+500022)
			},
			// amount: 100000000
			// tax(0.5%): 500022
			// use default value
			// burn: tax * 0.9
			// distributtion: tax * 0.1 = 500022
			// cp: 2% of distribution: 499994 * 0.1 * 0.02 ~ 1000
			// oracle: distribution - oracle = 49002
			expectedBurn:   sdk.NewCoins(sdk.NewInt64Coin("uluna", 450020)),
			expectedCp:     sdk.NewCoins(sdk.NewInt64Coin("uluna", 1000)),
			expectedOracle: sdk.NewCoins(sdk.NewInt64Coin("uluna", 49002)),
		},
		{
			name: "not enough fee",
			setupFunc: func() {
				s.setupTestCase(addr1, tx, 100000000, 100000, 600000)
			},
			expFail: true,
		},
		{
			name: "combine ante + post",
			setupFunc: func() {
				s.setupCombineAnteAndPost(addr1, tx, 100000, 7328, 707559)
			},
			expFail: true,
		},
		{
			name: "combine ante + post, enough fee",
			setupFunc: func() {
				s.setupCombineAnteAndPost(addr1, tx, 100000000, 100000, anteConsumedFee+postConsumedFee+500022)
			},
			expFail: false,
			// amount: 100000000
			// tax(0.5%): 500022
			// use default value
			// burn: tax*0.9 = 500022 * 0.9 = 455020
			// distributtion: tax - burn = 50002
			// cp: 2% of distribution: 50002 * 0.02 ~ 1000
			// oracle: distribution - oracle = 49002
			expectedOracle: sdk.NewCoins(sdk.NewInt64Coin("uluna", 49002)),
			expectedCp:     sdk.NewCoins(sdk.NewInt64Coin("uluna", 1000)),
			expectedBurn:   sdk.NewCoins(sdk.NewInt64Coin("uluna", 455020)),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupFunc()
			s.ctx = s.ctx.WithIsCheckTx(true)

			_, err = posthandler(s.ctx, tx, tc.simulation, true)
			s.assertTestCase(tc, err)
		})
	}
}

func (s *PostTestSuite) setupTestCase(addr sdk.AccAddress, tx signing.Tx, sendAmount, gasLimit, feeAmount int64) {
	msg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(sendAmount))))
	gm := tax2gastypes.NewTax2GasMeter(s.ctx.GasMeter().Limit(), false)
	s.setupTax2GasMeter(gm, msg)
	s.ctx = s.ctx.WithGasMeter(gm)
	s.ctx = s.ctx.WithValue(tax2gastypes.AnteConsumedGas, uint64(7328))
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(uint64(gasLimit))
	s.ctx = s.ctx.WithValue(tax2gastypes.PaidDenom, "uluna")
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uluna", feeAmount)))
}

func (s *PostTestSuite) setupCombineAnteAndPost(addr sdk.AccAddress, tx signing.Tx, sendAmount, gasLimit, feeAmount int64) {
	msg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(sendAmount))))
	gm := tax2gastypes.NewTax2GasMeter(s.ctx.GasMeter().Limit(), false)
	//s.setupTax2GasMeter(gm, msg)
	mfd := ante.NewFeeDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, s.app.TreasuryKeeper, s.app.Tax2gasKeeper)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uluna", feeAmount)))
	s.txBuilder.SetGasLimit(uint64(gasLimit))
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.ctx = s.ctx.WithGasMeter(gm)

	antehandler := sdk.ChainAnteDecorators(mfd)
	newCtx, _ := antehandler(s.ctx, tx, false)
	distrBalance := s.app.BankKeeper.GetBalance(s.ctx, s.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName), "uluna")

	// msg send consume 7328 gas, uluna gas price: 28.325
	// expect only this consumed gas is transferred to fee collector account after ante.
	s.Require().Equal(sdk.NewDecWithPrec(7328, 0).Mul(sdk.NewDecWithPrec(28325, 3)).RoundInt64(), distrBalance.Amount.Int64())
	s.ctx = newCtx
}

func (s *PostTestSuite) setupTax2GasMeter(gm sdk.GasMeter, msg sdk.Msg) {
	burnTaxRate := s.app.Tax2gasKeeper.GetBurnTaxRate(s.ctx)
	lunaGasPrice := sdk.NewDecCoinFromDec("uluna", sdk.NewDecWithPrec(28325, 3))
	taxes := utils.FilterMsgAndComputeTax(s.ctx, s.app.TreasuryKeeper, burnTaxRate, msg)
	taxGas, _ := utils.ComputeGas(sdk.DecCoins{lunaGasPrice}, taxes)
	gm.(*tax2gastypes.Tax2GasMeter).ConsumeTax(taxGas, "tax")
}

func (s *PostTestSuite) assertTestCase(tc struct {
	name           string
	simulation     bool
	checkTx        bool
	setupFunc      func()
	expectedOracle sdk.Coins
	expectedCp     sdk.Coins
	expectedBurn   sdk.Coins
	expFail        bool
	expErrMsg      string
}, err error) {
	currentOracle := s.app.BankKeeper.GetBalance(s.ctx, s.app.AccountKeeper.GetModuleAddress(oracle.ModuleName), "uluna")
	currentCp := s.app.DistrKeeper.GetFeePoolCommunityCoins(s.ctx).AmountOf("uluna")
	currentBurn := s.app.BankKeeper.GetBalance(s.ctx, s.app.AccountKeeper.GetModuleAddress(types.BurnModuleName), "uluna")
	if tc.expFail {
		s.Require().Error(err)
		s.Require().Contains(err.Error(), tc.expErrMsg)
	} else {
		s.Require().NoError(err)
	}
	s.Require().Equal(tc.expectedOracle.AmountOf("uluna").Int64(), currentOracle.Amount.Int64())
	s.Require().Equal(tc.expectedCp.AmountOf("uluna").Int64(), currentCp.RoundInt64())
	s.Require().Equal(tc.expectedBurn.AmountOf("uluna").Int64(), currentBurn.Amount.Int64())
}
