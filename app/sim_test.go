package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	terraapp "github.com/classic-terra/core/v2/app"
	helpers "github.com/classic-terra/core/v2/app/testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/rand"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/simapp"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
)

// SimAppChainID hardcoded chainID for simulation
const SimAppChainID = "simulation-app"

var emptyWasmOpts []wasmkeeper.Option

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

// fauxMerkleModeOpt is a BaseApp option function to enable faux Merkle Tree mode for faster sim speed
func fauxMerkleModeOpt() func(*baseapp.BaseApp) {
	return func(app *baseapp.BaseApp) { app.SetFauxMerkleMode() }
}

func init() {
	simcli.GetSimulatorFlags()
}

func setupSimulationApp(b *testing.B, msg string) (simtypes.Config, dbm.DB, simtestutil.AppOptionsMap, *terraapp.TerraApp) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID

	db, dir, logger, skip, err := simtestutil.SetupSimulation(config, "leveldb-app-sim", "Simulation", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	if skip {
		b.Skip(msg)
	}
	require.NoError(b, err, "simulation setup failed")

	b.Cleanup(func() {
		require.NoError(b, db.Close())
		require.NoError(b, os.RemoveAll(dir))
	})

	appOptions := make(simtestutil.AppOptionsMap, 0)

	app := terraapp.NewTerraApp(
		logger, db, nil, true, map[int64]bool{},
		dir, simcli.FlagPeriodValue, terraapp.MakeEncodingConfig(),
		appOptions, emptyWasmOpts, interBlockCacheOpt(), fauxMerkleModeOpt(), baseapp.SetChainID(SimAppChainID),
	)
	require.Equal(b, "WasmApp", app.Name())
	return config, db, appOptions, app
}

// Profile with:
// /usr/local/go/bin/go test -benchmem -run=^$ github.com/cosmos/cosmos-sdk/GaiaApp -bench ^BenchmarkFullAppSimulation$ -Commit=true -cpuprofile cpu.out
func BenchmarkFullAppSimulation(b *testing.B) {
	config, db, _, app := setupSimulationApp(b, "skipping application simulation")

	// Run randomized simulation:w
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		app.BaseApp,
		simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
		simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		simtestutil.SimulationOperations(app, app.AppCodec(), config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err := simtestutil.CheckExportSimulation(app, config, simParams); err != nil {
		b.Fatal(err)
	}

	if simErr != nil {
		b.Fatal(simErr)
	}

	if config.Commit {
		simtestutil.PrintStats(db)
	}
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
	config.ChainID = helpers.SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 3
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	for i := 0; i < numSeeds; i++ {
		config.Seed = rand.Int63()

		for j := 0; j < numTimesToRunPerSeed; j++ {
			var logger log.Logger
			if simcli.FlagVerboseValue {
				logger = log.TestingLogger()
			} else {
				logger = log.NewNopLogger()
			}

			db := dbm.NewMemDB()
			var emptyWasmOpts []wasm.Option
			app := terraapp.NewTerraApp(
				logger, db, nil, true, map[int64]bool{}, terraapp.DefaultNodeHome,
				simcli.FlagPeriodValue, terraapp.MakeEncodingConfig(),
				simtestutil.EmptyAppOptions{}, emptyWasmOpts, interBlockCacheOpt(), fauxMerkleModeOpt(),
			)

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				app.BaseApp,
				AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
				simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
				simtestutil.SimulationOperations(app, app.AppCodec(), config),
				app.ModuleAccountAddrs(),
				config,
				app.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHash := app.LastCommitID().Hash
			appHashList[j] = appHash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}

// AppStateFn returns the initial application state using a genesis or the simulation parameters.
// It panics if the user provides files for both of them.
// If a file is not given for the genesis or the sim params, it creates a randomized one.
func AppStateFn(codec codec.Codec, manager *module.SimulationManager, genesisState map[string]json.RawMessage) simtypes.AppStateFn {
	// quick hack to setup app state genesis with our app modules
	simapp.ModuleBasics = terraapp.ModuleBasics
	if simcli.FlagGenesisTimeValue == 0 { // always set to have a block time
		simcli.FlagGenesisTimeValue = time.Now().Unix()
	}
	return simtestutil.AppStateFn(codec, manager, genesisState)
}
