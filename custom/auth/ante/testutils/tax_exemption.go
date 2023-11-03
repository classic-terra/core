package testutils

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	customante "github.com/classic-terra/core/v2/custom/auth/ante"
	custompost "github.com/classic-terra/core/v2/custom/auth/post"
	core "github.com/classic-terra/core/v2/types"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
)

// go test -v -run ^TestAnteTestSuite/TestIntegrationTaxExemption$ github.com/classic-terra/core/v2/custom/auth/ante
func (s *AnteTestSuite) TestIntegrationTaxExemption() {
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
	sendAmt := int64(1_000_000)
	sendCoin := sdk.NewInt64Coin(core.MicroSDRDenom, sendAmt)
	feeAmt := int64(1000)
	taxAmt := int64(5000)

	cases := []struct {
		name              string
		account           sdk.AccAddress
		msgSigner         cryptotypes.PrivKey
		msgCreator        func() []sdk.Msg
		expectedFeeAmount int64
		expectedTaxAmount int64
	}{
		{
			name:      "MsgSend(exemption -> exemption)",
			account:   addrs[0],
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			expectedFeeAmount: feeAmt,
			expectedTaxAmount: 0,
		}, {
			name:      "MsgSend(normal -> normal)",
			account:   addrs[2],
			msgSigner: privs[2],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[2], addrs[3], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			expectedFeeAmount: feeAmt,
			expectedTaxAmount: taxAmt,
		}, {
			name:      "MsgSend(normal -> exemption)",
			account:   addrs[2],
			msgSigner: privs[2],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[2], addrs[0], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			expectedFeeAmount: feeAmt,
			expectedTaxAmount: taxAmt,
		}, {
			name:      "MsgSend(exemption -> normal), MsgSend(exemption -> exemption)",
			account:   addrs[0],
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[2], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg2)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			expectedFeeAmount: feeAmt * 2,
			expectedTaxAmount: taxAmt,
		}, {
			name:      "MsgSend(exemption -> exemption), MsgMultiSend(exemption -> normal, exemption)",
			account:   addrs[0],
			msgSigner: privs[0],
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
			expectedFeeAmount: feeAmt * 3,
			expectedTaxAmount: taxAmt * 2,
		},
	}

	for _, c := range cases {
		s.SetupTest(true) // setup
		tk := s.app.TreasuryKeeper
		ak := s.app.AccountKeeper
		bk := s.app.BankKeeper
		dk := s.app.DistrKeeper

		// Set burn split rate to 50%
		// fee amount should be 500, 50% of 10000
		burnSplitRate := sdk.NewDecWithPrec(5, 1)
		tk.SetBurnSplitRate(s.ctx, burnSplitRate) // 50%

		feeCollector := ak.GetModuleAccount(s.ctx, types.FeeCollectorName)
		burnModule := ak.GetModuleAccount(s.ctx, treasurytypes.BurnModuleName)

		encodingConfig := s.SetupEncoding()
		wasmConfig := wasmtypes.DefaultWasmConfig()
		antehandler, err := customante.NewAnteHandler(
			customante.HandlerOptions{
				AccountKeeper:      ak,
				BankKeeper:         bk,
				FeegrantKeeper:     s.app.FeeGrantKeeper,
				OracleKeeper:       s.app.OracleKeeper,
				TreasuryKeeper:     s.app.TreasuryKeeper,
				SigGasConsumer:     ante.DefaultSigVerificationGasConsumer,
				SignModeHandler:    encodingConfig.TxConfig.SignModeHandler(),
				IBCKeeper:          *s.app.IBCKeeper,
				DistributionKeeper: dk,
				WasmConfig:         &wasmConfig,
				TXCounterStoreKey:  s.app.GetKey(wasmtypes.StoreKey),
				ClassicTaxKeeper:   s.app.ClassicTaxKeeper,
			},
		)
		s.Require().NoError(err)

		posthandler, err := custompost.NewPostHandler(
			custompost.HandlerOptions{
				TreasuryKeeper:   s.app.TreasuryKeeper,
				BankKeeper:       s.app.BankKeeper,
				OracleKeeper:     s.app.OracleKeeper,
				ClassicTaxKeeper: s.app.ClassicTaxKeeper,
			},
		)
		s.Require().NoError(err)

		for i := 0; i < 4; i++ {
			coins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1_000_000_000))
			testutil.FundAccount(s.app.BankKeeper, s.ctx, addrs[i], coins)
		}

		s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

		tk.AddBurnTaxExemptionAddress(s.ctx, addrs[0].String())
		tk.AddBurnTaxExemptionAddress(s.ctx, addrs[1].String())

		// set zero gas prices
		s.ctx = s.ctx.WithMinGasPrices(sdk.NewDecCoins())
		curParams := s.app.ClassicTaxKeeper.GetParams(s.ctx)
		curParams.GasPrices = sdk.NewDecCoins()
		s.app.ClassicTaxKeeper.SetParams(s.ctx, curParams)

		s.Run(c.name, func() {
			// case 1 provides zero fee so not enough fee
			// case 2 provides enough fee (was never enabled)
			feeCases := []int64{c.expectedFeeAmount, c.expectedFeeAmount} // stability tax is not included in tax exemption!
			for i := 0; i < 1; i++ {
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, feeCases[i]))

				gasLimit := testdata.NewTestGasLimit()
				s.Require().NoError(s.txBuilder.SetMsgs(c.msgCreator()...))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)

				account := ak.GetAccount(s.ctx, c.account)
				privs, accNums, accSeqs := []cryptotypes.PrivKey{c.msgSigner}, []uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				feeCollectorBefore := bk.GetBalance(s.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
				burnBefore := bk.GetBalance(s.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
				communityBefore := dk.GetFeePool(s.ctx).CommunityPool.AmountOf(core.MicroSDRDenom)
				supplyBefore := bk.GetSupply(s.ctx, core.MicroSDRDenom)

				s.ctx, err = antehandler(s.ctx, tx, false)
				s.Require().NoError(err)

				s.ctx, err = posthandler(s.ctx, tx, false)
				if i == 0 && c.expectedTaxAmount != 0 {
					s.Require().EqualError(err, fmt.Sprintf(
						"insufficient fees; got: \"%dusdr\", required: \"%dusdr\" = \"\"(gas) + \"%dusdr\"(tax)/\"0uluna\"(tax_uluna) + \"%dusdr\"(stability): insufficient fee",
						c.expectedFeeAmount, c.expectedFeeAmount+c.expectedTaxAmount, c.expectedTaxAmount, c.expectedFeeAmount))
				} else {
					s.Require().NoError(err)
					newSeq := account.GetSequence()
					account.SetSequence(newSeq + 1)
				}

				feeCollectorAfter := bk.GetBalance(s.ctx, feeCollector.GetAddress(), core.MicroSDRDenom)
				burnAfter := bk.GetBalance(s.ctx, burnModule.GetAddress(), core.MicroSDRDenom)
				communityAfter := dk.GetFeePool(s.ctx).CommunityPool.AmountOf(core.MicroSDRDenom)
				supplyAfter := bk.GetSupply(s.ctx, core.MicroSDRDenom)

				if i == 0 {
					s.Require().Equal(feeCollectorBefore.AddAmount(sdk.NewInt(feeCases[i])), feeCollectorAfter)
					s.Require().Equal(burnBefore, burnAfter)
					s.Require().Equal(communityBefore, communityAfter)
					s.Require().Equal(supplyBefore, supplyAfter)
				}

				if i == 1 { // this test has never been active (see loop conditions)
					s.Require().Equal(feeCollectorBefore.AddAmount(sdk.NewInt(feeCases[i])), feeCollectorAfter)
					splitAmount := burnSplitRate.MulInt64(c.expectedFeeAmount).TruncateInt()
					s.Require().Equal(burnBefore, burnAfter.AddAmount(splitAmount))
					s.Require().Equal(communityBefore, communityAfter.Add(sdk.NewDecFromInt(splitAmount)))
					s.Require().Equal(supplyBefore, supplyAfter.SubAmount(splitAmount))
				}
			}
		})
	}
}
