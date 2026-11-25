package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/x/feegrant"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	terraapp "github.com/classic-terra/core/v4/app"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimAppChainID hardcoded chainID for simulation
const SimAppChainID = "simulation-app"

var emptyWasmOpts []wasmkeeper.Option

// Register the simulator flags (-Enabled, -NumBlocks, ...) used by sims.mk.
func init() {
	simcli.GetSimulatorFlags()
}

// fauxMerkleModeOpt returns a BaseApp option to use a dbStoreAdapter instead of
// an IAVLStore for faster simulation speed.
func fauxMerkleModeOpt(bapp *baseapp.BaseApp) {
	bapp.SetFauxMerkleMode()
}

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

func newSimApp(logger log.Logger, db dbm.DB, appOptions simtestutil.AppOptionsMap, baseAppOptions ...func(*baseapp.BaseApp)) *terraapp.TerraApp {
	baseAppOptions = append(baseAppOptions, baseapp.SetChainID(SimAppChainID))
	return terraapp.NewTerraApp(
		logger, db, nil, true, map[int64]bool{},
		cast(appOptions[flags.FlagHome]), terraapp.MakeEncodingConfig(),
		appOptions, emptyWasmOpts, baseAppOptions...,
	)
}

func cast(v interface{}) string {
	s, _ := v.(string)
	return s
}

func setupSimulationApp(t *testing.T, msg string) (simtypes.Config, dbm.DB, simtestutil.AppOptionsMap, *terraapp.TerraApp) {
	t.Helper()
	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID

	db, dir, logger, skip, err := simtestutil.SetupSimulation(config, "leveldb-app-sim", "Simulation", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	if skip {
		t.Skip(msg)
	}
	require.NoError(t, err, "simulation setup failed")

	t.Cleanup(func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	})

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = dir // ensure a unique folder
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	app := newSimApp(logger, db, appOptions, fauxMerkleModeOpt)
	return config, db, appOptions, app
}

func runSim(t *testing.T, app *terraapp.TerraApp, config simtypes.Config) (bool, *simulation.Params, error) {
	t.Helper()
	stopEarly, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		app.BaseApp,
		simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
		simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		simtestutil.SimulationOperations(app, app.AppCodec(), config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)
	return stopEarly, &simParams, simErr
}

func TestFullAppSimulation(t *testing.T) {
	config, db, _, app := setupSimulationApp(t, "skipping application simulation")

	_, simParams, simErr := runSim(t, app, config)

	// export state and simParams before the simulation error is checked
	require.NoError(t, simtestutil.CheckExportSimulation(app, config, *simParams))
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}

