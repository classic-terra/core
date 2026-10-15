package app_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/x/feegrant"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	terraapp "github.com/classic-terra/core/v4/app"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

const SimAppChainID = "simulation-app"

func init() {
	simcli.GetSimulatorFlags()
	// A reproducible default; an explicit -GenesisTime still overrides it.
	simcli.FlagGenesisTimeValue = 1700000000
	flag.Lookup("GenesisTime").DefValue = "1700000000"
}

func simulationConfig(tb testing.TB) simtypes.Config {
	tb.Helper()
	if !simcli.FlagEnabledValue {
		tb.Skip("application simulations require -Enabled=true")
	}
	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID
	return config
}

func newSimulationApp(tb testing.TB, config simtypes.Config) (*terraapp.TerraApp, dbm.DB, string, func()) {
	tb.Helper()
	db, dir, logger, skip, err := simtestutil.SetupSimulation(config, "terra-app-sim", "Simulation", simcli.FlagVerboseValue, true)
	require.NoError(tb, err)
	require.False(tb, skip)
	tb.Cleanup(func() { require.NoError(tb, db.Close()); require.NoError(tb, os.RemoveAll(dir)) })
	app, closeVM := simulationApp(tb, logger, db, dir, config)
	return app, db, dir, closeVM
}

func simulationApp(tb testing.TB, logger log.Logger, db dbm.DB, home string, config simtypes.Config) (*terraapp.TerraApp, func()) {
	options := simtestutil.AppOptionsMap{flags.FlagHome: home, server.FlagInvCheckPeriod: simcli.FlagPeriodValue}
	var closeVM func()
	var once sync.Once
	wasmOptions := []wasmkeeper.Option{wasmkeeper.WithWasmEngineDecorator(func(engine wasmtypes.WasmEngine) wasmtypes.WasmEngine {
		cleanup := engine.(interface{ Cleanup() })
		closeVM = func() { once.Do(cleanup.Cleanup) }
		return engine
	})}
	baseOptions := []func(*baseapp.BaseApp){baseapp.SetChainID(config.ChainID)}
	if simcli.FlagFauxMerkle {
		baseOptions = append(baseOptions, func(app *baseapp.BaseApp) { app.SetFauxMerkleMode() })
	}
	app := terraapp.NewTerraApp(logger, db, nil, true, map[int64]bool{}, home, terraapp.MakeEncodingConfig(), options, wasmOptions, baseOptions...)
	tb.Cleanup(closeVM)
	return app, closeVM
}

func runSimulation(tb testing.TB, app *terraapp.TerraApp, config simtypes.Config, state simtypes.AppStateFn) []simtypes.Account {
	tb.Helper()
	defer func() {
		if tb.Skipped() {
			tb.Errorf("SDK simulation skipped before executing all requested blocks")
		}
	}()
	// The SDK skips an empty genesis validator set. Treat it as a failed fixture
	// so CI cannot report a successful simulation without executing any blocks.
	checkedState := func(r *rand.Rand, accounts []simtypes.Account, cfg simtypes.Config) (json.RawMessage, []simtypes.Account, string, time.Time) {
		raw, accounts, chainID, timestamp := state(r, accounts, cfg)
		var genesis map[string]json.RawMessage
		require.NoError(tb, json.Unmarshal(raw, &genesis))
		var staking stakingtypes.GenesisState
		require.NoError(tb, app.AppCodec().UnmarshalJSON(genesis[stakingtypes.ModuleName], &staking))
		fundedValidator := false
		for _, validator := range staking.Validators {
			fundedValidator = fundedValidator || !validator.Jailed && validator.Tokens.IsPositive()
		}
		require.True(tb, fundedValidator, "simulation genesis must contain a funded, unjailed validator")
		return raw, accounts, chainID, timestamp
	}
	params, accounts, err := simulation.SimulateFromSeedX(tb, log.NewTestLogger(tb), os.Stdout,
		app.BaseApp, checkedState, simtypes.RandomAccounts,
		simtestutil.BuildSimulationOperations(app, app.AppCodec(), config, app.GetTxConfig()),
		app.ModuleAccountAddrs(), config, app.AppCodec(), &simulation.StandardLogWriter{Seed: config.Seed})
	require.NoError(tb, simtestutil.CheckExportSimulation(app, config, params))
	require.NoError(tb, err)
	if config.Commit {
		firstHeight := config.InitialBlockHeight
		if firstHeight == 0 {
			firstHeight = 1
		}
		require.Equal(tb, int64(firstHeight+config.NumBlocks-1), app.LastBlockHeight(), "simulation must execute every requested block")
	}
	return accounts
}

