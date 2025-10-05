package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	log "cosmossdk.io/log"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"cosmossdk.io/client/v2/autocli"
	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	snapshots "cosmossdk.io/store/snapshots"
	snapshottypes "cosmossdk.io/store/snapshots/types"
	storetypes "cosmossdk.io/store/types"
	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	snapshot "github.com/cosmos/cosmos-sdk/client/snapshot"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutil "github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	terraapp "github.com/classic-terra/core/v3/app"
	terralegacy "github.com/classic-terra/core/v3/app/legacy"
	"github.com/classic-terra/core/v3/app/params"
	authcustomcli "github.com/classic-terra/core/v3/custom/auth/client/cli"
	core "github.com/classic-terra/core/v3/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// NewRootCmd creates a new root command for terrad. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	// Set SDK config FIRST before creating any apps
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(core.CoinType)
	sdkConfig.SetPurpose(core.Purpose)
	sdkConfig.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	sdkConfig.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	sdkConfig.SetAddressVerifier(wasmtypes.VerifyAddressLen())
	sdkConfig.Seal()

	// Create temporary directory for CLI setup
	tempDir, err := os.MkdirTemp("", "terrad")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	encodingConfig := terraapp.MakeEncodingConfig()

	// Create a temporary app for CLI command setup
	// this is needed to initialize the app for the CLI command setup
	// the same method is used in the official wasmd sample app
	tempApp := terraapp.NewTerraApp(
		sdklog.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		tempDir,
		encodingConfig,
		simtestutil.EmptyAppOptions{},
		[]wasm.Option{}, // empty wasm options
	)

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(terraapp.DefaultNodeHome).
		WithViper("TERRA")

	rootCmd := &cobra.Command{
		Use:   "terrad",
		Short: "Stargate Terra App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			// attach command context (SDK 0.50 pattern)
			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// Enable SIGN_MODE_TEXTUAL when online (SDK 0.50 pattern)
			if !initClientCtx.Offline {
				enabledSignModes := append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL)
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           enabledSignModes,
					TextualCoinMetadataQueryFn: authtxconfig.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txCfg, err := tx.NewTxConfigWithOptions(initClientCtx.Codec, txConfigOpts)
				if err != nil {
					return err
				}
				initClientCtx = initClientCtx.WithTxConfig(txCfg)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			terraAppTemplate, terraAppConfig := initAppConfig()
			customTMConfig := initTendermintConfig()

			return server.InterceptConfigsPreRunHandler(cmd, terraAppTemplate, terraAppConfig, customTMConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig, tempApp.BasicModuleManager())

	// Enhance CLI with AutoCLI for modules that don't expose manual GetTxCmd/GetQueryCmd.
	// This adds missing upstream module commands (e.g., staking, distribution, gov) under query/tx.
	{
		sc := encodingConfig.InterfaceRegistry.SigningContext()
		modOpts := services.ExtractAutoCLIOptions(tempApp.Modules())
		// Only enhance Query via AutoCLI to avoid conflicting/duplicate TX flags and commands
		for _, opt := range modOpts {
			if opt != nil {
				opt.Tx = nil
			}
		}
		autoOpts := autocli.AppOptions{
			ModuleOptions:         modOpts,
			AddressCodec:          sc.AddressCodec(),
			ValidatorAddressCodec: sc.ValidatorAddressCodec(),
			ConsensusAddressCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
			ClientCtx:             initClientCtx,
		}
		if err := autoOpts.EnhanceRootCommand(rootCmd); err != nil {
			panic(err)
		}
	}

	return rootCmd, encodingConfig
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig params.EncodingConfig, basicMgr module.BasicManager) {
	a := appCreator{encodingConfig}

	gentxModule := terraapp.ModuleBasics[genutiltypes.ModuleName].(genutil.AppModuleBasic)

	// Use the app's TxConfig for genutil CLI
	txEnc := encodingConfig.TxConfig

	// Wrap app creator/exporter into the explicit types expected by helpers
	appCreatorFn := servertypes.AppCreator(func(_ sdklog.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
		// adapt SDK logger to Comet logger by using a Nop logger
		return a.newApp(log.NewNopLogger(), db, traceStore, appOpts)
	})
	appExporterFn := servertypes.AppExporter(func(_ sdklog.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailAllowedAddrs []string, appOpts servertypes.AppOptions, modulesToExport []string) (servertypes.ExportedApp, error) {
		return a.appExport(log.NewNopLogger(), db, traceStore, height, forZeroHeight, jailAllowedAddrs, appOpts, modulesToExport)
	})

	rootCmd.AddCommand(
		genutilcli.InitCmd(basicMgr, terraapp.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, terraapp.DefaultNodeHome, gentxModule.GenTxValidator, addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())),
		terralegacy.MigrateGenesisCmd(),
		genutilcli.GenTxCmd(basicMgr, txEnc, banktypes.GenesisBalancesIterator{}, terraapp.DefaultNodeHome, addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())),
		genutilcli.ValidateGenesisCmd(basicMgr),
		AddGenesisAccountCmd(terraapp.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		testnetCmd(basicMgr, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		pruning.Cmd(appCreatorFn, terraapp.DefaultNodeHome),
		snapshot.Cmd(appCreatorFn),
	)

	server.AddCommands(rootCmd, terraapp.DefaultNodeHome, appCreatorFn, appExporterFn, addModuleInitFlags)

	// add keybase, auxiliary status, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		queryCommand(basicMgr),
		txCommand(basicMgr),
		keys.Commands(),
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
	wasm.AddModuleInitFlags(startCmd)
}

