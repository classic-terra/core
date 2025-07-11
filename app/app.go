package app

import (
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	appmempool "github.com/classic-terra/core/v3/app/mempool"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/classic-terra/core/v3/app/keepers"
	terraappparams "github.com/classic-terra/core/v3/app/params"
	customserver "github.com/classic-terra/core/v3/server"

	// upgrades
	"github.com/classic-terra/core/v3/app/upgrades"
	v11_2 "github.com/classic-terra/core/v3/app/upgrades/v11_2"
	v2 "github.com/classic-terra/core/v3/app/upgrades/v2"
	v3 "github.com/classic-terra/core/v3/app/upgrades/v3"
	v4 "github.com/classic-terra/core/v3/app/upgrades/v4"
	v5 "github.com/classic-terra/core/v3/app/upgrades/v5"
	v6 "github.com/classic-terra/core/v3/app/upgrades/v6"
	v6_1 "github.com/classic-terra/core/v3/app/upgrades/v6_1"
	v7 "github.com/classic-terra/core/v3/app/upgrades/v7"
	v7_1 "github.com/classic-terra/core/v3/app/upgrades/v7_1"
	v8 "github.com/classic-terra/core/v3/app/upgrades/v8"
	v8_1 "github.com/classic-terra/core/v3/app/upgrades/v8_1"
	v8_2 "github.com/classic-terra/core/v3/app/upgrades/v8_2"
	v8_3 "github.com/classic-terra/core/v3/app/upgrades/v8_3"

	// v9 had been used by tax2gas and has to be skipped
	v10_1 "github.com/classic-terra/core/v3/app/upgrades/v10_1"
	v11 "github.com/classic-terra/core/v3/app/upgrades/v11"
	v11_1 "github.com/classic-terra/core/v3/app/upgrades/v11_1"
	v12 "github.com/classic-terra/core/v3/app/upgrades/v12"

	customante "github.com/classic-terra/core/v3/custom/auth/ante"
	custompost "github.com/classic-terra/core/v3/custom/auth/post"
	customauthtx "github.com/classic-terra/core/v3/custom/auth/tx"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/classic-terra/core/v3/client/docs/statik"
)

const appName = "TerraApp"

var (
	// DefaultNodeHome defines default home directories for terrad
	DefaultNodeHome string

	// Upgrades defines upgrades to be applied to the network
	Upgrades = []upgrades.Upgrade{
		v2.Upgrade,
		v3.Upgrade,
		v4.Upgrade,
		v5.Upgrade,
		v6.Upgrade,
		v6_1.Upgrade,
		v7.Upgrade,
		v7_1.Upgrade,
		v8.Upgrade,
		v8_1.Upgrade,
		v8_2.Upgrade,
		v8_3.Upgrade,
		v10_1.Upgrade,
		v11.Upgrade,
		v11_1.Upgrade,
		v11_2.Upgrade,
		v12.Upgrade,
	}

	// Forks defines forks to be applied to the network
	Forks = []upgrades.Fork{}
)

// Verify app interface at compile time
var (
	_ runtime.AppI            = (*TerraApp)(nil)
	_ servertypes.Application = (*TerraApp)(nil)
)

// TerraApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type TerraApp struct {
	*baseapp.BaseApp
	*keepers.AppKeepers

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	invCheckPeriod uint

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// the configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		stdlog.Println("Failed to get home dir %2", err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".terra")
}

