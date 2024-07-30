package ante_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	core "github.com/classic-terra/core/v3/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/classic-terra/core/v3/x/tax2gas/ante"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

var (
	sendCoin  = sdk.NewInt64Coin(core.MicroLunaDenom, int64(1000000))
	sendCoins = sdk.NewCoins(sendCoin)
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
				msg := testdata.NewTestMsg(addr1)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				// GasConsumed : 7328*28.325 = 207566
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(207566))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 207566))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Fail: deduct insufficient fees",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := testdata.NewTestMsg(addr1)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				// GasConsumed : 7328*28,325 = 207566
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(207565))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 207565))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail:   true,
			expErrMsg: "can't find coin",
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
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))
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
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
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
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833499))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833499))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail:   true,
			expErrMsg: "insufficient fees",
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
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))

				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Success: Bank send",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := banktypes.NewMsgSend(addr1, addr1, sendCoins)
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))

				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Success: Bank multisend",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := banktypes.NewMsgMultiSend(
					[]banktypes.Input{
						banktypes.NewInput(addr1, sendCoins),
						banktypes.NewInput(addr1, sendCoins),
					},
					[]banktypes.Output{
						banktypes.NewOutput(addr1, sendCoins),
						banktypes.NewOutput(addr1, sendCoins),
					})
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 2000 (tax) = 2834500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2834500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2834500))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Success: Market swapsend",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := markettypes.NewMsgSwapSend(addr1, addr1, sendCoin, core.MicroKRWDenom)
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Success: Authz exec",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := authz.NewMsgExec(addr1, []sdk.Msg{banktypes.NewMsgSend(addr1, addr1, sendCoins)})
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(&msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Fail: Authz exec",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := authz.NewMsgExec(addr1, []sdk.Msg{banktypes.NewMsgSend(addr1, addr1, sendCoins)})
				// Consumed gas at the point of ante is: 7220 but the gas limit is 100000
				// 100000*28.325 (gas fee) + 1000 (tax) = 2833500
				s.Require().NoError(s.txBuilder.SetMsgs(&msg))
				err = testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(2833500))))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 2833500))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(100000)
			},
			expFail: false,
		},
		{
			name:       "Bypass: ibc MsgRecvPacket",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := ibcchanneltypes.NewMsgRecvPacket(
					ibcchanneltypes.Packet{},
					[]byte(""),
					ibcclienttypes.ZeroHeight(),
					addr1.String(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
		{
			name:       "Not Bypass: ibc MsgRecvPacket",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := ibcchanneltypes.NewMsgRecvPacket(
					ibcchanneltypes.Packet{},
					[]byte(""),
					ibcclienttypes.ZeroHeight(),
					addr1.String(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_001)
			},
			expFail:   true,
			expErrMsg: "can't find coin",
		},
		{
			name:       "Bypass: ibc MsgAcknowledgement",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := ibcchanneltypes.NewMsgAcknowledgement(
					ibcchanneltypes.Packet{},
					[]byte(""),
					[]byte(""),
					ibcclienttypes.ZeroHeight(),
					addr1.String(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
		{
			name:       "Bypass: ibc MsgUpdateClient",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.app.AppCodec(), "solomachine", "", 2)
				msg, err := ibcclienttypes.NewMsgUpdateClient(
					soloMachine.ClientID,
					soloMachine.CreateHeader(soloMachine.Diversifier),
					string(addr1),
				)
				s.Require().NoError(err)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
		{
			name:       "Bypass: ibc MsgTimeout",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := ibcchanneltypes.NewMsgTimeout(
					ibcchanneltypes.Packet{},
					1,
					[]byte(""),
					ibcclienttypes.ZeroHeight(),
					addr1.String(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
		{
			name:       "Bypass: ibc MsgTimeoutOnClose",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := ibcchanneltypes.NewMsgTimeoutOnClose(
					ibcchanneltypes.Packet{},
					1,
					[]byte(""),
					[]byte(""),
					ibcclienttypes.ZeroHeight(),
					addr1.String(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
		{
			name:       "Other msgs must pay gas fee",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				msg := stakingtypes.NewMsgDelegate(
					addr1,
					sdk.ValAddress(addr1),
					sdk.NewCoin(core.MicroLunaDenom, math.NewInt(100000)),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail:   true,
			expErrMsg: "can't find coin",
		},
		{
			name:       "Oracle zero fee",
			simulation: false,
			checkTx:    true,
			mallate: func() {
				val, err := stakingtypes.NewValidator(sdk.ValAddress(addr1), priv1.PubKey(), stakingtypes.Description{})
				s.Require().NoError(err)

				msg := oracletypes.NewMsgAggregateExchangeRatePrevote(
					oracletypes.GetAggregateVoteHash("salt", "exchange rates", val.GetOperator()),
					addr1,
					val.GetOperator(),
				)
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 0))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(1_000_000)
			},
			expFail: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			tc.mallate()
			s.ctx = s.app.BaseApp.NewContext(tc.checkTx, tmproto.Header{})

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

func (s *AnteTestSuite) TestTaxExemption() {
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
	sendCoin := sdk.NewInt64Coin(core.MicroLunaDenom, sendAmt)
	feeAmt := int64(1000)

	cases := []struct {
		name         string
		msgSigner    cryptotypes.PrivKey
		msgCreator   func() []sdk.Msg
		minFeeAmount int64
		gasLimit     uint64
	}{
		{
			name:      "MsgSend(exemption -> exemption)",
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// 262593*28.325 = 7437947 - only gas fee
			minFeeAmount: 7437947,
		}, {
			name:      "MsgSend(normal -> normal)",
			msgSigner: privs[2],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[2], addrs[3], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			// gasLimit * 28.325 = 8497500
			minFeeAmount: 8497500 + feeAmt,
		}, {
			name:      "MsgExec(MsgSend(normal -> normal))",
			msgSigner: privs[2],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := authz.NewMsgExec(addrs[1], []sdk.Msg{banktypes.NewMsgSend(addrs[2], addrs[3], sdk.NewCoins(sendCoin))})
				msgs = append(msgs, &msg1)

				return msgs
			},
			// tax this one hence burn amount is fee amount
			// gasLimit * 28.325 = 8497500
			minFeeAmount: 8497500 + feeAmt,
		}, {
			name:      "MsgSend(exemption -> normal), MsgSend(exemption -> exemption)",
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
			// gasLimit * 28.325 = 8497500
			minFeeAmount: 8497500 + feeAmt,
		}, {
			name:      "MsgSend(exemption -> exemption), MsgMultiSend(exemption -> normal, exemption -> exemption)",
			msgSigner: privs[0],
			msgCreator: func() []sdk.Msg {
				var msgs []sdk.Msg

				msg1 := banktypes.NewMsgSend(addrs[0], addrs[1], sdk.NewCoins(sendCoin))
				msgs = append(msgs, msg1)
				msg2 := banktypes.NewMsgMultiSend(
					[]banktypes.Input{
						{
							Address: addrs[0].String(),
							Coins:   sdk.NewCoins(sendCoin),
						},
						{
							Address: addrs[0].String(),
							Coins:   sdk.NewCoins(sendCoin),
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
			// gasLimit * 28.325 = 8497500
			minFeeAmount: 8497500 + feeAmt*2,
		}, {
			name:      "MsgExecuteContract(exemption), MsgExecuteContract(normal)",
			msgSigner: privs[3],
			msgCreator: func() []sdk.Msg {
				sendAmount := int64(1000000)
				sendCoins := sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, sendAmount))
				// get wasm code for wasm contract create and instantiate
				wasmCode, err := os.ReadFile("./testdata/hackatom.wasm")
				s.Require().NoError(err)
				per := wasmkeeper.NewDefaultPermissionKeeper(s.app.WasmKeeper)
				// set wasm default params
				s.app.WasmKeeper.SetParams(s.ctx, wasmtypes.DefaultParams())
				// wasm create
				CodeID, _, err := per.Create(s.ctx, addrs[0], wasmCode, nil)
				s.Require().NoError(err)
				// params for contract init
				r := wasmkeeper.HackatomExampleInitMsg{Verifier: addrs[0], Beneficiary: addrs[0]}
				bz, err := json.Marshal(r)
				s.Require().NoError(err)
				// change block time for contract instantiate
				s.ctx = s.ctx.WithBlockTime(time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC))
				// instantiate contract then set the contract address to tax exemption
				addr, _, err := per.Instantiate(s.ctx, CodeID, addrs[0], nil, bz, "my label", nil)
				s.Require().NoError(err)
				s.app.TreasuryKeeper.AddBurnTaxExemptionAddress(s.ctx, addr.String())
				// instantiate contract then not set to tax exemption
				addr1, _, err := per.Instantiate(s.ctx, CodeID, addrs[0], nil, bz, "my label", nil)
				s.Require().NoError(err)

				var msgs []sdk.Msg
				// msg and signatures
				msg1 := &wasmtypes.MsgExecuteContract{
					Sender:   addrs[0].String(),
					Contract: addr.String(),
					Msg:      []byte{},
					Funds:    sendCoins,
				}
				msgs = append(msgs, msg1)

				msg2 := &wasmtypes.MsgExecuteContract{
					Sender:   addrs[3].String(),
					Contract: addr1.String(),
					Msg:      []byte{},
					Funds:    sendCoins,
				}
				msgs = append(msgs, msg2)
				return msgs
			},
			// gasLimit*28.325 = 33990000
			minFeeAmount: 33990000 + feeAmt,
			gasLimit:     1200000,
		},
	}

	// there should be no coin in burn module
	for _, c := range cases {
		s.SetupTest(true) // setup
		require := s.Require()
		tk := s.app.TreasuryKeeper
		burnSplitRate := sdk.NewDecWithPrec(5, 1)

		// Set burn split rate to 50%
		tk.SetBurnSplitRate(s.ctx, burnSplitRate)

		fmt.Printf("CASE = %s \n", c.name)
		s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

		tk.AddBurnTaxExemptionAddress(s.ctx, addrs[0].String())
		tk.AddBurnTaxExemptionAddress(s.ctx, addrs[1].String())

		mfd := ante.NewFeeDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, s.app.TreasuryKeeper, s.app.Tax2gasKeeper)
		antehandler := sdk.ChainAnteDecorators(mfd)

		for i := 0; i < 4; i++ {
			coins := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(100000000000)))
			testutil.FundAccount(s.app.BankKeeper, s.ctx, addrs[i], coins)
		}

		// msg and signatures
		feeAmount := sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, c.minFeeAmount))
		gasLimit := uint64(300000)
		if c.gasLimit != 0 {
			gasLimit = c.gasLimit
		}
		require.NoError(s.txBuilder.SetMsgs(c.msgCreator()...))
		s.txBuilder.SetFeeAmount(feeAmount)
		s.txBuilder.SetGasLimit(gasLimit)

		privs, accNums, accSeqs := []cryptotypes.PrivKey{c.msgSigner}, []uint64{0}, []uint64{0}
		tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
		require.NoError(err)

		_, err = antehandler(s.ctx, tx, false)
		require.NoError(err)
	}
}
