package keepers

import (
	"fmt"
	"path/filepath"

	sdklog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasm "github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	customstaking "github.com/classic-terra/core/v4/custom/staking"
	customwasmkeeper "github.com/classic-terra/core/v4/custom/wasm/keeper"
	terrawasm "github.com/classic-terra/core/v4/wasmbinding"
	dyncommkeeper "github.com/classic-terra/core/v4/x/dyncomm/keeper"
	dyncommtypes "github.com/classic-terra/core/v4/x/dyncomm/types"
	marketkeeper "github.com/classic-terra/core/v4/x/market/keeper"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oraclekeeper "github.com/classic-terra/core/v4/x/oracle/keeper"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	taxkeeper "github.com/classic-terra/core/v4/x/tax/keeper"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	taxexemptionkeeper "github.com/classic-terra/core/v4/x/taxexemption/keeper"
	taxexemptiontypes "github.com/classic-terra/core/v4/x/taxexemption/types"
	treasurykeeper "github.com/classic-terra/core/v4/x/treasury/keeper"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10"
	ibchookskeeper "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10/keeper"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10/types"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

type AppKeepers struct {
	// appKeepers.keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	AuthzKeeper           authzkeeper.Keeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the appKeepers, so we can SetRouter on it correctly
	ICAControllerKeeper   icacontrollerkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	OracleKeeper          oraclekeeper.Keeper
	MarketKeeper          marketkeeper.Keeper
	TreasuryKeeper        treasurykeeper.Keeper
	TaxExemptionKeeper    taxexemptionkeeper.Keeper
	WasmKeeper            wasmkeeper.Keeper
	DyncommKeeper         dyncommkeeper.Keeper
	IBCHooksKeeper        *ibchookskeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	TaxKeeper             taxkeeper.Keeper

	Ics20WasmHooks  *ibchooks.WasmHooks
	IBCHooksWrapper *ibchooks.ICS4Middleware
	TransferStack   ibctransfer.IBCModule
}

func NewAppKeepers(
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	allowedReceivingModAcc map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	wasmOpts []wasmkeeper.Option,
	appOpts servertypes.AppOptions,
) *AppKeepers {
	keys := map[string]*storetypes.KVStoreKey{
		authtypes.StoreKey:           storetypes.NewKVStoreKey(authtypes.StoreKey),
		banktypes.StoreKey:           storetypes.NewKVStoreKey(banktypes.StoreKey),
		stakingtypes.StoreKey:        storetypes.NewKVStoreKey(stakingtypes.StoreKey),
		minttypes.StoreKey:           storetypes.NewKVStoreKey(minttypes.StoreKey),
		distrtypes.StoreKey:          storetypes.NewKVStoreKey(distrtypes.StoreKey),
		slashingtypes.StoreKey:       storetypes.NewKVStoreKey(slashingtypes.StoreKey),
		govtypes.StoreKey:            storetypes.NewKVStoreKey(govtypes.StoreKey),
		paramstypes.StoreKey:         storetypes.NewKVStoreKey(paramstypes.StoreKey),
		consensusparamtypes.StoreKey: storetypes.NewKVStoreKey(consensusparamtypes.StoreKey),
		upgradetypes.StoreKey:        storetypes.NewKVStoreKey(upgradetypes.StoreKey),
		feegrant.StoreKey:            storetypes.NewKVStoreKey(feegrant.StoreKey),
		evidencetypes.StoreKey:       storetypes.NewKVStoreKey(evidencetypes.StoreKey),
		authzkeeper.StoreKey:         storetypes.NewKVStoreKey(authzkeeper.StoreKey),
		ibcexported.StoreKey:         storetypes.NewKVStoreKey(ibcexported.StoreKey),
		ibctransfertypes.StoreKey:    storetypes.NewKVStoreKey(ibctransfertypes.StoreKey),
		icacontrollertypes.StoreKey:  storetypes.NewKVStoreKey(icacontrollertypes.StoreKey),
		icahosttypes.StoreKey:        storetypes.NewKVStoreKey(icahosttypes.StoreKey),
		ibchookstypes.StoreKey:       storetypes.NewKVStoreKey(ibchookstypes.StoreKey),
		oracletypes.StoreKey:         storetypes.NewKVStoreKey(oracletypes.StoreKey),
		markettypes.StoreKey:         storetypes.NewKVStoreKey(markettypes.StoreKey),
		treasurytypes.StoreKey:       storetypes.NewKVStoreKey(treasurytypes.StoreKey),
		taxexemptiontypes.StoreKey:   storetypes.NewKVStoreKey(taxexemptiontypes.StoreKey),
		wasmtypes.StoreKey:           storetypes.NewKVStoreKey(wasmtypes.StoreKey),
		dyncommtypes.StoreKey:        storetypes.NewKVStoreKey(dyncommtypes.StoreKey),
		taxtypes.StoreKey:            storetypes.NewKVStoreKey(taxtypes.StoreKey),
	}
	tkeys := map[string]*storetypes.TransientStoreKey{
		paramstypes.TStoreKey: storetypes.NewTransientStoreKey(paramstypes.TStoreKey),
	}
	memKeys := map[string]*storetypes.MemoryStoreKey{}

	appKeepers := &AppKeepers{
		keys:    keys,
		tkeys:   tkeys,
		memKeys: memKeys,
	}

	// Address codecs (v0.50)
	accAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	valConsAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	// load state streaming if enabled
	if err := bApp.RegisterStreamingServices(appOpts, appKeepers.keys); err != nil {
		panic(fmt.Errorf("failed to load state streaming err %v", err))
	}

	// init params keeper and subspaces
	appKeepers.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		appKeepers.keys[paramstypes.StoreKey],
		appKeepers.tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	appKeepers.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	bApp.SetParamStore(appKeepers.ConsensusParamsKeeper.ParamsStore)

	// add keepers
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		accAddrCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BlacklistedAccAddrs(maccPerms, allowedReceivingModAcc),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		sdklog.NewNopLogger(),
	)
	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[authzkeeper.StoreKey]),
		appCodec,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
	)
	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[feegrant.StoreKey]),
		appKeepers.AccountKeeper,
	)
	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[stakingtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		valAddrCodec,
		valConsAddrCodec,
	)
	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[minttypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[distrtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(appKeepers.keys[slashingtypes.StoreKey]),
		appKeepers.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(appKeepers.keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		bApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(customstaking.NewTerraStakingHooks(*appKeepers.StakingKeeper), appKeepers.DistrKeeper.Hooks(), appKeepers.SlashingKeeper.Hooks()),
	)

	// Create IBC Keeper (v10 signature)
	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibcexported.StoreKey]),
		appKeepers.GetSubspace(ibcexported.ModuleName),
		appKeepers.UpgradeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Register IBC light clients (v10 requires explicit registration)
	{
		clientKeeper := appKeepers.IBCKeeper.ClientKeeper
		storeProvider := clientKeeper.GetStoreProvider()
		tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
		clientKeeper.AddRoute(ibctm.ModuleName, tmLightClientModule)
	}

	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[icahosttypes.StoreKey]),
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCHooksWrapper, // ICS4Wrapper from ibc-hooks
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.AccountKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[icacontrollertypes.StoreKey]),
		appKeepers.GetSubspace(icacontrollertypes.SubModuleName),
		appKeepers.IBCHooksWrapper, // ICS4Wrapper from ibc-hooks
		appKeepers.IBCKeeper.ChannelKeeper,
		bApp.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[evidencetypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		appKeepers.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the appKeepers, set routes in router here and seal
	appKeepers.EvidenceKeeper = *evidenceKeeper

	// Initialize terra module keepers
	appKeepers.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec, appKeepers.keys[oracletypes.StoreKey], appKeepers.GetSubspace(oracletypes.ModuleName),
		appKeepers.AccountKeeper, appKeepers.BankKeeper, appKeepers.DistrKeeper, appKeepers.StakingKeeper, distrtypes.ModuleName,
	)
	appKeepers.MarketKeeper = marketkeeper.NewKeeper(
		appCodec, appKeepers.keys[markettypes.StoreKey],
		appKeepers.GetSubspace(markettypes.ModuleName),
		appKeepers.AccountKeeper, appKeepers.BankKeeper, appKeepers.OracleKeeper,
	)
	appKeepers.TreasuryKeeper = treasurykeeper.NewKeeper(
		appCodec, appKeepers.keys[treasurytypes.StoreKey],
		appKeepers.GetSubspace(treasurytypes.ModuleName),
		appKeepers.AccountKeeper, appKeepers.BankKeeper,
		appKeepers.MarketKeeper, appKeepers.OracleKeeper,
		appKeepers.StakingKeeper, appKeepers.DistrKeeper,
		&appKeepers.WasmKeeper, distrtypes.ModuleName,
	)

	appKeepers.TaxExemptionKeeper = taxexemptionkeeper.NewKeeper(
		appCodec, appKeepers.keys[taxexemptiontypes.StoreKey],
		appKeepers.GetSubspace(taxexemptiontypes.ModuleName),
		appKeepers.AccountKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	hooksKeeper := ibchookskeeper.NewKeeper(
		appKeepers.keys[ibchookstypes.StoreKey],
	)
	appKeepers.IBCHooksKeeper = &hooksKeeper

	// - contract keeper needs to be initialized after wasm
	// - transfer needs to be initialized before wasm
	// - hooks needs to be initialized before transfer
	wasmHooks := ibchooks.NewWasmHooks(
		appKeepers.IBCHooksKeeper, nil,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
	)
	appKeepers.Ics20WasmHooks = &wasmHooks

	hooksMiddleware := ibchooks.NewICS4Middleware(
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.Ics20WasmHooks,
	)
	appKeepers.IBCHooksWrapper = &hooksMiddleware

	// Create Transfer Keepers AFTER Hooks keeper but BEFORE wasm
	appKeepers.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibctransfertypes.StoreKey]),
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCHooksWrapper,         // ICS4Wrapper (hooks)
		appKeepers.IBCKeeper.ChannelKeeper, // ChannelKeeper
		bApp.MsgServiceRouter(),            // MessageRouter
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	wasmDir := filepath.Join(homePath, "data")
	wasmNodeConfig, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		panic(err)
	}
	wasmVMConfig := wasmtypes.VMConfig{}

	wasmMsgHandler := customwasmkeeper.NewMessageHandler(
		bApp.MsgServiceRouter(),
		&appKeepers.WasmKeeper,
		appKeepers.IBCHooksWrapper,
		appKeepers.IBCKeeper.ChannelKeeperV2,
		appKeepers.BankKeeper,
		appCodec,
		appKeepers.TransferKeeper,
	)
	// the first slice will replace all default msh handler with custom one
	wasmOpts = append([]wasmkeeper.Option{wasmkeeper.WithMessageHandler(wasmMsgHandler)}, wasmOpts...)
	// the second slice will add custom querier and message handler decorator
	// this order must be uphold else error will be thrown
	wasmOpts = append(
		wasmOpts,
		terrawasm.RegisterCustomPlugins(
			&appKeepers.MarketKeeper,
			&appKeepers.OracleKeeper,
			&appKeepers.TreasuryKeeper,
		)...,
	)
	wasmOpts = append(
		wasmOpts,
		terrawasm.RegisterStargateQueries(
			*bApp.GRPCQueryRouter(),
			appCodec,
		)...,
	)
	// Register legacy query handler for contract-to-contract queries at historical heights
	wasmOpts = append(wasmOpts, terrawasm.RegisterLegacyQueryHandler(appKeepers.keys[wasmtypes.StoreKey]))

	appKeepers.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[wasmtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		distrkeeper.NewQuerier(appKeepers.DistrKeeper),           // DistributionKeeper
		appKeepers.IBCHooksWrapper,                               // ICS4Wrapper (hooks)
		appKeepers.IBCKeeper.ChannelKeeper,                       // ChannelKeeper
		appKeepers.IBCKeeper.ChannelKeeperV2,                     // ChannelKeeperV2
		appKeepers.TransferKeeper,                                // ICS20TransferPortSource
		bApp.MsgServiceRouter(),                                  // MessageRouter
		bApp.GRPCQueryRouter(),                                   // GRPCQueryRouter
		wasmDir,                                                  // homeDir
		wasmNodeConfig,                                           // NodeConfig
		wasmVMConfig,                                             // VMConfig
		append(wasmkeeper.BuiltInCapabilities(), "terra"),        // availableCapabilities
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), // authority
		wasmOpts..., // Options
	)

	// AFTER wasm set contractKeeper for ics20 wasm hook
	appKeepers.Ics20WasmHooks.ContractKeeper = &appKeepers.WasmKeeper

	// register the proposal types
	govRouter := appKeepers.newGovRouter()
	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[govtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
		bApp.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)
	appKeepers.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	appKeepers.TaxKeeper = taxkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[taxtypes.StoreKey],
		appKeepers.BankKeeper,
		appKeepers.TreasuryKeeper,
		appKeepers.DistrKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.DyncommKeeper = dyncommkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[dyncommtypes.StoreKey],
		appKeepers.GetSubspace(dyncommtypes.ModuleName),
		appKeepers.StakingKeeper,
	)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := appKeepers.newIBCRouter()
	appKeepers.IBCKeeper.SetRouter(ibcRouter)

	return appKeepers
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec,
	legacyAmino *codec.LegacyAmino,
	key,
	tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName).WithKeyTable(authtypes.ParamKeyTable())
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable())
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypesv1.ParamKeyTable())
	// IBC Transfer legacy params key table (SendEnabled, ReceiveEnabled)
	{
		transferSS := paramsKeeper.Subspace(ibctransfertypes.ModuleName)
		if !transferSS.HasKeyTable() {
			transferSS.WithKeyTable(ibctransfertypes.ParamKeyTable())
		}
	}
	// IBC core (legacy x/params) subspace: register both client and connection param key tables once
	// NOTE: calling WithKeyTable twice panics; build a combined key table instead and guard with HasKeyTable
	{
		ibcCoreSubspace := paramsKeeper.Subspace(ibcexported.ModuleName)
		if !ibcCoreSubspace.HasKeyTable() {
			ibcCoreKT := paramstypes.NewKeyTable()
			ibcCoreKT = ibcCoreKT.RegisterParamSet(&clienttypes.Params{})
			ibcCoreKT = ibcCoreKT.RegisterParamSet(&connectiontypes.Params{})
			ibcCoreSubspace.WithKeyTable(ibcCoreKT)
		}
	}
	// ICA Host legacy params key table
	{
		hostSS := paramsKeeper.Subspace(icahosttypes.SubModuleName)
		if !hostSS.HasKeyTable() {
			hostSS.WithKeyTable(icahosttypes.ParamKeyTable())
		}
	}
	// ICA Controller legacy params key table
	{
		ctrlSS := paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
		if !ctrlSS.HasKeyTable() {
			ctrlSS.WithKeyTable(icacontrollertypes.ParamKeyTable())
		}
	}
	paramsKeeper.Subspace(markettypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(taxexemptiontypes.ModuleName)
	paramsKeeper.Subspace(treasurytypes.ModuleName)
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	paramsKeeper.Subspace(dyncommtypes.ModuleName)
	paramsKeeper.Subspace(taxtypes.ModuleName)

	return paramsKeeper
}

// GetSubspace returns a param subspace for a given module name.
func (appKeepers *AppKeepers) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := appKeepers.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// BlacklistedAccAddrs returns all the app's module account addresses black listed for receiving tokens.
func (appKeepers *AppKeepers) BlacklistedAccAddrs(
	maccPerms map[string][]string,
	allowedReceivingModAcc map[string]bool,
) map[string]bool {
	blacklistedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blacklistedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blacklistedAddrs
}