// NewTerraApp returns a reference to an initialized TerraApp.
func NewTerraApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, skipUpgradeHeights map[int64]bool,
	homePath string, encodingConfig terraappparams.EncodingConfig, appOpts servertypes.AppOptions,
	wasmOpts []wasmkeeper.Option, baseAppOptions ...func(*baseapp.BaseApp),
) *TerraApp {
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	iavlCacheSize := cast.ToInt(appOpts.Get(server.FlagIAVLCacheSize))
	iavlDisableFastNode := cast.ToBool(appOpts.Get(server.FlagDisableIAVLFastNode))

	// option for cosmos sdk
	baseAppOptions = append(baseAppOptions, baseapp.SetIAVLCacheSize(iavlCacheSize))
	baseAppOptions = append(baseAppOptions, baseapp.SetIAVLDisableFastNode(iavlDisableFastNode))

	// option for mempool
	baseAppOptions = append(baseAppOptions, func(app *baseapp.BaseApp) {
		var mempool *appmempool.FifoMempool
		if maxTxs := cast.ToInt(appOpts.Get(server.FlagMempoolMaxTxs)); maxTxs > 0 {
			mempool = appmempool.NewFifoMempool(appmempool.FifoMaxTxOpt(maxTxs))
		} else {
			mempool = appmempool.NewFifoMempool()
		}
		handler := baseapp.NewDefaultProposalHandler(mempool, app)
		app.SetMempool(mempool)
		app.SetTxEncoder(txConfig.TxEncoder())
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	})

	bApp := baseapp.NewBaseApp(appName, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetCommitMultiStoreTracer(traceStore)

	app := &TerraApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          txConfig,
		invCheckPeriod:    invCheckPeriod,
	}

	// Setup keepers
	app.AppKeepers = keepers.NewAppKeepers(
		appCodec,
		bApp,
		legacyAmino,
		maccPerms,
		allowedReceivingModAcc,
		skipUpgradeHeights,
		homePath,
		invCheckPeriod,
		wasmOpts,
		appOpts,
	)

	/****  Module Options ****/
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(appModules(app, encodingConfig, skipGenesisInvariants)...)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(orderBeginBlockers()...)
	app.mm.SetOrderEndBlockers(orderEndBlockers()...)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	// NOTE: Treasury must occur after bank module so that initial supply is properly set
	app.mm.SetOrderInitGenesis(orderInitGenesis()...)

	app.mm.RegisterInvariants(app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)
	app.setupUpgradeHandlers()
	app.setupUpgradeStoreLoaders()

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	app.sm = module.NewSimulationManager(simulationModules(app, encodingConfig, skipGenesisInvariants)...)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(app.GetKVStoreKey())
	app.MountTransientStores(app.GetTransientStoreKey())
	app.MountMemoryStores(app.GetMemoryStoreKey())

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)

	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	anteHandler, err := customante.NewAnteHandler(
		customante.HandlerOptions{
			AccountKeeper:      app.AccountKeeper,
			BankKeeper:         app.BankKeeper,
			FeegrantKeeper:     app.FeeGrantKeeper,
			OracleKeeper:       app.OracleKeeper,
			TreasuryKeeper:     app.TreasuryKeeper,
			TaxExemptionKeeper: app.TaxExemptionKeeper,
			SigGasConsumer:     ante.DefaultSigVerificationGasConsumer,
			SignModeHandler:    encodingConfig.TxConfig.SignModeHandler(),
			IBCKeeper:          *app.IBCKeeper,
			WasmKeeper:         &app.WasmKeeper,
			DistributionKeeper: app.DistrKeeper,
			GovKeeper:          app.GovKeeper,
			WasmConfig:         &wasmConfig,
			TXCounterStoreKey:  app.GetKey(wasmtypes.StoreKey),
			DyncommKeeper:      app.DyncommKeeper,
			StakingKeeper:      app.StakingKeeper,
			TaxKeeper:          &app.TaxKeeper,
			Cdc:                app.appCodec,
		},
	)
	if err != nil {
		panic(err)
	}

	postHandler, err := custompost.NewPostHandler(
		custompost.HandlerOptions{
			DyncommKeeper:  app.DyncommKeeper,
			TaxKeeper:      app.TaxKeeper,
			BankKeeper:     app.BankKeeper,
			AccountKeeper:  app.AccountKeeper,
			TreasuryKeeper: app.TreasuryKeeper,
		},
	)
	if err != nil {
		panic(err)
	}

	app.SetAnteHandler(anteHandler)
	app.SetPostHandler(postHandler)
	app.SetEndBlocker(app.EndBlocker)

	// must be before Loading version
	// requires the snapshot store to be created and registered as a BaseAppOption
	// see cmd/wasmd/root.go: 206 - 214 approx
	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		)
		if err != nil {
			panic(fmt.Errorf("failed to register snapshot extension: %s", err))
		}
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}

		ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{})
		// Initialize pinned codes in wasmvm as they are not persisted there
		if err := app.WasmKeeper.InitializePinnedCodes(ctx); err != nil {
			tmos.Exit(fmt.Sprintf("failed initialize pinned codes %s", err))
		}
	}

	return app
}

// Name returns the name of the App
func (app *TerraApp) Name() string { return app.BaseApp.Name() }

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *TerraApp) DefaultGenesis() map[string]json.RawMessage {
	return ModuleBasics.DefaultGenesis(app.appCodec)
}

// BeginBlocker application updates every begin block
func (app *TerraApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	BeginBlockForks(ctx, app)
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *TerraApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *TerraApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *TerraApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *TerraApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlacklistedAccAddrs returns all the app's module account addresses black listed for receiving tokens.
func (app *TerraApp) BlacklistedAccAddrs() map[string]bool {
	blacklistedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blacklistedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blacklistedAddrs
}

// LegacyAmino returns TerraApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *TerraApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Gaia's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *TerraApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Gaia's InterfaceRegistry
func (app *TerraApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *TerraApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *TerraApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *TerraApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register custom tx routes from grpc-gateway.
	customauthtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}

	// Apply custom middleware
	apiSvr.Router.Use(customserver.BlockHeightMiddleware)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *TerraApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
	customauthtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.TreasuryKeeper, app.TaxKeeper)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *TerraApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

func (app *TerraApp) RegisterNodeService(clientCtx client.Context) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.NewWithNamespace("terrad")
	if err != nil {
		panic(err)
	}
	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

func (app *TerraApp) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			storeUpgrades := upgrade.StoreUpgrades
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

func (app *TerraApp) setupUpgradeHandlers() {
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.mm,
				app.configurator,
				app.BaseApp,
				app.AppKeepers,
			),
		)
	}
}

// GetTxConfig for testing
func (app *TerraApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}
