package post_test

import (
	oracle "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/classic-terra/core/v3/x/tax2gas/post"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	"github.com/classic-terra/core/v3/x/tax2gas/utils"
	"github.com/classic-terra/core/v3/x/treasury/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (s *PostTestSuite) TestDeductFeeDecorator() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := post.NewTax2GasPostDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, s.app.TreasuryKeeper, s.app.DistrKeeper, s.app.Tax2gasKeeper)
	posthandler := sdk.ChainPostDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(1000000000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	burnTaxRate := s.app.Tax2gasKeeper.GetBurnTaxRate(s.ctx)
	lunaGasPrice := sdk.NewDecCoinFromDec("uluna", sdk.NewDecWithPrec(28325, 3))
	testCases := []struct {
		name           string
		simulation     bool
		checkTx        bool
		mallate        func()
		feeAmount      sdk.Coins
		expectedOracle sdk.Coins
		expectedCp     sdk.Coins
		expectedBurn   sdk.Coins
		expFail        bool
		expErrMsg      string
	}{
		{
			name:       "happy case",
			simulation: false,
			checkTx:    false,
			mallate: func() {
				msg := banktypes.NewMsgSend(addr1, addr1, sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100000000))))
				gm := tax2gastypes.NewTax2GasMeter(s.ctx.GasMeter().Limit(), false)
				tax := utils.FilterMsgAndComputeTax(s.ctx, s.app.TreasuryKeeper, burnTaxRate, msg)
				taxDecCoin := sdk.NewDecFromInt(tax.AmountOf("uluna"))
				gm.(*tax2gastypes.Tax2GasMeter).ConsumeTax(taxDecCoin.Quo(lunaGasPrice.Amount).RoundInt(), "tax")
				s.ctx = s.ctx.WithGasMeter(gm)
				s.ctx = s.ctx.WithValue(tax2gastypes.AnteConsumedGas, uint64(7328))
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetGasLimit(100000)
				s.ctx = s.ctx.WithValue(tax2gastypes.PaidDenom, "uluna")
			},
			// amount: 100000000
			// tax(0.5%): 499993
			// use default value
			// burn: tax * 0.9
			// distributtion: tax * 0.1 = 49999
			// cp: 2% of distribution: 499994 * 0.1 * 0.02 ~ 1000
			// oracle: distribution - oracle = 48999
			feeAmount:      sdk.NewCoins(sdk.NewInt64Coin("uluna", 711128)),
			expFail:        false,
			expectedBurn:   sdk.NewCoins(sdk.NewInt64Coin("uluna", 449994)),
			expectedCp:     sdk.NewCoins(sdk.NewInt64Coin("uluna", 1000)),
			expectedOracle: sdk.NewCoins(sdk.NewInt64Coin("uluna", 48999)),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			tc.mallate()
			s.ctx = s.ctx.WithIsCheckTx(true)
			s.txBuilder.SetFeeAmount(tc.feeAmount)

			_, err = posthandler(s.ctx, tx, tc.simulation, true)
			currentOracle := s.app.BankKeeper.GetBalance(s.ctx, s.app.AccountKeeper.GetModuleAddress(oracle.ModuleName), "uluna")
			currentCp := s.app.DistrKeeper.GetFeePoolCommunityCoins(s.ctx).AmountOf("uluna")
			currentBurn := s.app.BankKeeper.GetBalance(s.ctx, s.app.AccountKeeper.GetModuleAddress(types.BurnModuleName), "uluna")
			if tc.expFail {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedOracle[0].Amount.Int64(), currentOracle.Amount.Int64())
				s.Require().Equal(tc.expectedCp[0].Amount.Int64(), currentCp.RoundInt64())
				s.Require().Equal(tc.expectedBurn[0].Amount.Int64(), currentBurn.Amount.Int64())
			}
		})
	}
}