func queryCommand(basicMgr module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		server.ShowAddressCmd(),
		server.ShowValidatorCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
		authcustomcli.GetTxFeesEstimateCommand(),
	)

	basicMgr.AddQueryCommands(cmd)
	// expose common query flags (node, height, etc.) so that AutoCLI commands
	// like staking queries receive --height and perform historic queries
	flags.AddQueryFlagsToCmd(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand(basicMgr module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		flags.LineBreak,
	)

	// Add module transaction commands from module basics
	basicMgr.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// emptyAppOptions is a minimal AppOptions used for constructing a temporary app for CLI wiring
type emptyAppOptions struct{}

func (emptyAppOptions) Get(_ string) interface{} { return nil }

type appCreator struct {
	encodingConfig params.EncodingConfig
}

// newApp is an AppCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	var cache storetypes.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	if homeDir == "" {
		homeDir = terraapp.DefaultNodeHome
	}
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if chainID == "" {
		// Try to read chain-id from genesis.json if it exists; otherwise fall back to a safe default
		genDocFile := filepath.Join(homeDir, "config", "genesis.json")
		if fi, statErr := os.Stat(genDocFile); statErr == nil && !fi.IsDir() {
			appGenesis, gErr := genutiltypes.AppGenesisFromFile(genDocFile)
			if gErr == nil {
				chainID = appGenesis.ChainID
			}
		}
		// If still empty (e.g., when running CLI help without an initialized home), use a benign default
		if chainID == "" {
			chainID = "terra-local"
		}
	}

	snapshotDir := filepath.Join(homeDir, "data", "snapshots")
	err = os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	snapshotDB, err := dbm.NewDB("metadata", server.GetAppDBBackend(appOpts), snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent)),
	)

	app := terraapp.NewTerraApp(
		logger, db, traceStore, true, skipUpgradeHeights,
		homeDir,
		a.encodingConfig,
		appOpts,
		nil,
		baseapp.SetChainID(chainID),
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(server.FlagIAVLCacheSize))),
		baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get(server.FlagDisableIAVLFastNode))),
		//baseapp.SetIAVLLazyLoading(cast.ToBool(appOpts.Get(server.FlagIAVLLazyLoading))),
	)

	return app
}

func (a appCreator) appExport(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailAllowedAddrs []string,
	appOpts servertypes.AppOptions, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	var terraApp *terraapp.TerraApp
	if height != -1 {
		terraApp = terraapp.NewTerraApp(logger, db, traceStore, false, map[int64]bool{}, homePath, a.encodingConfig, appOpts, nil)

		if err := terraApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		terraApp = terraapp.NewTerraApp(logger, db, traceStore, true, map[int64]bool{}, homePath, a.encodingConfig, appOpts, nil)
	}

	return terraApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}
