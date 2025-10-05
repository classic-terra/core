package ante_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/classic-terra/core/v3/app"
	appparams "github.com/classic-terra/core/v3/app/params"
	apptesting "github.com/classic-terra/core/v3/app/testing"
	core "github.com/classic-terra/core/v3/types"
	dyncommante "github.com/classic-terra/core/v3/x/dyncomm/ante"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/suite"
)

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	apptesting.KeeperTestHelper

	clientCtx client.Context
	txBuilder client.TxBuilder
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (suite *AnteTestSuite) SetupTest() {
	suite.Setup(suite.T(), apptesting.SimAppChainID)

	// Set up TxConfig.
	encodingConfig := suite.SetupEncoding()

	suite.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig)
}

func (suite *AnteTestSuite) SetupEncoding() appparams.EncodingConfig {
	encodingConfig := app.MakeEncodingConfig()
	// We're using TestMsg encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)

	return encodingConfig
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (suite *AnteTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signing.SignMode(suite.clientCtx.TxConfig.SignModeHandler().DefaultMode()),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err := suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			context.Background(), signing.SignMode(suite.clientCtx.TxConfig.SignModeHandler().DefaultMode()), signerData,
			suite.txBuilder, priv, suite.clientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func (suite *AnteTestSuite) CreateValidator(tokens int64) (cryptotypes.PrivKey, cryptotypes.PubKey, stakingtypes.Validator, authtypes.AccountI) {
	// Create a new account and fund it
	priv, pub, addr := testdata.KeyTestPubAddr()
	_, valPub, _ := suite.Ed25519PubAddr()
	valAddr := sdk.ValAddress(addr)

	account := suite.App.AccountKeeper.GetAccount(suite.Ctx, addr)
	if account == nil {
		base := authtypes.NewBaseAccountWithAddress(addr)
		base.SetPubKey(pub)
		account = suite.App.AccountKeeper.NewAccount(suite.Ctx, base)
		suite.App.AccountKeeper.SetAccount(suite.Ctx, account)
	}

	// Fund after account creation to avoid implicit zero account numbers
	sendCoins := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(2*tokens)))
	suite.FundAcc(addr, sendCoins)

	// Build MsgCreateValidator
	commissionRates := stakingtypes.NewCommissionRates(
		sdkmath.LegacyNewDecWithPrec(1, 2), sdkmath.LegacyNewDecWithPrec(1, 0),
		sdkmath.LegacyNewDecWithPrec(1, 0),
	)
	delegationCoin := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(tokens))
	desc := stakingtypes.NewDescription("moniker", "", "", "", "")
	msgCreateValidator, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		valPub,
		delegationCoin,
		desc,
		commissionRates,
		sdkmath.NewInt(tokens),
	)
	suite.Require().NoError(err)

	err = suite.txBuilder.SetMsgs(msgCreateValidator)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, suite.Ctx.ChainID())
	suite.Require().NoError(err)

	txBytes, err := suite.clientCtx.TxConfig.TxEncoder()(tx)
	suite.Require().NoError(err)

	// advance height/time
	nextHeight := suite.App.LastBlockHeight() + 1
	now := suite.Ctx.BlockTime()
	if now.IsZero() {
		now = time.Now()
	}
	suite.Ctx = suite.Ctx.WithBlockHeight(nextHeight).WithBlockTime(now)

	// run FinalizeBlock with the tx
	fb, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: nextHeight,
		Txs:    [][]byte{txBytes},
		Time:   now,
		// BlockTime is not a field in abci.RequestFinalizeBlock; set block time in context if needed
	})
	suite.Require().Len(fb.TxResults, 1)
	suite.Require().Equal(uint32(0), fb.TxResults[0].Code, fb.TxResults[0].Log)

	// commit
	suite.App.Commit()

	retval, err := suite.App.StakingKeeper.GetValidator(suite.Ctx, valAddr)
	suite.Require().NoError(err)

	updatedAccount := suite.App.AccountKeeper.GetAccount(suite.Ctx, addr)

	return priv, pub, retval, updatedAccount
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (suite *AnteTestSuite) TestAnte_EnsureDynCommissionIsMinComm() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.txBuilder.SetGasLimit(400_000)
	suite.Ctx = suite.Ctx.WithIsCheckTx(false)

	priv1, _, val1, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)
	suite.App.DyncommKeeper.UpdateAllBondedValidatorRates(suite.Ctx)

	mfd := dyncommante.NewDyncommDecorator(suite.App.AppCodec(), suite.App.DyncommKeeper, suite.App.StakingKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	dyncomm := suite.App.DyncommKeeper.CalculateDynCommission(suite.Ctx, val1)
	invalidtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(9, 1))
	validtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(11, 1))

	// invalid tx fails
	editmsg := stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &invalidtarget, &val1.MinSelfDelegation,
	)
	err := suite.txBuilder.SetMsgs(editmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().Error(err)

	// valid tx passes
	editmsg = stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &validtarget, &val1.MinSelfDelegation,
	)
	err = suite.txBuilder.SetMsgs(editmsg)
	suite.Require().NoError(err)
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().NoError(err)
}