func TestAppImportExport(t *testing.T) {
	config, db, appOptions, app := setupSimulationApp(t, "skipping application import/export simulation")

	_, simParams, simErr := runSim(t, app, config)

	require.NoError(t, simtestutil.CheckExportSimulation(app, config, *simParams))
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	t.Log("exporting genesis...")
	exported, err := app.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)

	t.Log("importing genesis...")
	newDB, newDir, _, _, err := simtestutil.SetupSimulation(config, "leveldb-app-sim-2", "Simulation-2", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	require.NoError(t, err, "simulation setup failed")
	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	appOptions[flags.FlagHome] = t.TempDir() // ensure a unique folder for the new app
	newApp := newSimApp(log.NewNopLogger(), newDB, appOptions, fauxMerkleModeOpt)

	ctxA := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	_, err = newApp.InitChainer(ctxB, &abci.RequestInitChain{AppStateBytes: exported.AppState})
	if err != nil && strings.Contains(err.Error(), "validator set is empty after InitGenesis") {
		t.Log("Skipping simulation as all validators have been unbonded")
		t.Logf("err: %s stacktrace: %s\n", err, string(debug.Stack()))
		return
	}
	require.NoError(t, err)
	require.NoError(t, newApp.StoreConsensusParams(ctxB, exported.ConsensusParams))

	t.Log("comparing stores...")
	// skip prefixes that are legitimately not carried over by export/import
	skipPrefixes := map[string][][]byte{
		stakingtypes.StoreKey: {
			stakingtypes.UnbondingQueueKey, stakingtypes.RedelegationQueueKey, stakingtypes.ValidatorQueueKey,
			stakingtypes.HistoricalInfoKey, stakingtypes.UnbondingIDKey, stakingtypes.UnbondingIndexKey,
			stakingtypes.UnbondingTypeKey, stakingtypes.ValidatorUpdatesKey,
		},
		authzkeeper.StoreKey:   {authzkeeper.GrantQueuePrefix},
		feegrant.StoreKey:      {feegrant.FeeAllowanceQueueKeyPrefix},
		slashingtypes.StoreKey: {slashingtypes.ValidatorMissedBlockBitmapKeyPrefix},
		wasmtypes.StoreKey:     {wasmtypes.TXCounterPrefix},
	}

	storeKeys := app.GetKVStoreKey()
	require.NotEmpty(t, storeKeys)

	for keyName, appKeyA := range storeKeys {
		appKeyB := newApp.GetKey(keyName)

		storeA := ctxA.KVStore(appKeyA)
		storeB := ctxB.KVStore(appKeyB)

		failedKVAs, failedKVBs := simtestutil.DiffKVStores(storeA, storeB, skipPrefixes[keyName])
		if !assert.Equal(t, len(failedKVAs), len(failedKVBs), "unequal sets of key-values to compare in %q", keyName) {
			for _, v := range failedKVBs {
				t.Logf("store mismatch: %q\n", v)
			}
			t.FailNow()
		}

		t.Logf("compared %d different key/value pairs between %s and %s\n", len(failedKVAs), appKeyA, appKeyB)
		if !assert.Equal(t, 0, len(failedKVAs), simtestutil.GetSimulationLog(keyName, app.SimulationManager().StoreDecoders, failedKVAs, failedKVBs)) {
			for _, v := range failedKVAs {
				t.Logf("store mismatch: %q\n", v)
			}
			t.FailNow()
		}
	}
}

func TestAppSimulationAfterImport(t *testing.T) {
	config, db, appOptions, app := setupSimulationApp(t, "skipping application simulation after import")

	stopEarly, simParams, simErr := runSim(t, app, config)

	require.NoError(t, simtestutil.CheckExportSimulation(app, config, *simParams))
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	if stopEarly {
		t.Log("can't export or import a zero-validator genesis, exiting test...")
		return
	}

	t.Log("exporting genesis...")
	exported, err := app.ExportAppStateAndValidators(true, []string{}, []string{})
	require.NoError(t, err)

	t.Log("importing genesis...")
	newDB, newDir, _, _, err := simtestutil.SetupSimulation(config, "leveldb-app-sim-2", "Simulation-2", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	require.NoError(t, err, "simulation setup failed")
	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	appOptions[flags.FlagHome] = t.TempDir()
	newApp := newSimApp(log.NewNopLogger(), newDB, appOptions, fauxMerkleModeOpt)

	_, err = newApp.InitChain(&abci.RequestInitChain{
		ChainId:       SimAppChainID,
		AppStateBytes: exported.AppState,
	})
	require.NoError(t, err)

	_, _, err = runSim(t, newApp, config)
	require.NoError(t, err)
}

// TODO: Make another test for the fuzzer itself, which just has noOp txs
// and doesn't depend on the application.
func TestAppStateDeterminism(t *testing.T) {
	if !simcli.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := simcli.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 3
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	// override the random seed and run a single simulation on the provided seed value
	if config.Seed != simcli.DefaultSeedValue {
		numSeeds = 1
	}

	for i := 0; i < numSeeds; i++ {
		config.Seed += int64(i)
		for j := 0; j < numTimesToRunPerSeed; j++ {
			var logger log.Logger
			if simcli.FlagVerboseValue {
				logger = log.NewTestLogger(t)
			} else {
				logger = log.NewNopLogger()
			}

			appOptions := make(simtestutil.AppOptionsMap, 0)
			appOptions[flags.FlagHome] = t.TempDir() // ensure a unique folder per run
			appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

			db := dbm.NewMemDB()
			app := newSimApp(logger, db, appOptions, interBlockCacheOpt())

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err := runSim(t, app, config)
			require.NoError(t, err)

			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHashList[j] = app.LastCommitID().Hash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}
