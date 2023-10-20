package ante_test

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v2/custom/auth/ante"
	markettypes "github.com/classic-terra/core/v2/x/market/types"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	"github.com/gogo/protobuf/proto"
)

func (s *AnteTestSuite) TestFreezeDecorator_BlockedBankSend() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(300_000)))
	send := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	bankmsg := banktypes.NewMsgSend(addr1, addr2, send)
	err := s.txBuilder.SetMsgs(bankmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedBankMultiSend() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(300_000)))
	send := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	bankmsg := banktypes.NewMsgMultiSend(
		[]banktypes.Input{banktypes.NewInput(addr1, send)},
		[]banktypes.Output{banktypes.NewOutput(addr2, send)},
	)
	err := s.txBuilder.SetMsgs(bankmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedSwapSend() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	send := sdk.NewCoin("uluna", sdk.NewInt(100_000))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	marketmsg := markettypes.NewMsgSwapSend(addr1, addr2, send, "uusd")
	err := s.txBuilder.SetMsgs(marketmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedSwap() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	send := sdk.NewCoin("uluna", sdk.NewInt(100_000))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	marketmsg := markettypes.NewMsgSwap(addr1, send, "uusd")
	err := s.txBuilder.SetMsgs(marketmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedInstantiateContract() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	send := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	wasmmsg := wasmtypes.MsgInstantiateContract{
		Sender: addr1.String(),
		Admin:  addr1.String(),
		CodeID: 1,
		Label:  "dummy",
		Msg:    []byte("{}"),
		Funds:  send,
	}
	err := s.txBuilder.SetMsgs(&wasmmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedMsgInstantiateContract2() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	send := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	wasmmsg := wasmtypes.MsgInstantiateContract2{
		Sender: addr1.String(),
		Admin:  addr1.String(),
		CodeID: 1,
		Label:  "dummy",
		Msg:    []byte("{}"),
		Funds:  send,
		Salt:   []byte("crytal"),
		FixMsg: true,
	}
	err := s.txBuilder.SetMsgs(&wasmmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedExecuteContract() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	send := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	wasmmsg := wasmtypes.MsgExecuteContract{
		Sender:   addr1.String(),
		Contract: "mycontractaddress",
		Msg:      []byte("{}"),
		Funds:    send,
	}
	err := s.txBuilder.SetMsgs(&wasmmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedMsgTransfer() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	sendcoin := sdk.NewCoin("uluna", sdk.NewInt(100_000))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	ibcmsg := ibctransfertypes.NewMsgTransfer(
		"transfer", "channel-0", sendcoin,
		addr1.String(), addr2.String(), clienttypes.NewHeight(1, 1), 0,
		"awesomeibc",
	)
	err := s.txBuilder.SetMsgs(ibcmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedAuthz() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, grantee := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	sendcoins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	wrappedmsg := banktypes.NewMsgSend(addr1, addr2, sendcoins)
	authzmsg := authztypes.NewMsgExec(grantee, []sdk.Msg{wrappedmsg})
	err := s.txBuilder.SetMsgs(&authzmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}

func (s *AnteTestSuite) TestFreezeDecorator_BlockedIbcHostMsg() {
	s.SetupTest(true) // setup
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	mfd := ante.NewFreezeDecorator(s.app.AppCodec(), s.app.ICAHostKeeper, s.app.TreasuryKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	coins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(300_000)))
	sendcoins := sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(100_000)))
	testutil.FundAccount(s.app.BankKeeper, s.ctx, addr1, coins)

	wrappedmsg := &banktypes.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sendcoins,
	}
	bz, err := icatypes.SerializeCosmosTx(s.app.AppCodec(), []proto.Message{wrappedmsg})
	s.Require().NoError(err)
	hostpacketdata := &icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "awsesomeinterchain",
	}
	bz2 := icatypes.ModuleCdc.MustMarshalJSON(hostpacketdata)
	packet := channeltypes.NewPacket(
		bz2, 1, "srcport", "srcchannel",
		"dstport", "dstchannel",
		clienttypes.NewHeight(1, 1), 0,
	)
	recvmsg := channeltypes.NewMsgRecvPacket(
		packet, []byte("proof"), clienttypes.NewHeight(1, 1),
		addr1.String(),
	)
	err = s.txBuilder.SetMsgs(recvmsg)
	s.Require().NoError(err)
	s.txBuilder.SetGasLimit(400_000)
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set blocked address
	blocked := treasurytypes.NewFreezeList()
	blocked.Add(addr1.String())
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must fail
	_, err = antehandler(s.ctx, tx, false)
	s.Require().Error(err)

	// set empty blocked addresses
	blocked = treasurytypes.NewFreezeList()
	s.app.TreasuryKeeper.SetFreezeAddrs(s.ctx, blocked)

	// must succeed
	_, err = antehandler(s.ctx, tx, false)
	s.Require().NoError(err)

}