func (suite *AnteTestSuite) TestAnte_EnsureDynCommissionIsMinCommAuthz() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.txBuilder.SetGasLimit(400_000)
	suite.Ctx = suite.Ctx.WithIsCheckTx(false)

	_, _, val1, _ := suite.CreateValidator(50_000_000_000)
	priv2, _, acc2 := testdata.KeyTestPubAddr()
	suite.CreateValidator(50_000_000_000)
	suite.App.DyncommKeeper.UpdateAllBondedValidatorRates(suite.Ctx)

	mfd := dyncommante.NewDyncommDecorator(suite.App.AppCodec(), suite.App.DyncommKeeper, suite.App.StakingKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	dyncomm := suite.App.DyncommKeeper.CalculateDynCommission(suite.Ctx, val1)
	invalidtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(9, 1))
	validtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(11, 1))

	// invalid tx fails
	editmsg := stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &invalidtarget, &val1.MinSelfDelegation,
	)

	execmsg := authz.NewMsgExec(acc2, []sdk.Msg{editmsg})

	err := suite.txBuilder.SetMsgs(&execmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().Error(err)

	// valid tx passes
	editmsg = stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &validtarget, &val1.MinSelfDelegation,
	)
	execmsg = authz.NewMsgExec(acc2, []sdk.Msg{editmsg})

	err = suite.txBuilder.SetMsgs(editmsg)
	suite.Require().NoError(err)
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().NoError(err)
}

func (suite *AnteTestSuite) TestAnte_EnsureDynCommissionIsMinCommICA() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.txBuilder.SetGasLimit(400_000)
	suite.Ctx = suite.Ctx.WithIsCheckTx(false)

	_, _, val1, _ := suite.CreateValidator(50_000_000_000)
	priv2, _, _ := testdata.KeyTestPubAddr()
	suite.CreateValidator(50_000_000_000)
	suite.App.DyncommKeeper.UpdateAllBondedValidatorRates(suite.Ctx)

	mfd := dyncommante.NewDyncommDecorator(suite.App.AppCodec(), suite.App.DyncommKeeper, suite.App.StakingKeeper)
	antehandler := sdk.ChainAnteDecorators(mfd)

	dyncomm := suite.App.DyncommKeeper.CalculateDynCommission(suite.Ctx, val1)
	invalidtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(9, 1))
	validtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(11, 1))

	// prepare invalid tx -> expect it fails
	editmsg := stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &invalidtarget, &val1.MinSelfDelegation,
	)
	data, err := icatypes.SerializeCosmosTx(suite.App.AppCodec(), []proto.Message{editmsg}, "proto3")
	suite.Require().NoError(err)
	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}
	packetData := icaPacketData.GetBytes()
	packet := channeltypes.NewPacket(
		packetData, 1, "source-port", "source-channel",
		"dest-port", "dest-channel",
		clienttypes.NewHeight(1, 1), 0,
	)
	recvmsg := channeltypes.NewMsgRecvPacket(
		packet, []byte{}, clienttypes.NewHeight(1, 1), "signer",
	)

	err = suite.txBuilder.SetMsgs(recvmsg)
	suite.Require().NoError(err)
	tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().Error(err)

	// prepare valid tx -> expect it passes
	editmsg = stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &validtarget, &val1.MinSelfDelegation,
	)
	data, err = icatypes.SerializeCosmosTx(suite.App.AppCodec(), []proto.Message{editmsg}, "proto3")
	suite.Require().NoError(err)
	icaPacketData = icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}
	packetData = icaPacketData.GetBytes()
	packet = channeltypes.NewPacket(
		packetData, 1, "source-port", "source-channel",
		"dest-port", "dest-channel",
		clienttypes.NewHeight(1, 1), 0,
	)
	recvmsg = channeltypes.NewMsgRecvPacket(
		packet, []byte{}, clienttypes.NewHeight(1, 1), "signer",
	)

	err = suite.txBuilder.SetMsgs(recvmsg)
	suite.Require().NoError(err)
	tx, err = suite.CreateTestTx([]cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}, suite.Ctx.ChainID())
	suite.Require().NoError(err)
	_, err = antehandler(suite.Ctx, tx, false)
	suite.Require().NoError(err)
}

