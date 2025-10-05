package helpers

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	sdklog "cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v3/app"
	appparams "github.com/classic-terra/core/v3/app/params"
	core "github.com/classic-terra/core/v3/types"
	dyncommtypes "github.com/classic-terra/core/v3/x/dyncomm/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	taxtypes "github.com/classic-terra/core/v3/x/tax/types"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SimAppChainID hardcoded chainID for simulation
const (
	SimAppChainID = ""
)

var emptyWasmOpts []wasmkeeper.Option

// EmptyBaseAppOptions is a stub implementing AppOptions
type EmptyBaseAppOptions struct{}

type KeeperTestHelper struct {
	suite.Suite

	App         *app.TerraApp
	Ctx         sdk.Context // ctx is deliver ctx
	CheckCtx    sdk.Context
	QueryHelper *baseapp.QueryServiceTestHelper
	TestAccs    []sdk.AccAddress
}

func (s *KeeperTestHelper) Setup(_ *testing.T, chainID string) {

	s.App = SetupApp(s.T(), chainID)
	// Create context after genesis has been initialized and committed
	// Height 1 because InitChain and Commit have been called in SetupApp
	header := tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()}
	s.Ctx = s.App.BaseApp.NewUncachedContext(false, header)
	s.CheckCtx = s.App.BaseApp.NewUncachedContext(true, header)

	s.App.ConsensusParamsKeeper.ParamsStore.Set(s.Ctx, *simtestutil.DefaultConsensusParams)

	s.App.MintKeeper.Params.Set(s.Ctx, minttypes.DefaultParams())

	// Set gas price to 0 in tax params
	taxParams := taxtypes.DefaultParams()
	taxParams.GasPrices = sdk.NewDecCoins(sdk.NewDecCoin(core.MicroSDRDenom, sdkmath.ZeroInt()))
	if err := s.App.TaxKeeper.SetParams(s.Ctx, taxParams); err != nil {
		panic(err)
	}
	s.App.MintKeeper.Minter.Set(s.Ctx, minttypes.DefaultInitialMinter())
	// Distribution params must be explicitly set in tests, or else queries fail
	// due to collections-based param store requiring explicit initialization
	// (unlike x/mint, which has a default value in keeper.go)

	s.App.AccountKeeper.Params.Set(s.Ctx, authtypes.DefaultParams())

	s.App.DistrKeeper.Params.Set(s.Ctx, distrtypes.DefaultParams())
	s.App.DistrKeeper.FeePool.Set(s.Ctx, distrtypes.InitialFeePool())

	// Explicitly set WASM params to ensure they're available in collections
	// This is needed because the collections-based param store requires explicit initialization
	s.App.WasmKeeper.SetParams(s.Ctx, wasmtypes.DefaultParams())

	// Explicitly set bank params to ensure sends are enabled
	// This ensures uluna transfers work properly in tests
	bankParams := banktypes.DefaultParams()
	bankParams.DefaultSendEnabled = true
	s.App.BankKeeper.SetParams(s.Ctx, bankParams)
	s.App.BankKeeper.SetSendEnabled(s.Ctx, "uluna", true)

	// Explicitly set Terra module params to ensure they're available in paramSpace
	// This is needed because modules using paramSpace.Get() require explicit initialization
	stakingparams := stakingtypes.DefaultParams()
	stakingparams.BondDenom = appparams.BondDenom
	s.App.StakingKeeper.SetParams(s.Ctx, stakingparams)
	s.App.MarketKeeper.SetParams(s.Ctx, markettypes.DefaultParams())
	s.App.OracleKeeper.SetParams(s.Ctx, oracletypes.DefaultParams())
	s.App.TreasuryKeeper.SetParams(s.Ctx, treasurytypes.DefaultParams())
	s.App.DyncommKeeper.SetParams(s.Ctx, dyncommtypes.DefaultParams())

	s.QueryHelper = baseapp.NewQueryServerTestHelper(s.Ctx, s.App.InterfaceRegistry())

	s.TestAccs = s.RandomAccountAddresses(3)
}

