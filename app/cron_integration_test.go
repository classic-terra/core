package app_test

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v4/app"
	appparams "github.com/classic-terra/core/v4/app/params"
	helpers "github.com/classic-terra/core/v4/app/testing"
	coretypes "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/cron/types"
	dyncommtypes "github.com/classic-terra/core/v4/x/dyncomm/types"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

type appCronWasmMsgServer struct {
	calls []*wasmtypes.MsgExecuteContract
}

func (s *appCronWasmMsgServer) ExecuteContract(_ context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	s.calls = append(s.calls, msg)
	return &wasmtypes.MsgExecuteContractResponse{}, nil
}

func initCronAppState(t *testing.T, terraApp *app.TerraApp, ctx sdk.Context) {
	t.Helper()

	terraApp.ConsensusParamsKeeper.ParamsStore.Set(ctx, *helpers.DefaultConsensusParams)
	terraApp.MintKeeper.Params.Set(ctx, minttypes.DefaultParams())
	terraApp.MintKeeper.Minter.Set(ctx, minttypes.DefaultInitialMinter())

	taxParams := taxtypes.DefaultParams()
	taxParams.GasPrices = sdk.NewDecCoins(sdk.NewDecCoin(coretypes.MicroSDRDenom, sdkmath.ZeroInt()))
	require.NoError(t, terraApp.TaxKeeper.SetParams(ctx, taxParams))

	bankParams := banktypes.DefaultParams()
	bankParams.DefaultSendEnabled = true
	require.NoError(t, terraApp.BankKeeper.SetParams(ctx, bankParams))
	terraApp.BankKeeper.SetSendEnabled(ctx, "uluna", true)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = appparams.BondDenom
	require.NoError(t, terraApp.StakingKeeper.SetParams(ctx, stakingParams))
	terraApp.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams())
	terraApp.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())
	terraApp.WasmKeeper.SetParams(ctx, wasmtypes.DefaultParams())
	terraApp.MarketKeeper.SetParams(ctx, markettypes.DefaultParams())
	terraApp.OracleKeeper.SetParams(ctx, oracletypes.DefaultParams())
	terraApp.TreasuryKeeper.SetParams(ctx, treasurytypes.DefaultParams())
	terraApp.DyncommKeeper.SetParams(ctx, dyncommtypes.DefaultParams())
}

type appBlockModule interface {
	BeginBlock(context.Context) ([]abci.ValidatorUpdate, error)
	EndBlock(context.Context) ([]abci.ValidatorUpdate, error)
}

func TestCronModuleWiredIntoApp(t *testing.T) {
	terraApp := helpers.SetupApp(t, "cron-test")

	require.Contains(t, terraApp.Modules(), types.ModuleName)
	require.NotNil(t, terraApp.CronKeeper.WasmMsgServer)
	require.True(t, terraApp.ModuleAccountAddrs()[authtypes.NewModuleAddress(types.ModuleName).String()])
}

func TestCronExecutesInBeginBlock(t *testing.T) {
	terraApp := helpers.SetupApp(t, "cron-test")
	fakeServer := &appCronWasmMsgServer{}
	terraApp.CronKeeper.WasmMsgServer = fakeServer

	ctx := terraApp.NewUncachedContext(false, tmproto.Header{ChainID: "cron-test"})
	ctx = ctx.WithBlockHeight(9).WithBlockTime(time.Now().UTC())
	initCronAppState(t, terraApp, ctx)

	require.NoError(t, terraApp.CronKeeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, terraApp.CronKeeper.AddSchedule(
		ctx,
		"begin-job",
		1,
		[]types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}},
		9,
		types.ExecutionStage_EXECUTION_STAGE_BEGIN_BLOCKER,
	))

	execCtx := terraApp.NewUncachedContext(false, tmproto.Header{ChainID: "cron-test"})
	execCtx = execCtx.WithBlockHeight(10).WithBlockTime(time.Now().UTC())
	require.Equal(t, int64(10), execCtx.BlockHeight())

	// Use the app-registered cron module instance so the test exercises the module
	// exactly as Terra wires it into the app.
	cronModule := terraApp.Modules()[types.ModuleName].(appBlockModule)
	_, err := cronModule.BeginBlock(sdk.WrapSDKContext(execCtx))
	require.NoError(t, err)
	require.Len(t, fakeServer.calls, 1)

	schedule, found := terraApp.CronKeeper.GetSchedule(execCtx, "begin-job")
	require.True(t, found)
	require.Equal(t, uint64(10), schedule.LastExecuteHeight)
}

func TestCronExecutesInEndBlock(t *testing.T) {
	terraApp := helpers.SetupApp(t, "cron-test")
	fakeServer := &appCronWasmMsgServer{}
	terraApp.CronKeeper.WasmMsgServer = fakeServer

	ctx := terraApp.NewUncachedContext(false, tmproto.Header{ChainID: "cron-test"})
	ctx = ctx.WithBlockHeight(10).WithBlockTime(time.Now().UTC())
	initCronAppState(t, terraApp, ctx)

	require.NoError(t, terraApp.CronKeeper.SetParams(ctx, types.NewParams(1)))
	require.NoError(t, terraApp.CronKeeper.AddSchedule(
		ctx,
		"end-job",
		1,
		[]types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"pong":{}}`}},
		10,
		types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER,
	))

	execCtx := terraApp.NewUncachedContext(false, tmproto.Header{ChainID: "cron-test"})
	execCtx = execCtx.WithBlockHeight(11).WithBlockTime(time.Now().UTC())
	require.Equal(t, int64(11), execCtx.BlockHeight())

	// Use the app-registered cron module instance so the test exercises the module
	// exactly as Terra wires it into the app.
	cronModule := terraApp.Modules()[types.ModuleName].(appBlockModule)
	_, err := cronModule.EndBlock(sdk.WrapSDKContext(execCtx))
	require.NoError(t, err)
	require.Len(t, fakeServer.calls, 1)

	schedule, found := terraApp.CronKeeper.GetSchedule(execCtx, "end-job")
	require.True(t, found)
	require.Equal(t, uint64(11), schedule.LastExecuteHeight)
}
