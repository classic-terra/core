package app

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	terraappparams "github.com/classic-terra/core/v3/app/params"
	customauth "github.com/classic-terra/core/v3/custom/auth"
	customauthsim "github.com/classic-terra/core/v3/custom/auth/simulation"
	customauthz "github.com/classic-terra/core/v3/custom/authz"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customcrisis "github.com/classic-terra/core/v3/custom/crisis"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customevidence "github.com/classic-terra/core/v3/custom/evidence"
	customfeegrant "github.com/classic-terra/core/v3/custom/feegrant"
	customgov "github.com/classic-terra/core/v3/custom/gov"
	custommint "github.com/classic-terra/core/v3/custom/mint"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customslashing "github.com/classic-terra/core/v3/custom/slashing"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	customupgrade "github.com/classic-terra/core/v3/custom/upgrade"
	customwasm "github.com/classic-terra/core/v3/custom/wasm"
	"github.com/classic-terra/core/v3/x/dyncomm"
	dyncommtypes "github.com/classic-terra/core/v3/x/dyncomm/types"
	"github.com/classic-terra/core/v3/x/market"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	"github.com/classic-terra/core/v3/x/oracle"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	taxmodule "github.com/classic-terra/core/v3/x/tax/module"
	"github.com/classic-terra/core/v3/x/taxexemption"
	taxexemptiontypes "github.com/classic-terra/core/v3/x/taxexemption/types"
	"github.com/classic-terra/core/v3/x/treasury"
	treasuryclient "github.com/classic-terra/core/v3/x/treasury/client"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	"github.com/classic-terra/core/v3/x/vesting"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensus "github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
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
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"

	taxbank "github.com/classic-terra/core/v3/x/tax/modules/bank"
	taxmarket "github.com/classic-terra/core/v3/x/tax/modules/market"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/classic-terra/core/v3/client/docs/statik"
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
		capability.AppModuleBasic{},
		customstaking.AppModuleBasic{},
		custommint.AppModuleBasic{},
		customdistr.AppModuleBasic{},
		customgov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				upgradeclient.LegacyProposalHandler,
				upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler,
				ibcclientclient.UpgradeProposalHandler,
				treasuryclient.ProposalAddBurnTaxExemptionAddressHandler,
				treasuryclient.ProposalRemoveBurnTaxExemptionAddressHandler,
			},
		),
		customparams.AppModuleBasic{},
		customcrisis.AppModuleBasic{},
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
		ibcfee.AppModuleBasic{},
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
		ibcfeetypes.ModuleName:         nil,
		icatypes.ModuleName:            nil,
		wasmtypes.ModuleName:           {authtypes.Burner},
		ibchookstypes.ModuleName:       nil,
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
	skipGenesisInvariants bool,
) []module.AppModule {
	appCodec := encodingConfig.Marshaler
	return []module.AppModule{
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		taxbank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.TaxExemptionKeeper, app.TreasuryKeeper, app.GetSubspace(banktypes.ModuleName), app.TaxKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		taxmarket.NewAppModule(appCodec, app.MarketKeeper, app.AccountKeeper, app.TreasuryKeeper, app.BankKeeper, app.OracleKeeper, app.TaxKeeper),
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		treasury.NewAppModule(appCodec, app.TreasuryKeeper),
		taxexemption.NewAppModule(appCodec, app.TaxExemptionKeeper),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		dyncomm.NewAppModule(appCodec, app.DyncommKeeper, app.StakingKeeper),
		ibchooks.NewAppModule(app.AccountKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		taxmodule.NewAppModule(appCodec, app.TaxKeeper),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	}
}

func simulationModules(
	app *TerraApp,
	encodingConfig terraappparams.EncodingConfig,
	_ bool,
) []module.AppModuleSimulation {
	appCodec := encodingConfig.Marshaler
	return []module.AppModuleSimulation{
		customauth.NewAppModule(appCodec, app.AccountKeeper, customauthsim.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		custombank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		market.NewAppModule(appCodec, app.MarketKeeper, app.AccountKeeper, app.BankKeeper, app.OracleKeeper),
		treasury.NewAppModule(appCodec, app.TreasuryKeeper),
		taxexemption.NewAppModule(appCodec, app.TaxExemptionKeeper),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		dyncomm.NewAppModule(appCodec, app.DyncommKeeper, app.StakingKeeper),
		taxmodule.NewAppModule(appCodec, app.TaxKeeper),
	}
}

func orderBeginBlockers() []string {
	return []string{
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
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
		ibcfeetypes.ModuleName,
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
		capabilitytypes.ModuleName,
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
		ibcfeetypes.ModuleName,
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
		capabilitytypes.ModuleName,
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
		ibcfeetypes.ModuleName,
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