// go test -v -run ^TestAnteTestSuite/TestAnte_EditValidatorAccountSequence$ github.com/classic-terra/core/v3/x/dyncomm/ante
// check that account keeper sequence no longer increases when editing validator unsuccessfully
func (suite *AnteTestSuite) TestAnte_EditValidatorAccountSequence() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.txBuilder.SetGasLimit(400_000)

	priv1, _, val1, acc := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	// Advance time by more than 24 hours to avoid commission change restriction
	suite.Ctx = suite.Ctx.WithBlockHeight(suite.Ctx.BlockHeight() + 1).WithBlockTime(suite.Ctx.BlockTime().Add(25 * time.Hour))
	_, _ = suite.App.BeginBlocker(suite.Ctx)
	// refresh deliver ctx for this height
	suite.Ctx = suite.App.BaseApp.NewUncachedContext(false, suite.Ctx.BlockHeader())

	// Update validator rates after time advancement
	suite.App.DyncommKeeper.UpdateAllBondedValidatorRates(suite.Ctx)

	dyncomm := suite.App.DyncommKeeper.CalculateDynCommission(suite.Ctx, val1)
	invalidtarget := dyncomm.Mul(sdkmath.LegacyNewDecWithPrec(9, 1))

	// invalid tx fails, not updating account sequence in account keeper
	editmsg := stakingtypes.NewMsgEditValidator(
		val1.GetOperator(),
		val1.Description, &invalidtarget, nil, // Set MinSelfDelegation to nil to avoid changing it
	)

	err := suite.txBuilder.SetMsgs(editmsg)
	suite.Require().NoError(err)

	// due to submitting a create validator tx before, thus account sequence is now 1
	for i := 0; i < 5; i++ {
		suite.Ctx = suite.Ctx.WithBlockHeight(suite.Ctx.BlockHeight() + 1)
		_, _ = suite.App.BeginBlocker(suite.Ctx)
		// refresh deliver ctx for this height
		suite.Ctx = suite.App.BaseApp.NewUncachedContext(false, suite.Ctx.BlockHeader())

		tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv1}, []uint64{acc.GetAccountNumber()}, []uint64{acc.GetSequence()}, suite.Ctx.ChainID())
		suite.Require().NoError(err)

		_, checkRes, err := suite.App.SimCheck(suite.clientCtx.TxConfig.TxEncoder(), tx)
		fmt.Printf("check response: %+v, error = %v \n", checkRes, err)
		suite.Ctx = suite.Ctx.WithIsCheckTx(false)

		txBytes, err := suite.clientCtx.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		// advance height/time
		nextHeight := suite.App.LastBlockHeight() + 1
		now := suite.Ctx.BlockTime()
		if now.IsZero() {
			now = time.Now()
		}
		suite.Ctx = suite.Ctx.WithBlockHeight(nextHeight).WithBlockTime(now)

		// run FinalizeBlock with the tx
		fb, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: nextHeight,
			Txs:    [][]byte{txBytes},
			Time:   now,
			// BlockTime is not a field in abci.RequestFinalizeBlock; set block time in context if needed
		})
		suite.Require().Len(fb.TxResults, 1)
		suite.Require().NotEqual(uint32(0), fb.TxResults[0].Code, "Transaction should fail due to commission validation")
		suite.Require().Contains(fb.TxResults[0].Log, "commission for")

		// commit
		suite.App.Commit()

		// check and update account keeper
		acc = suite.App.AccountKeeper.GetAccount(suite.CheckCtx, acc.GetAddress())
		checkSeq := acc.GetSequence()
		// checkSeq not updated when checkTx fails
		suite.Require().Equal(uint64(1), checkSeq)
		acc = suite.App.AccountKeeper.GetAccount(suite.Ctx, acc.GetAddress())
		deliverSeq := acc.GetSequence()
		// deliverSeq not updated when deliverTx fails
		suite.Require().Equal(uint64(1), deliverSeq)
	}
}
