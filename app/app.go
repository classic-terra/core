package app

import (
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	sdklog "cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v4/app/keepers"
	appmempool "github.com/classic-terra/core/v4/app/mempool"
	terraappparams "github.com/classic-terra/core/v4/app/params"

	// upgrades
	"github.com/classic-terra/core/v4/app/upgrades"
	// v9 had been used by tax2gas and has to be skipped
	v10_1 "github.com/classic-terra/core/v4/app/upgrades/v10_1"
	v11 "github.com/classic-terra/core/v4/app/upgrades/v11"
	v11_1 "github.com/classic-terra/core/v4/app/upgrades/v11_1"
	v11_2 "github.com/classic-terra/core/v4/app/upgrades/v11_2"
	v12 "github.com/classic-terra/core/v4/app/upgrades/v12"
	v13 "github.com/classic-terra/core/v4/app/upgrades/v13"
	v13_1 "github.com/classic-terra/core/v4/app/upgrades/v13_1"
	v14_1 "github.com/classic-terra/core/v4/app/upgrades/v14_1"
	v14_2 "github.com/classic-terra/core/v4/app/upgrades/v14_2"
	v2 "github.com/classic-terra/core/v4/app/upgrades/v2"
	v3 "github.com/classic-terra/core/v4/app/upgrades/v3"
	v4 "github.com/classic-terra/core/v4/app/upgrades/v4"
	v5 "github.com/classic-terra/core/v4/app/upgrades/v5"
	v6 "github.com/classic-terra/core/v4/app/upgrades/v6"
	v6_1 "github.com/classic-terra/core/v4/app/upgrades/v6_1"
	v7 "github.com/classic-terra/core/v4/app/upgrades/v7"
	v7_1 "github.com/classic-terra/core/v4/app/upgrades/v7_1"
	v8 "github.com/classic-terra/core/v4/app/upgrades/v8"
	v8_1 "github.com/classic-terra/core/v4/app/upgrades/v8_1"
	v8_2 "github.com/classic-terra/core/v4/app/upgrades/v8_2"
	v8_3 "github.com/classic-terra/core/v4/app/upgrades/v8_3"

	// unnamed import of statik for swagger UI support
	_ "github.com/classic-terra/core/v4/client/docs/statik"
	customante "github.com/classic-terra/core/v4/custom/auth/ante"
	custompost "github.com/classic-terra/core/v4/custom/auth/post"
	customauthtx "github.com/classic-terra/core/v4/custom/auth/tx"
	customserver "github.com/classic-terra/core/v4/server"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	cmtservice "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"
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
		v13.Upgrade,
		v13_1.Upgrade,
		v14_1.Upgrade,
		v14_2.Upgrade,
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
	logger sdklog.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, skipUpgradeHeights map[int64]bool,
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

	// adapt CometBFT logger to cosmossdk.io/log.Logger expected by BaseApp
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

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(appModules(app, encodingConfig)...)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// PreBlockers run before BeginBlockers. In v0.50, x/upgrade must run in PreBlock.
	app.mm.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		authtypes.ModuleName,
	)
	app.mm.SetOrderBeginBlockers(orderBeginBlockers()...)
	app.mm.SetOrderEndBlockers(orderEndBlockers()...)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	// NOTE: Treasury must occur after bank module so that initial supply is properly set
	app.mm.SetOrderInitGenesis(orderInitGenesis()...)
	app.mm.SetOrderExportGenesis(orderInitGenesis()...)

	// NOTE: PreBlocker is supported in SDK v0.50; if needed, enable via BaseApp.SetPreBlocker.

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err := app.mm.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	app.setupUpgradeHandlers()
	app.setupUpgradeStoreLoaders()

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	app.sm = module.NewSimulationManager(simulationModules(app, encodingConfig)...)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	storeKeys := app.GetKVStoreKey()
	app.MountKVStores(storeKeys)
	app.MountTransientStores(app.GetTransientStoreKey())
	app.MountMemoryStores(app.GetMemoryStoreKey())

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	// In v0.50, modules like x/upgrade must run in PreBlock to update consensus params
	// Ensure PreBlocker is registered so module PreBlock ordering executes.
	app.SetPreBlocker(app.PreBlocker)

	wasmConfig, err := wasm.ReadNodeConfig(appOpts)
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
			TXCounterStore:     runtime.NewKVStoreService(app.GetKey(wasmtypes.StoreKey)),
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

		{
			/* TODO: check if there is a better way to make sure the client params are set
			this is a workaround for the fact that the client params are not set in the
			genesis and the upgrade handler is not enough */
			// Create a writeable context outside block processing
			ctx := app.NewUncachedContext(true, tmproto.Header{})

			// Raw-store check avoids calling GetParams() (which panics if missing)
			store := ctx.KVStore(app.GetKey(ibcexported.StoreKey))
			if !store.Has([]byte(clienttypes.ParamsKey)) {
				app.IBCKeeper.ClientKeeper.SetParams(ctx, clienttypes.DefaultParams())
				// no explicit commit needed; BaseApp will persist on next commit
			}
		}

		ctx := app.NewUncachedContext(true, tmproto.Header{})
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
	return app.BasicModuleManager().DefaultGenesis(app.appCodec)
}

func (app *TerraApp) Modules() map[string]interface{} {
	return app.mm.Modules
}

// BeginBlocker application updates every begin block
func (app *TerraApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	BeginBlockForks(ctx, app)
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *TerraApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

// PreBlocker runs before BeginBlocker in v0.50 and allows modules like x/upgrade
// to make consensus parameter changes visible to the rest of the block.
func (app *TerraApp) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}

// InitChainer application update at chain initialization
func (app *TerraApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	ctx.Logger().Debug("init genesis", "genesisState", genesisState)
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

// BasicModuleManager returns a BasicManager derived from the app's module manager.
// This is useful for CLI wiring where module Basic instances must be fully initialized
// (e.g., with codecs) to construct tx/query commands safely.
func (app *TerraApp) BasicModuleManager() module.BasicManager {
	// Use the SDK helper which extracts module basics (with initialized codecs)
	// from the module manager, ensuring CLI commands from upstream modules are
	// wired correctly.
	return module.NewBasicManagerFromManager(app.mm, nil)
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *TerraApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register custom tx routes from grpc-gateway.
	customauthtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new CometBFT queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register grpc-gateway routes for all modules via BasicModuleManager (SDK v0.50 style)
	app.BasicModuleManager().RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}

	// Apply custom middleware
	// TxLogsMiddleware reconstructs the deprecated logs field from events for backwards compatibility
	apiSvr.Router.Use(customserver.TxLogsMiddleware)
	apiSvr.Router.Use(customserver.BlockHeightMiddleware)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *TerraApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
	customauthtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.TreasuryKeeper, app.TaxExemptionKeeper, app.TaxKeeper)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *TerraApp) RegisterTendermintService(clientCtx client.Context) {
	cmtApp := server.NewCometABCIWrapper(app)
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.GRPCQueryRouter(),
		app.interfaceRegistry,
		cmtApp.Query,
	)
}

func (app *TerraApp) RegisterNodeService(clientCtx client.Context, config config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), config)
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