func TestFullAppSimulation(t *testing.T) {
	config := simulationConfig(t)
	app, _, _, _ := newSimulationApp(t, config)
	require.Equal(t, "TerraApp", app.Name())
	runSimulation(t, app, config, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()))
}

// These fixtures deliberately exhaust the validator set. They establish the
// terminal behavior and committed heights, and do not count as 50-block coverage.
func TestAppSimulationValidatorExhaustion(t *testing.T) {
	config := simulationConfig(t)
	require.True(t, config.Commit, "terminal-state checks require -Commit=true")
	require.Empty(t, config.GenesisFile, "terminal fixtures use randomized genesis")
	require.Empty(t, config.ParamsFile, "terminal fixtures use default parameters")
	config.InitialBlockHeight = 1
	config.NumBlocks = 50
	for _, fixture := range []struct{ seed, lastHeight int64 }{{582, 34}, {7601778, 27}, {89182391, 48}} {
		t.Run(fmt.Sprint(fixture.seed), func(t *testing.T) {
			config := config
			config.Seed = fixture.seed
			app, _, _, _ := newSimulationApp(t, config)
			var skipped bool
			// The SDK reports validator exhaustion through testing.T.Skip. Keep
			// that outcome inside a child; the parent verifies its terminal state.
			ok := t.Run("sdk-terminal", func(t *testing.T) {
				defer func() { skipped = t.Skipped() }()
				_, _, err := simulation.SimulateFromSeedX(t, log.NewTestLogger(t), os.Stdout,
					app.BaseApp, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
					simtypes.RandomAccounts, simtestutil.BuildSimulationOperations(app, app.AppCodec(), config, app.GetTxConfig()),
					app.ModuleAccountAddrs(), config, app.AppCodec(), &simulation.StandardLogWriter{Seed: config.Seed})
				require.NoError(t, err)
			})
			require.True(t, ok, "terminal simulation must not fail or panic")
			require.True(t, skipped, "SDK must report the expected terminal validator state")
			require.Equal(t, fixture.lastHeight, app.LastBlockHeight())
			validators, err := app.StakingKeeper.GetLastValidators(app.NewContext(true))
			require.NoError(t, err)
			require.Empty(t, validators, "terminal state must contain no active validators")
			t.Logf("validator exhaustion after %d/%d committed blocks; excluded from full-block coverage", app.LastBlockHeight(), config.NumBlocks)
		})
	}
}

func BenchmarkFullAppSimulation(b *testing.B) {
	config := simulationConfig(b)
	app, _, _, _ := newSimulationApp(b, config)
	runSimulation(b, app, config, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()))
}

func TestAppStateDeterminism(t *testing.T) {
	config := simulationConfig(t)
	require.True(t, config.Commit, "determinism requires -Commit=true")
	seeds := []int64{config.Seed}
	if config.Seed == simcli.DefaultSeedValue {
		seeds = []int64{42, 7, 99}
	}
	for _, seed := range seeds {
		config.Seed = seed
		var hash []byte
		for attempt := 0; attempt < 3; attempt++ {
			app, _, _, _ := newSimulationApp(t, config)
			runSimulation(t, app, config, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()))
			current := app.LastCommitID().Hash
			require.NotEmpty(t, current)
			if attempt == 0 {
				hash = append([]byte(nil), current...)
			} else {
				require.Equal(t, hash, current, "seed %d attempt %d", seed, attempt)
			}
		}
	}
}

