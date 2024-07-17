package ante_test

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/tax2gas/ante"
)

var (
	sendCoins = sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, int64(1000000)))
)

func (s *AnteTestSuite) TestDeductFeeDecorator() {
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
			name:       "Success: deduct sufficient fees",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				// GasConsumed : 147542*28,325 = 4179127
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(4438670))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 4438670))
				gasLimit := uint64(15)
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
			},
			expFail: false,
		},
		{
			name:       "Success: Instantiate contract",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := &wasmtypes.MsgInstantiateContract{
					Sender: addr1.String(),
					Admin:  addr1.String(),
					CodeID: 0,
					Msg:    []byte{},
					Funds:  sendCoins,
				}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				// Consumed gas at the point of ante is: 238237
				// 239339*28.325 (gas fee) + 1000 (tax) = 6779278
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(6779278))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 6779278))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Success: Instantiate2 contract",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := &wasmtypes.MsgInstantiateContract2{
					Sender: addr1.String(),
					Admin:  addr1.String(),
					CodeID: 0,
					Msg:    []byte{},
					Funds:  sendCoins,
				}
				// Consumed gas at the point of ante is: 305215
				// 305215*28.325 (gas fee) + 1000 (tax) = 8646214
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(8646214))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 8646214))
				gasLimit := uint64(15)
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
			},
			expFail: false,
		},
		{
			name:       "Fail: Instantiate2 contract insufficient fees",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := &wasmtypes.MsgInstantiateContract2{
					Sender: addr1.String(),
					Admin:  addr1.String(),
					CodeID: 0,
					Msg:    []byte{},
					Funds:  sendCoins,
				}
				// Consumed gas at the point of ante is: 305215
				// 305215*28.325 (gas fee) + 1000 (tax) = 8646214
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(8646213))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 8646213))
				gasLimit := uint64(15)
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
			},
			expFail:   true,
			expErrMsg: "can't find coin",
		},
		{
			name:       "Success: Execute contract",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := &wasmtypes.MsgExecuteContract{
					Sender:   addr1.String(),
					Contract: addr1.String(),
					Msg:      []byte{},
					Funds:    sendCoins,
				}
				// Consumed gas at the point of ante is: 406592
				// 406592*28.325 (gas fee) + 1000 (tax) = 11517719
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(11517719))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 11517719))
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
				s.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				s.Require().NoError(err)
			}
		})
	}

}
