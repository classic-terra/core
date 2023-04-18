package ante_test

import (
	"github.com/classic-terra/core/custom/auth/ante"
	core "github.com/classic-terra/core/types"
	markettypes "github.com/classic-terra/core/x/market/types"
	wasmtypes "github.com/classic-terra/core/x/wasm/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// go test -v -run ^TestAnteTestSuite/TestBlockAddrTx$ github.com/classic-terra/core/custom/auth/ante
func (suite *AnteTestSuite) TestBlockAddrTx() {
	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	priv2, _, addr2 := testdata.KeyTestPubAddr()
	priv3, _, addr3 := testdata.KeyTestPubAddr()

	ante.BlockedAddr[addr1.String()] = true
	ante.BlockedAddr[addr2.String()] = true

	// prepare amount
	sendAmount := int64(1000000)
	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))

	testCases := []struct {
		name    string
		msg     sdk.Msg
		privs   []cryptotypes.PrivKey
		blocked bool
	}{
		{
			"send from blocked address",
			banktypes.NewMsgSend(addr1, addr3, sendCoins),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"multisend from blocked address",
			banktypes.NewMsgMultiSend([]banktypes.Input{
				{Address: addr2.String(), Coins: sendCoins},
			}, []banktypes.Output{
				{Address: addr3.String(), Coins: sendCoins},
			}),
			[]cryptotypes.PrivKey{priv2},
			true,
		},
		{
			"swap from blocked address",
			markettypes.NewMsgSwapSend(addr2, addr3, sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount), core.MicroLunaDenom),
			[]cryptotypes.PrivKey{priv2},
			true,
		},
		{
			"execute contract from blocked address",
			wasmtypes.NewMsgExecuteContract(addr1, nil, nil, sendCoins),
			[]cryptotypes.PrivKey{priv1},
			true,
		},
		{
			"send from not blocked address",
			banktypes.NewMsgSend(addr3, addr1, sendCoins),
			[]cryptotypes.PrivKey{priv3},
			false,
		},
	}

	for _, testcase := range testCases {
		suite.Run(testcase.name, func() {
			suite.SetupTest(true) // setup
			suite.ctx = suite.ctx.WithBlockHeight(ante.FreezeAddrHeight + 1)

			feeAmount := testdata.NewTestFeeAmount()
			gasLimit := testdata.NewTestGasLimit()
			suite.Require().NoError(suite.txBuilder.SetMsgs(testcase.msg))
			suite.txBuilder.SetFeeAmount(feeAmount)
			suite.txBuilder.SetGasLimit(gasLimit)

			tx, err := suite.CreateTestTx(testcase.privs, []uint64{0}, []uint64{0}, suite.ctx.ChainID())
			suite.Require().NoError(err)

			antehandler := sdk.ChainAnteDecorators(ante.NewFreezeAddrDecorator())
			_, err = antehandler(suite.ctx, tx, false)

			if testcase.blocked {
				suite.Require().ErrorContains(err, "blocked address")
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
