package ante_test

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"

	"github.com/classic-terra/core/v3/x/tax2gas/ante"
)

func (s *AnteTestSuite) TestDeductFeeDecorator_ZeroGas() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFeeDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, s.app.TreasuryKeeper, s.app.Tax2gasKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(300)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	// msg and signatures
	msg := testdata.NewTestMsg(addr1)
	s.Require().NoError(s.txBuilder.SetMsgs(msg))

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	testCases := []struct {
		name       string
		simulation bool
		checkTx    bool
		mallate    func()
		expFail    bool
		expErrMsg  string
	}{
		{
			name:       "success: zero gas in simulation",
			simulation: true,
			checkTx:    true,
			mallate: func() {
				// set zero gas
				s.txBuilder.SetGasLimit(0)
			},
			expFail: false,
		},
		{
			name:       "success: ensure mempool fees",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				feeAmount := testdata.NewTestFeeAmount()
				gasLimit := uint64(15)
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
			},
			expFail: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			tc.mallate()
			// Set IsCheckTx to true
			s.ctx = s.ctx.WithIsCheckTx(tc.checkTx)

			// zero gas is accepted in simulation mode
			_, err = antehandler(s.ctx, tx, tc.simulation)

			if tc.expFail {
				s.Require().Error(err)
				s.Require().Contains(err, tc.expErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}

}
