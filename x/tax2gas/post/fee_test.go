package post_test

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v3/x/tax2gas/post"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
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

	testCases := []struct {
		name       string
		simulation bool
		checkTx    bool
		mallate    func()
		feeAmount  sdk.Coins
		expFail    bool
		expErrMsg  string
	}{
		{name: "happy case",
			simulation: false,
			checkTx:    false,
			mallate: func() {
				msg := banktypes.NewMsgSend(addr1, addr1, sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100000000))))
				gm := tax2gastypes.NewTax2GasMeter(s.ctx.GasMeter().Limit(), false)
				gm.(*tax2gastypes.Tax2GasMeter).ConsumeTax(math.NewInt(17778), "tax")
				s.ctx = s.ctx.WithGasMeter(gm)
				s.ctx = s.ctx.WithValue(tax2gastypes.AnteConsumedGas, uint64(1000))
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetGasLimit(100000)
				s.ctx = s.ctx.WithValue(tax2gastypes.PaidDenom, "uluna")
			},
			feeAmount: sdk.NewCoins(sdk.NewInt64Coin("uluna", 100000)),
			expFail:   false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			tc.mallate()
			s.ctx = s.ctx.WithIsCheckTx(true)
			s.txBuilder.SetFeeAmount(tc.feeAmount)

			_, err = posthandler(s.ctx, tx, tc.simulation, true)

			if tc.expFail {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
