package app

import (
	"cosmossdk.io/x/evidence"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	terraappparams "github.com/classic-terra/core/v4/app/params"
	// unnamed import of statik for swagger UI support
	_ "github.com/classic-terra/core/v4/client/docs/statik"
	customauth "github.com/classic-terra/core/v4/custom/auth"
	customauthsim "github.com/classic-terra/core/v4/custom/auth/simulation"
	customauthz "github.com/classic-terra/core/v4/custom/authz"
	custombank "github.com/classic-terra/core/v4/custom/bank"
	customdistr "github.com/classic-terra/core/v4/custom/distribution"
	customevidence "github.com/classic-terra/core/v4/custom/evidence"
	customfeegrant "github.com/classic-terra/core/v4/custom/feegrant"
	customgov "github.com/classic-terra/core/v4/custom/gov"
	custommint "github.com/classic-terra/core/v4/custom/mint"
	customparams "github.com/classic-terra/core/v4/custom/params"
	customslashing "github.com/classic-terra/core/v4/custom/slashing"
	customstaking "github.com/classic-terra/core/v4/custom/staking"
	customupgrade "github.com/classic-terra/core/v4/custom/upgrade"
	customwasm "github.com/classic-terra/core/v4/custom/wasm"
	"github.com/classic-terra/core/v4/x/dyncomm"
	dyncommtypes "github.com/classic-terra/core/v4/x/dyncomm/types"
	"github.com/classic-terra/core/v4/x/market"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	"github.com/classic-terra/core/v4/x/oracle"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	taxmodule "github.com/classic-terra/core/v4/x/tax/module"
	taxbank "github.com/classic-terra/core/v4/x/tax/modules/bank"
	taxmarket "github.com/classic-terra/core/v4/x/tax/modules/market"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	"github.com/classic-terra/core/v4/x/taxexemption"
	taxexemptiontypes "github.com/classic-terra/core/v4/x/taxexemption/types"
	"github.com/classic-terra/core/v4/x/treasury"
	treasuryclient "github.com/classic-terra/core/v4/x/treasury/client"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	"github.com/classic-terra/core/v4/x/vesting"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10/types"
	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

var (
	// ModuleBasics = The ModuleBasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		customauth.AppModuleBasic{},
		customauthz.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		custombank.AppModuleBasic{},
		customstaking.AppModuleBasic{},
		custommint.AppModuleBasic{},
		customdistr.AppModuleBasic{},
		customgov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				treasuryclient.ProposalAddBurnTaxExemptionAddressHandler,
				treasuryclient.ProposalRemoveBurnTaxExemptionAddressHandler,
			},
		),
		customparams.AppModuleBasic{},
		customslashing.AppModuleBasic{},
		customfeegrant.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ica.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		customupgrade.AppModuleBasic{},
		customevidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		oracle.AppModuleBasic{},
		market.AppModuleBasic{},
		treasury.AppModuleBasic{},
		taxexemption.AppModuleBasic{},
		customwasm.AppModuleBasic{},
		dyncomm.AppModuleBasic{},
		ibchooks.AppModuleBasic{},
		consensus.AppModuleBasic{},
		taxmodule.AppModuleBasic{},
	)
	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil, // just added to enable align fee
		treasurytypes.BurnModuleName:   {authtypes.Burner},
		minttypes.ModuleName:           {authtypes.Minter},
		markettypes.ModuleName:         {authtypes.Minter, authtypes.Burner},
		oracletypes.ModuleName:         nil,
		distrtypes.ModuleName:          nil,
		treasurytypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:            nil,
		wasmtypes.ModuleName:           {authtypes.Burner},
	}
	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		oracletypes.ModuleName:       true,
		treasurytypes.BurnModuleName: true,
	}
)

func appModules(
	app *TerraApp,
	encodingConfig terraappparams.EncodingConfig,
) []module.AppModule {
	appCodec := encodingConfig.Marshaler
	return []module.AppModule{
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		taxbank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.TaxExemptionKeeper, app.TreasuryKeeper, app.GetSubspace(banktypes.ModuleName), app.TaxKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.InterfaceRegistry()),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		customstaking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.ParamsKeeper, app.GetSubspace(stakingtypes.ModuleName), app.GetKey(stakingtypes.StoreKey), app.GetKey(distrtypes.StoreKey)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		taxmarket.NewAppModule(appCodec, app.MarketKeeper, app.AccountKeeper, app.TreasuryKeeper, app.BankKeeper, app.OracleKeeper, app.TaxKeeper),
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		treasury.NewAppModule(appCodec, app.TreasuryKeeper),
		taxexemption.NewAppModule(appCodec, app.TaxExemptionKeeper),
		customwasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName), app.GetKey(wasmtypes.StoreKey)),
		dyncomm.NewAppModule(appCodec, app.DyncommKeeper, app.StakingKeeper),
		ibchooks.NewAppModule(app.AccountKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		taxmodule.NewAppModule(appCodec, app.TaxKeeper),
	}
}

func simulationModules(
	app *TerraApp,
	encodingConfig terraappparams.EncodingConfig,
) []module.AppModuleSimulation {
	appCodec := encodingConfig.Marshaler
	return []module.AppModuleSimulation{
		customauth.NewAppModule(appCodec, app.AccountKeeper, customauthsim.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		custombank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.InterfaceRegistry()),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		market.NewAppModule(appCodec, app.MarketKeeper, app.AccountKeeper, app.BankKeeper, app.OracleKeeper),
		treasury.NewAppModule(appCodec, app.TreasuryKeeper),
		taxexemption.NewAppModule(appCodec, app.TaxExemptionKeeper),
		customwasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName), app.GetKey(wasmtypes.StoreKey)),
		dyncomm.NewAppModule(appCodec, app.DyncommKeeper, app.StakingKeeper),
		taxmodule.NewAppModule(appCodec, app.TaxKeeper),
	}
}

func orderBeginBlockers() []string {
	return []string{
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		// additional non simd modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		ibchookstypes.ModuleName,
		// Terra Classic modules
		oracletypes.ModuleName,
		treasurytypes.ModuleName,
		taxexemptiontypes.ModuleName,
		markettypes.ModuleName,
		wasmtypes.ModuleName,
		dyncommtypes.ModuleName,
		taxtypes.ModuleName,
		// consensus module
		consensusparamtypes.ModuleName,
	}
}

func orderEndBlockers() []string {
	return []string{
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		// additional non simd modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		ibchookstypes.ModuleName,
		// Terra Classic modules
		oracletypes.ModuleName,
		treasurytypes.ModuleName,
		taxexemptiontypes.ModuleName,
		markettypes.ModuleName,
		wasmtypes.ModuleName,
		dyncommtypes.ModuleName,
		taxtypes.ModuleName,
		// consensus module
		consensusparamtypes.ModuleName,
	}
}

func orderInitGenesis() []string {
	return []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		feegrant.ModuleName,
		// additional non simd modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		ibchookstypes.ModuleName,
		// Terra Classic modules
		markettypes.ModuleName,
		oracletypes.ModuleName,
		treasurytypes.ModuleName,
		taxexemptiontypes.ModuleName,
		wasmtypes.ModuleName,
		dyncommtypes.ModuleName,
		taxtypes.ModuleName,
		// consensus module
		consensusparamtypes.ModuleName,
	}
}
