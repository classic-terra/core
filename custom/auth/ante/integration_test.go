package ante_test

import (
	"fmt"

	"github.com/classic-terra/core/custom/auth/ante"
	customante "github.com/classic-terra/core/custom/auth/ante"
	core "github.com/classic-terra/core/types"
	treasurytypes "github.com/classic-terra/core/x/treasury/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	appparams "github.com/classic-terra/core/app/params"
	feeshareante "github.com/classic-terra/core/x/feeshare/ante"
)

// go test -v -run ^TestAnteTestSuite/TestIntegrationTaxExemption$ github.com/classic-terra/core/custom/auth/ante
func (suite *AnteTestSuite) TestIntegrationTaxExemption() {
	// keys and addresses
	var privs []cryptotypes.PrivKey
	var addrs []sdk.AccAddress

	// 0, 1: exemption
	// 2, 3: normal
	for i := 0; i < 4; i++ {
		priv, _, addr := testdata.KeyTestPubAddr()
		privs = append(privs, priv)
		addrs = append(addrs, addr)
	}

	// set send amount
	sendAmt := int64(1000000)
	sendCoin := sdk.NewInt64Coin(core.MicroSDRDenom, sendAmt)
	feeAmt := int64(1000)

	cases := []struct {
		name              string
		msgSigner         int
		msgCreator        func() []sdk.Msg
		expectedFeeAmount int64
	}{
		{
			name:      "MsgSend(exemption -> exemption)",
			msgSigner: 0,
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			expectedFeeAmount: 0,
		}, {
			name:      "MsgSend(normal -> normal)",
			msgSigner: 2,
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[2], addrs[3], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			expectedFeeAmount: feeAmt,
		}, {
			name:      "MsgSend(exemption -> normal), MsgSend(exemption -> exemption)",
			msgSigner: 0,
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[2], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg2)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			expectedFeeAmount: feeAmt,
		}, {
			name:      "MsgSend(exemption -> exemption), MsgMultiSend(exemption -> normal, exemption)",
			msgSigner: 0,
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgMultiSend(
					[]banktypes.Input{
						{
							Address: addrs[0].String(),
							Coins:   sdk.NewCoins(sendCoin.Add(sendCoin)),
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
			expectedFeeAmount: feeAmt * 2,
		},
	}

	for _, c := range cases {
		suite.SetupTest(true) // setup
		tk := suite.app.TreasuryKeeper
		ak := suite.app.AccountKeeper
		bk := suite.app.BankKeeper
		dk := suite.app.DistrKeeper
		fsk := suite.app.FeeShareKeeper

		// Set burn split rate to 50%
		// fee amount should be 500, 50% of 10000
		tk.SetBurnSplitRate(suite.ctx, sdk.NewDecWithPrec(5, 1)) //50%

		feeCollector := ak.GetModuleAccount(suite.ctx, types.FeeCollectorName)
		burnModule := ak.GetModuleAccount(suite.ctx, treasurytypes.BurnModuleName)

		encodingConfig := suite.SetupEncoding()
		antehandler, err := customante.NewAnteHandler(
			customante.HandlerOptions{
				AccountKeeper:      ak,
				BankKeeper:         bk,
				FeegrantKeeper:     suite.app.FeeGrantKeeper,
				OracleKeeper:       suite.app.OracleKeeper,
				TreasuryKeeper:     suite.app.TreasuryKeeper,
				SigGasConsumer:     ante.DefaultSigVerificationGasConsumer,
				SignModeHandler:    encodingConfig.TxConfig.SignModeHandler(),
				IBCChannelKeeper:   suite.app.IBCKeeper.ChannelKeeper,
				DistributionKeeper: dk,
				FeeShareKeeper:     fsk,
			},
		)
		suite.Require().NoError(err)

		fmt.Printf("CASE = %s \n", c.name)
		suite.ctx = suite.ctx.WithBlockHeight(ante.TaxPowerUpgradeHeight)
		suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

		tk.AddBurnTaxExemptionAddress(suite.ctx, addrs[0].String())
		tk.AddBurnTaxExemptionAddress(suite.ctx, addrs[1].String())

		for i := 0; i < 4; i++ {
			fundCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1_000_000_000_000))
			acc := ak.NewAccountWithAddress(suite.ctx, addrs[i])
			suite.Require().NoError(acc.SetAccountNumber(uint64(i)))
			ak.SetAccount(suite.ctx, acc)
			bk.MintCoins(suite.ctx, minttypes.ModuleName, fundCoins)
			bk.SendCoinsFromModuleToAccount(suite.ctx, minttypes.ModuleName, addrs[i], fundCoins)
		}

		// case 1 provides zero fee so not enough fee
		// case 2 provides enough fee
		feeCases := []int64{0, feeAmt}
		for i := 0; i < 1; i++ {
			feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, feeCases[i]))
			gasLimit := testdata.NewTestGasLimit()
			suite.Require().NoError(suite.txBuilder.SetMsgs(c.msgCreator()...))
			suite.txBuilder.SetFeeAmount(feeAmount)
			suite.txBuilder.SetGasLimit(gasLimit)

			privs, accNums, accSeqs := []cryptotypes.PrivKey{privs[c.msgSigner]}, []uint64{uint64(c.msgSigner)}, []uint64{uint64(i)}
			tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
			suite.Require().NoError(err)

			feeCollectorBefore := bk.GetBalance(suite.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
			burnBefore := bk.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
			communityBefore := dk.GetFeePool(suite.ctx).CommunityPool.AmountOf(core.MicroSDRDenom)
			supplyBefore := bk.GetSupply(suite.ctx, core.MicroSDRDenom)

			_, err = antehandler(suite.ctx, tx, false)
			if i == 0 && c.expectedFeeAmount != 0 {
				suite.Require().EqualError(err, fmt.Sprintf("insufficient fees; got: \"\", required: \"%dusdr\" = \"\"(gas) +\"%dusdr\"(stability): insufficient fee", c.expectedFeeAmount, c.expectedFeeAmount))
			} else {
				suite.Require().NoError(err)
			}

			feeCollectorAfter := bk.GetBalance(suite.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
			burnAfter := bk.GetBalance(suite.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
			communityAfter := dk.GetFeePool(suite.ctx).CommunityPool.AmountOf(core.MicroSDRDenom)
			supplyAfter := bk.GetSupply(suite.ctx, core.MicroSDRDenom)

			if i == 0 {
				suite.Require().Equal(feeCollectorBefore, feeCollectorAfter)
				suite.Require().Equal(burnBefore, burnAfter)
				suite.Require().Equal(communityBefore, communityAfter)
				suite.Require().Equal(supplyBefore, supplyAfter)
			}

			if i == 1 {
				suite.Require().Equal(feeCollectorBefore, feeCollectorAfter)
				splitAmount := sdk.NewInt(int64(float64(c.expectedFeeAmount) * 0.5))
				suite.Require().Equal(burnBefore, burnAfter.AddAmount(splitAmount))
				suite.Require().Equal(communityBefore, communityAfter.Add(sdk.NewDecFromInt(splitAmount)))
				suite.Require().Equal(supplyBefore, supplyAfter.SubAmount(splitAmount))
			}
		}
	}
}

// go test -v -run ^TestAnteTestSuite/TestFeeLogic$ github.com/classic-terra/core/custom/auth/ante
func (suite *AnteTestSuite) TestFeeLogic() {
	// We expect all to pass
	feeCoins := sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(500)), sdk.NewCoin("utoken", sdk.NewInt(250)))

	testCases := []struct {
		name               string
		incomingFee        sdk.Coins
		govPercent         sdk.Dec
		numContracts       int
		expectedFeePayment sdk.Coins
	}{
		{
			"100% fee / 1 contract",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			1,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(500)), sdk.NewCoin("utoken", sdk.NewInt(250))),
		},
		{
			"100% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			2,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(250)), sdk.NewCoin("utoken", sdk.NewInt(125))),
		},
		{
			"100% fee / 10 contracts",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			10,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(50)), sdk.NewCoin("utoken", sdk.NewInt(25))),
		},
		{
			"67% fee / 7 contracts",
			feeCoins,
			sdk.NewDecWithPrec(67, 2),
			7,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(48)), sdk.NewCoin("utoken", sdk.NewInt(24))),
		},
		{
			"50% fee / 1 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			1,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(250)), sdk.NewCoin("utoken", sdk.NewInt(125))),
		},
		{
			"50% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			2,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(125)), sdk.NewCoin("utoken", sdk.NewInt(62))),
		},
		{
			"50% fee / 3 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			3,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(83)), sdk.NewCoin("utoken", sdk.NewInt(42))),
		},
		{
			"25% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(25, 2),
			2,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(62)), sdk.NewCoin("utoken", sdk.NewInt(31))),
		},
		{
			"15% fee / 3 contracts",
			feeCoins,
			sdk.NewDecWithPrec(15, 2),
			3,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(25)), sdk.NewCoin("utoken", sdk.NewInt(12))),
		},
		{
			"1% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(1, 2),
			2,
			sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(2)), sdk.NewCoin("utoken", sdk.NewInt(1))),
		},
	}

	for _, tc := range testCases {
		// coins := feesharekeeper.FeePaySplitLogic(tc.incomingFee, tc.govPercent, tc.numContracts)
		coins := feeshareante.FeePayLogic(tc.incomingFee, tc.govPercent, tc.numContracts)

		for _, coin := range coins {
			for _, expectedCoin := range tc.expectedFeePayment {
				if coin.Denom == expectedCoin.Denom {
					suite.Require().Equal(expectedCoin.Amount.Int64(), coin.Amount.Int64(), tc.name)
				}
			}
		}
	}
}
