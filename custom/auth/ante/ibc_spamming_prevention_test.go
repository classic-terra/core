package ante_test

import (
	"github.com/classic-terra/core/v3/custom/auth/ante"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

func (suite *AnteTestSuite) TestIBCTransferSpamPrevention() {
	testCases := []struct {
		name      string
		malleate  func() (sdk.Tx, error)
		expectErr error
	}{
		{
			"memo too long",
			func() (sdk.Tx, error) {
				suite.SetupTest(true)
				suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

				priv1, _, addr1 := testdata.KeyTestPubAddr()
				_, _, addr2 := testdata.KeyTestPubAddr()

				msg := ibctransfertypes.NewMsgTransfer(
					"transfer",
					"channel-0",
					sdk.NewCoin("uluna", sdk.NewInt(100000)),
					addr1.String(),
					addr2.String(),
					clienttypes.NewHeight(1, 1000),
					0,
					string(make([]byte, ante.DefaultMaxMemoLength+1)), // greater than max memo length
				)

				feeAmount := testdata.NewTestFeeAmount()
				gasLimit := testdata.NewTestGasLimit()
				suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
				suite.txBuilder.SetFeeAmount(feeAmount)
				suite.txBuilder.SetGasLimit(gasLimit)

				privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
				return suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			},
			ante.ErrMemoTooLong,
		},
		{
			"receiver too long",
			func() (sdk.Tx, error) {
				suite.SetupTest(true)
				suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

				priv1, _, addr1 := testdata.KeyTestPubAddr()

				msg := ibctransfertypes.NewMsgTransfer(
					"transfer",
					"channel-0",
					sdk.NewCoin("uluna", sdk.NewInt(100000)),
					addr1.String(),
					string(make([]byte, ante.DefaultMaxReceiverLength+1)), // greater than max receiver length
					clienttypes.NewHeight(1, 1000),
					0,
					"normal memo",
				)

				feeAmount := testdata.NewTestFeeAmount()
				gasLimit := testdata.NewTestGasLimit()
				suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
				suite.txBuilder.SetFeeAmount(feeAmount)
				suite.txBuilder.SetGasLimit(gasLimit)

				privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
				return suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			},
			ante.ErrReceiverTooLong,
		},
		{
			"valid transfer",
			func() (sdk.Tx, error) {
				suite.SetupTest(true)
				suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

				priv1, _, addr1 := testdata.KeyTestPubAddr()
				_, _, addr2 := testdata.KeyTestPubAddr()

				msg := ibctransfertypes.NewMsgTransfer(
					"transfer",
					"channel-0",
					sdk.NewCoin("uluna", sdk.NewInt(100000)),
					addr1.String(),
					addr2.String(),
					clienttypes.NewHeight(1, 1000),
					0,
					"normal memo",
				)

				feeAmount := testdata.NewTestFeeAmount()
				gasLimit := testdata.NewTestGasLimit()
				suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
				suite.txBuilder.SetFeeAmount(feeAmount)
				suite.txBuilder.SetGasLimit(gasLimit)

				privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
				return suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			decorator := ante.NewIBCTransferSpamPreventionDecorator()
			antehandler := sdk.ChainAnteDecorators(decorator)

			tx, err := tc.malleate()
			suite.Require().NoError(err)

			_, err = antehandler(suite.ctx.WithIsCheckTx(true), tx, false)
			if tc.expectErr != nil {
				suite.Require().Equal(tc.expectErr, err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
