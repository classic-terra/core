package ante_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Regression test for #648: a tx that PrepareProposal dropped from the
// app-side mempool stayed in the CometBFT mempool and blocked its sender.

type stuckTxEnv struct {
	height int64
	time   time.Time
	// build signs a MsgEditValidator of the validator operator with the given
	// gas limit at the operator's current sequence plus seqOffset.
	build func(gas, seqOffset uint64) []byte
}

func (suite *AnteTestSuite) setupStuckTxEnv() stuckTxEnv {
	suite.SetupTest()
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
	suite.txBuilder.SetGasLimit(1_000_000)

	priv, _, val, _ := suite.CreateValidator(50_000_000_000)
	suite.CreateValidator(50_000_000_000)

	// advance > 24h so that the commission may change
	t := suite.Ctx.BlockTime().Add(25 * time.Hour)
	h := suite.App.LastBlockHeight() + 1
	_, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{Height: h, Time: t})
	suite.Require().NoError(err)
	_, err = suite.App.Commit()
	suite.Require().NoError(err)

	valAddr, err := sdk.ValAddressFromBech32(val.GetOperator())
	suite.Require().NoError(err)
	acc := suite.App.AccountKeeper.GetAccount(suite.App.NewUncachedContext(false, suite.Ctx.BlockHeader()), sdk.AccAddress(valAddr))

	rate := sdkmath.LegacyNewDecWithPrec(2, 1)
	msg := stakingtypes.NewMsgEditValidator(val.GetOperator(), stakingtypes.Description{Moniker: stakingtypes.DoNotModifyDesc}, &rate, nil)

	return stuckTxEnv{
		height: h,
		time:   t,
		build: func(gas, seqOffset uint64) []byte {
			suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
			suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
			suite.txBuilder.SetGasLimit(gas)
			tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{acc.GetAccountNumber()}, []uint64{acc.GetSequence() + seqOffset}, suite.Ctx.ChainID())
			suite.Require().NoError(err)
			bz, err := suite.clientCtx.TxConfig.TxEncoder()(tx)
			suite.Require().NoError(err)
			return bz
		},
	}
}

func (suite *AnteTestSuite) checkTx(bz []byte, typ abci.CheckTxType) *abci.ResponseCheckTx {
	res, err := suite.App.CheckTx(&abci.RequestCheckTx{Tx: bz, Type: typ})
	suite.Require().NoError(err)
	return res
}

// A tx that is no longer in the app-side mempool must be evicted on ReCheckTx
// instead of staying in the CometBFT mempool forever.
func (suite *AnteTestSuite) TestStuckTx_RecheckEvictsTxMissingFromAppMempool() {
	env := suite.setupStuckTxEnv()

	bz := env.build(1_000_000, 0)
	suite.Require().Equal(uint32(0), suite.checkTx(bz, abci.CheckTxType_New).Code)

	tx, err := suite.clientCtx.TxConfig.TxDecoder()(bz)
	suite.Require().NoError(err)

	// empty block, then CometBFT rechecks the still pending tx
	commitEmptyBlock := func(offset time.Duration) {
		_, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{Height: suite.App.LastBlockHeight() + 1, Time: env.time.Add(offset)})
		suite.Require().NoError(err)
		_, err = suite.App.Commit()
		suite.Require().NoError(err)
	}

	commitEmptyBlock(6 * time.Second)
	res := suite.checkTx(bz, abci.CheckTxType_Recheck)
	suite.Require().Equal(uint32(0), res.Code, res.Log)

	// PrepareProposal dropping the tx removes it from the app-side mempool only.
	suite.Require().NoError(suite.App.Mempool().Remove(tx))

	commitEmptyBlock(12 * time.Second)
	res = suite.checkTx(bz, abci.CheckTxType_Recheck)
	suite.Require().NotEqual(uint32(0), res.Code)
	suite.Require().Contains(res.Log, "no longer in the app-side mempool")
}