func TestAppImportExport(t *testing.T) {
	config := simulationConfig(t)
	app, _, _, _ := newSimulationApp(t, config)
	runSimulation(t, app, config, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()))
	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	imported, _, _, _ := newSimulationApp(t, config)
	resumeTime := exportedUSTCGenesisTime(t, app, exported.AppState)
	_, err = imported.InitChain(&abci.RequestInitChain{ChainId: config.ChainID, Time: resumeTime, InitialHeight: exported.Height, AppStateBytes: exported.AppState, ConsensusParams: &exported.ConsensusParams})
	require.NoError(t, err)

	// InitChain writes its genesis into finalize state, before the first block.
	// Comparing stores here avoids reading the still-empty CheckTx state.
	originalContext := app.NewContext(true)
	importedContext := imported.NewContext(false)
	keys := app.GetKVStoreKey()
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var ignored [][]byte
		if name == stakingtypes.StoreKey {
			// SDK recreates queue order, historical headers and unbonding indexes on import.
			ignored = [][]byte{stakingtypes.UnbondingQueueKey, stakingtypes.RedelegationQueueKey, stakingtypes.ValidatorQueueKey, stakingtypes.HistoricalInfoKey, stakingtypes.UnbondingIDKey, stakingtypes.UnbondingIndexKey, stakingtypes.UnbondingTypeKey, stakingtypes.ValidatorUpdatesKey}
		}
		if name == authzkeeper.StoreKey {
			// Genesis contains live grants; the SDK reconstructs the expiration queue.
			ignored = [][]byte{authzkeeper.GrantQueuePrefix}
		}
		if name == feegrant.StoreKey {
			// Live allowances are exported; their expiration queue is rebuilt on import.
			ignored = [][]byte{feegrant.FeeAllowanceQueueKeyPrefix}
		}
		if name == wasmtypes.StoreKey {
			// The per-block transaction counter is runtime metadata, absent from wasm genesis.
			ignored = [][]byte{wasmtypes.TXCounterPrefix}
		}
		before, after := simtestutil.DiffKVStores(originalContext.KVStore(keys[name]), importedContext.KVStore(imported.GetKVStoreKey()[name]), ignored)
		require.Empty(t, after, "imported store %s contains unexpected data", name)
		require.Empty(t, before, "module %s: %s", name, simtestutil.GetSimulationLog(name, app.SimulationManager().StoreDecoders, before, after))
	}

	// Ordinary exports must resume at their exported height. In particular,
	// distribution starting heights must never be ahead of the imported chain.
	require.Equal(t, app.LastBlockHeight()+1, exported.Height)
	require.NotEmpty(t, exported.Validators)
	_, err = imported.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: exported.Height, Time: resumeTime.Add(time.Second),
		ProposerAddress: exported.Validators[0].Address,
	})
	require.NoError(t, err)
	_, err = imported.Commit()
	require.NoError(t, err)
	require.Equal(t, exported.Height, imported.LastBlockHeight())
	require.NotEmpty(t, imported.LastCommitID().Hash)
}

func TestAppSimulationAfterImport(t *testing.T) {
	config := simulationConfig(t)
	require.True(t, config.Commit, "restart simulation requires -Commit=true")
	app, _, _, _ := newSimulationApp(t, config)
	accounts := runSimulation(t, app, config, simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()))
	exported, err := app.ExportAppStateAndValidators(true, nil, nil)
	require.NoError(t, err)
	require.Zero(t, exported.Height, "SDK simulation initializes at height 1 and requires zero-height genesis")
	// A simulator always calls InitChain at height 0, then produces block 1.
	// Ordinary exports retain historical heights and cannot be restarted here.
	var modules map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported.AppState, &modules))
	var distribution distributiontypes.GenesisState
	require.NoError(t, app.AppCodec().UnmarshalJSON(modules[distributiontypes.ModuleName], &distribution))
	require.Empty(t, distribution.ValidatorSlashEvents)
	for _, starting := range distribution.DelegatorStartingInfos {
		require.Zero(t, starting.StartingInfo.Height, "zero-height export must reset distribution history")
	}

	imported, db, dir, closeVM := newSimulationApp(t, config)
	config.InitialBlockHeight = 1
	resumeTime := exportedUSTCGenesisTime(t, app, exported.AppState)
	state := func(_ *rand.Rand, _ []simtypes.Account, _ simtypes.Config) (json.RawMessage, []simtypes.Account, string, time.Time) {
		return exported.AppState, accounts, config.ChainID, resumeTime
	}
	// SDK v0.53 initializes the app itself. Supplying exported genesis here avoids double InitChain.
	runSimulation(t, imported, config, state)
	hash := append([]byte(nil), imported.LastCommitID().Hash...)
	height := imported.LastBlockHeight()
	closeVM() // Release the real Wasm VM lock before opening the same home directory.
	restarted, _ := simulationApp(t, log.NewNopLogger(), db, dir, config)
	require.Equal(t, height, restarted.LastBlockHeight())
	require.Equal(t, hash, restarted.LastCommitID().Hash)
	_, err = restarted.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
}

func exportedUSTCGenesisTime(tb testing.TB, app *terraapp.TerraApp, appState json.RawMessage) time.Time {
	tb.Helper()
	var modules map[string]json.RawMessage
	require.NoError(tb, json.Unmarshal(appState, &modules))
	var genesis ustcstakingtypes.GenesisState
	require.NoError(tb, app.AppCodec().UnmarshalJSON(modules[ustcstakingtypes.ModuleName], &genesis))
	require.NotNil(tb, genesis.RewardState.LastRewardTime, "streaming export must contain its consensus settlement time")
	return *genesis.RewardState.LastRewardTime
}