// Get implements AppOptions
func (ao EmptyBaseAppOptions) Get(_ string) interface{} {
	return nil
}

type EmptyAppOptions struct{}

func (EmptyAppOptions) Get(_ string) interface{} { return nil }

// DefaultConsensusParams defines the default Tendermint consensus params used
// in app testing.
var DefaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func SetupApp(t *testing.T, chainID string) *app.TerraApp {
	t.Helper()

	// Ensure Terra bech32 prefixes are set before keepers initialize address codecs
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	sdk.GetConfig().SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	sdk.GetConfig().SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)

	privVal := NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)
	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdkmath.NewInt(100000000000000))),
	}
	genesisAccounts := []authtypes.GenesisAccount{acc}
	app := SetupWithGenesisValSet(t, chainID, valSet, genesisAccounts, balance)

	return app
}

// SetupWithGenesisValSet initializes a new app with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit in the default token of the app from first genesis
// account. A Nop logger is set in app.
func SetupWithGenesisValSet(
	t *testing.T, chainID string, valSet *tmtypes.ValidatorSet,
	genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance,
) *app.TerraApp {
	t.Helper()

	terraApp, genesisState := setup(chainID)
	genesisState = genesisStateWithValSet(t, terraApp, genesisState, valSet, genAccs, balances...)

	stateBytes, err := json.MarshalIndent(genesisState, "", "")
	require.NoError(t, err)

	// InitChain writes all module genesis
	terraApp.InitChain(&abci.RequestInitChain{
		ChainId:         chainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	// Commit genesis
	_, terr := terraApp.Commit()
	require.NoError(t, terr)

	// Do not produce a block here; upstream integration tests keep app at post-genesis state
	// and let tests drive block progression if needed.
	return terraApp
}

func setup(chainID string) (*app.TerraApp, app.GenesisState) {
	db := dbm.NewMemDB()
	encCdc := app.MakeEncodingConfig()
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[server.FlagInvCheckPeriod] = 5
	appOptions[server.FlagMinGasPrices] = "0" + appparams.BondDenom

	// unique temp dir for each test
	baseDir, err := os.MkdirTemp("", "terrapp")
	if err != nil {
		panic(err)
	}

	terraapp := app.NewTerraApp(
		sdklog.NewNopLogger(), // for debugging we can use sdklog.NewLogger(os.Stdout, sdklog.LevelOption(zerolog.DebugLevel)),
		db,
		nil,
		true,
		map[int64]bool{},
		baseDir,
		encCdc,
		simtestutil.EmptyAppOptions{},
		emptyWasmOpts,
		baseapp.SetChainID(chainID),
	)

	return terraapp, app.GenesisState{}
}

func genesisStateWithValSet(t *testing.T,
	app *app.TerraApp, genesisState app.GenesisState,
	valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) app.GenesisState {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.DefaultPowerReduction

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		require.NoError(t, err)
		pkAny, err := codectypes.NewAnyWithValue(pk)
		require.NoError(t, err)
		validator := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdkmath.LegacyOneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress().String(), sdk.ValAddress(val.Address).String(), sdkmath.LegacyOneDec()))
	}
	// set validators and delegations
	defaultStParams := stakingtypes.DefaultParams()
	stParams := stakingtypes.NewParams(
		defaultStParams.UnbondingTime,
		defaultStParams.MaxValidators,
		defaultStParams.MaxEntries,
		defaultStParams.HistoricalEntries,
		appparams.BondDenom,
		defaultStParams.MinCommissionRate,
	)

	// set validators and delegations
	stakingGenesis := stakingtypes.NewGenesisState(stParams, validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	for range delegations {
		// add delegated tokens to total supply
		totalSupply = totalSupply.Add(sdk.NewCoin(appparams.BondDenom, bondAmt))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(appparams.BondDenom, bondAmt)},
	})

	// update total supply
	// enable send by default to avoid manual post-genesis mutations in tests
	bankParams := banktypes.DefaultParams()
	bankParams.DefaultSendEnabled = true
	bankGenesis := banktypes.NewGenesisState(
		bankParams,
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{
			{Denom: appparams.BondDenom, Enabled: true}, // Enable uluna transfers
		},
	)

	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	// update mint genesis state: ensure correct denom and minter initialized
	mintGenesis := minttypes.DefaultGenesisState()
	mintGenesis.Params.MintDenom = appparams.BondDenom
	mintGenesis.Minter = minttypes.DefaultInitialMinter()
	genesisState[minttypes.ModuleName] = app.AppCodec().MustMarshalJSON(mintGenesis)

	// distribution default params/state
	distGenesis := distrtypes.DefaultGenesisState()
	genesisState[distrtypes.ModuleName] = app.AppCodec().MustMarshalJSON(distGenesis)

	// update oracle genesis state
	oracleGenesis := oracletypes.DefaultGenesisState()
	genesisState[oracletypes.ModuleName] = app.AppCodec().MustMarshalJSON(oracleGenesis)

	// update market gensis state
	marketGenesis := markettypes.DefaultGenesisState()
	genesisState[markettypes.ModuleName] = app.AppCodec().MustMarshalJSON(marketGenesis)

	// update dyncomm genesis state
	dyncommGenesis := dyncommtypes.DefaultGenesisState()
	genesisState[dyncommtypes.ModuleName] = app.AppCodec().MustMarshalJSON(dyncommGenesis)

	// update treasury genesis state
	treasuryGensis := treasurytypes.DefaultGenesisState()
	genesisState[treasurytypes.ModuleName] = app.AppCodec().MustMarshalJSON(treasuryGensis)

	// tax genesis state; keep gas price zero to match many legacy tests
	taxGenesis := taxtypes.DefaultGenesisState()
	taxGenesis.Params.GasPrices = sdk.NewDecCoins(sdk.NewDecCoin(core.MicroSDRDenom, sdkmath.ZeroInt()))
	genesisState[taxtypes.ModuleName] = app.AppCodec().MustMarshalJSON(taxGenesis)

	// ensure wasm genesis state present with default params
	wasmGenesis := &wasmtypes.GenesisState{Params: wasmtypes.DefaultParams()}
	genesisState[wasmtypes.ModuleName] = app.AppCodec().MustMarshalJSON(wasmGenesis)

	return genesisState
}

