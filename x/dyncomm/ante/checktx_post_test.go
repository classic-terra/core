package ante_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	core "github.com/classic-terra/core/v4/types"
	dyncommpost "github.com/classic-terra/core/v4/x/dyncomm/post"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// The dyncomm post handler only runs in FinalizeBlock and simulate, the modes
// that execute msgs. It used to run in PrepareProposal and ProcessProposal
// too, where a MsgEditValidator whose gas limit covered only the ante handler
// passed CheckTx but ran out of gas, so it was never proposed and stayed in
// the CometBFT mempool forever, blocking its sender (#648).

type checkTxPostEnv struct {
	height int64
	time   time.Time
	valOp  string
	addr   sdk.AccAddress
	// build signs msgs from the validator operator with the given gas limit at
	// the operator's current sequence plus seqOffset.
	build func(gas, seqOffset uint64, msgs ...sdk.Msg) []byte
}

func (suite *AnteTestSuite) setupCheckTxPostEnv() checkTxPostEnv {
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
	addr := sdk.AccAddress(valAddr)
	acc := suite.App.AccountKeeper.GetAccount(suite.App.NewUncachedContext(false, suite.Ctx.BlockHeader()), addr)

	return checkTxPostEnv{
		height: h,
		time:   t,
		valOp:  val.GetOperator(),
		addr:   addr,
		build: func(gas, seqOffset uint64, msgs ...sdk.Msg) []byte {
			suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()
			suite.Require().NoError(suite.txBuilder.SetMsgs(msgs...))
			suite.txBuilder.SetGasLimit(gas)
			tx, err := suite.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{acc.GetAccountNumber()}, []uint64{acc.GetSequence() + seqOffset}, suite.Ctx.ChainID())
			suite.Require().NoError(err)
			bz, err := suite.clientCtx.TxConfig.TxEncoder()(tx)
			suite.Require().NoError(err)
			return bz
		},
	}
}

func (suite *AnteTestSuite) editCommissionMsg(env checkTxPostEnv) sdk.Msg {
	rate := sdkmath.LegacyNewDecWithPrec(2, 1)
	return stakingtypes.NewMsgEditValidator(env.valOp, stakingtypes.Description{Moniker: stakingtypes.DoNotModifyDesc}, &rate, nil)
}

func (suite *AnteTestSuite) runCheckTx(bz []byte, typ abci.CheckTxType) *abci.ResponseCheckTx {
	res, err := suite.App.CheckTx(&abci.RequestCheckTx{Tx: bz, Type: typ})
	suite.Require().NoError(err)
	return res
}

// checkTxGas returns the gas CheckTx needs for msgs, measured on a fresh app
// because a successful CheckTx advances the check state.
func (suite *AnteTestSuite) checkTxGas(msgs func(checkTxPostEnv) []sdk.Msg) uint64 {
	env := suite.setupCheckTxPostEnv()
	res := suite.runCheckTx(env.build(1_000_000, 0, msgs(env)...), abci.CheckTxType_New)
	suite.Require().Equal(uint32(0), res.Code, res.Log)
	return uint64(res.GasUsed)
}

// Invariant: a gas limit that is exactly enough for CheckTx is also enough for
// PrepareProposal, and the resulting proposal is accepted.
func (suite *AnteTestSuite) TestCheckTxPost_CheckTxGasCoversProposal() {
	cases := map[string]func(checkTxPostEnv) []sdk.Msg{
		"edit validator with commission": func(env checkTxPostEnv) []sdk.Msg {
			return []sdk.Msg{suite.editCommissionMsg(env)}
		},
		"edit validator without commission": func(env checkTxPostEnv) []sdk.Msg {
			return []sdk.Msg{stakingtypes.NewMsgEditValidator(env.valOp, stakingtypes.NewDescription("renamed", "", "", "", ""), nil, nil)}
		},
		"bank send": func(env checkTxPostEnv) []sdk.Msg {
			return []sdk.Msg{banktypes.NewMsgSend(env.addr, env.addr, sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1)))}
		},
	}

	for name, msgs := range cases {
		suite.Run(name, func() {
			gas := suite.checkTxGas(msgs)

			env := suite.setupCheckTxPostEnv()
			bz := env.build(gas, 0, msgs(env)...)
			next := env.build(1_000_000, 1, msgs(env)...)
			res := suite.runCheckTx(bz, abci.CheckTxType_New)
			suite.Require().Equal(uint32(0), res.Code, res.Log)
			suite.Require().Equal(uint32(0), suite.runCheckTx(next, abci.CheckTxType_New).Code)

			pp, err := suite.App.PrepareProposal(&abci.RequestPrepareProposal{
				Height: env.height + 1, Time: env.time.Add(6 * time.Second), MaxTxBytes: 1 << 20, Txs: [][]byte{bz, next},
			})
			suite.Require().NoError(err)
			suite.Require().Equal([][]byte{bz, next}, pp.Txs)

			proc, err := suite.App.ProcessProposal(&abci.RequestProcessProposal{
				Height: env.height + 1, Time: env.time.Add(6 * time.Second), Txs: pp.Txs,
			})
			suite.Require().NoError(err)
			suite.Require().Equal(abci.ResponseProcessProposal_ACCEPT, proc.Status)
		})
	}
}