func (s *KeeperTestHelper) Ed25519PubAddr() (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func (s *KeeperTestHelper) RandomAccountAddresses(n int) []sdk.AccAddress {
	addrsList := make([]sdk.AccAddress, n)
	for i := 0; i < n; i++ {
		_, _, addrs := testdata.KeyTestPubAddr()
		addrsList[i] = addrs
	}
	return addrsList
}

// FundAcc funds target address with specified amount.
func (s *KeeperTestHelper) FundAcc(acc sdk.AccAddress, amounts sdk.Coins) {
	err := banktestutil.FundAccount(s.Ctx, s.App.BankKeeper, acc, amounts)
	s.Require().NoError(err)
}

// NextBlock finalizes and commits the next block, updating contexts accordingly.
func (s *KeeperTestHelper) NextBlock() {
	height := s.App.LastBlockHeight() + 1
	// Use current header to preserve chain-id and hashes
	hdr := tmproto.Header{Height: height, ChainID: s.Ctx.ChainID(), Time: time.Now().UTC()}
	_, err := s.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: height,
		Time:   hdr.Time,
	})
	s.Require().NoError(err)
	_, err = s.App.Commit()
	s.Require().NoError(err)
	s.Ctx = s.App.BaseApp.NewUncachedContext(false, hdr)
	s.CheckCtx = s.App.BaseApp.NewUncachedContext(true, hdr)
}