// A MsgEditValidator with exactly the CheckTx gas has too little gas for block
// execution. It must still be proposed and accepted, fail in FinalizeBlock and
// consume its sequence, so that the sender's next tx goes through.
func (suite *AnteTestSuite) TestCheckTxPost_TightGasTxIsIncludedAndFails() {
	msgs := func(env checkTxPostEnv) []sdk.Msg { return []sdk.Msg{suite.editCommissionMsg(env)} }
	gas := suite.checkTxGas(msgs)

	env := suite.setupCheckTxPostEnv()
	bz := env.build(gas, 0, msgs(env)...)
	next := env.build(1_000_000, 1, msgs(env)...)
	suite.Require().Equal(uint32(0), suite.runCheckTx(bz, abci.CheckTxType_New).Code)
	suite.Require().Equal(uint32(0), suite.runCheckTx(next, abci.CheckTxType_New).Code)

	blockTime := env.time.Add(6 * time.Second)
	pp, err := suite.App.PrepareProposal(&abci.RequestPrepareProposal{
		Height: env.height + 1, Time: blockTime, MaxTxBytes: 1 << 20, Txs: [][]byte{bz, next},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([][]byte{bz, next}, pp.Txs)

	proc, err := suite.App.ProcessProposal(&abci.RequestProcessProposal{Height: env.height + 1, Time: blockTime, Txs: pp.Txs})
	suite.Require().NoError(err)
	suite.Require().Equal(abci.ResponseProcessProposal_ACCEPT, proc.Status)

	fb, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{Height: env.height + 1, Time: blockTime, Txs: pp.Txs})
	suite.Require().NoError(err)
	suite.Require().Len(fb.TxResults, 2)
	suite.Require().NotEqual(uint32(0), fb.TxResults[0].Code)
	suite.Require().Contains(fb.TxResults[0].Log, "out of gas")
	suite.Require().Equal(uint32(0), fb.TxResults[1].Code, fb.TxResults[1].Log)
}

// The dyncomm post handler charges gas only in the modes that execute msgs,
// and the same in simulate as in FinalizeBlock, so that gas estimation
// (--gas auto) includes it.
func (suite *AnteTestSuite) TestCheckTxPost_PostGasByExecMode() {
	env := suite.setupCheckTxPostEnv()
	tx, err := suite.clientCtx.TxConfig.TxDecoder()(env.build(1_000_000, 0, suite.editCommissionMsg(env)))
	suite.Require().NoError(err)

	postGas := func(mode sdk.ExecMode) uint64 {
		ctx, _ := suite.App.NewUncachedContext(false, suite.Ctx.BlockHeader()).CacheContext()
		ctx = ctx.WithExecMode(mode).WithGasMeter(storetypes.NewInfiniteGasMeter())
		next := func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) { return ctx, nil }
		_, err := dyncommpost.NewDyncommPostDecorator(suite.App.DyncommKeeper).PostHandle(ctx, tx, mode == sdk.ExecModeSimulate, true, next)
		suite.Require().NoError(err)
		return ctx.GasMeter().GasConsumed()
	}

	finalize := postGas(sdk.ExecModeFinalize)
	suite.Require().Positive(finalize)
	suite.Require().Equal(finalize, postGas(sdk.ExecModeSimulate))
	for _, mode := range []sdk.ExecMode{sdk.ExecModeCheck, sdk.ExecModeReCheck, sdk.ExecModePrepareProposal, sdk.ExecModeProcessProposal} {
		suite.Require().Zero(postGas(mode), "exec mode %d", mode)
	}
}
