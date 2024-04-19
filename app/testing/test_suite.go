package helpers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v3/app"
	appparams "github.com/classic-terra/core/v3/app/params"
	dyncommtypes "github.com/classic-terra/core/v3/x/dyncomm/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
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
	SimAppChainID = "terra-app"
)

var emptyWasmOpts []wasm.Option

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
	s.Ctx = s.App.BaseApp.NewContext(false, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})
	s.CheckCtx = s.App.BaseApp.NewContext(true, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})
	s.QueryHelper = &baseapp.QueryServiceTestHelper{
		GRPCQueryRouter: s.App.GRPCQueryRouter(),
		Ctx:             s.Ctx,
	}

	s.TestAccs = s.RandomAccountAddresses(3)
}

// Get implements AppOptions
func (ao EmptyBaseAppOptions) Get(_ string) interface{} {
	return nil
}

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

type EmptyAppOptions struct{}

func (EmptyAppOptions) Get(_ string) interface{} { return nil }

func SetupApp(t *testing.T, chainId string) *app.TerraApp {
	t.Helper()

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
		Coins:   sdk.NewCoins(sdk.NewCoin(appparams.BondDenom, sdk.NewInt(100000000000000))),
	}
	genesisAccounts := []authtypes.GenesisAccount{acc}
	app := SetupWithGenesisValSet(t, chainId, valSet, genesisAccounts, balance)

	return app
}

// SetupWithGenesisValSet initializes a new app with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit in the default token of the app from first genesis
// account. A Nop logger is set in app.
func SetupWithGenesisValSet(t *testing.T, chainId string, valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *app.TerraApp {
	t.Helper()

	terraApp, genesisState := setup(chainId)
	genesisState = genesisStateWithValSet(t, terraApp, genesisState, valSet, genAccs, balances...)

	stateBytes, err := json.MarshalIndent(genesisState, "", "")
	require.NoError(t, err)

	// init chain will set the validator set and initialize the genesis accounts
	terraApp.InitChain(
		abci.RequestInitChain{
			ChainId:         chainId,
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	// commit genesis changes
	terraApp.Commit()
	terraApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ChainID:            chainId,
		Height:             terraApp.LastBlockHeight() + 1,
		AppHash:            terraApp.LastCommitID().Hash,
		ValidatorsHash:     valSet.Hash(),
		NextValidatorsHash: valSet.Hash(),
	}})

	return terraApp
}

func setup(chainId string) (*app.TerraApp, app.GenesisState) {
	db := dbm.NewMemDB()
	encCdc := app.MakeEncodingConfig()
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[server.FlagInvCheckPeriod] = 5
	appOptions[server.FlagMinGasPrices] = "0luna"

	terraapp := app.NewTerraApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		encCdc,
		simtestutil.EmptyAppOptions{},
		emptyWasmOpts,
		baseapp.SetChainID(chainId),
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
			DelegatorShares:   sdk.OneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			MinSelfDelegation: sdk.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))

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
	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultGenesisState().Params,
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	)

	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	// update mint genesis state
	mintGenesis := minttypes.DefaultGenesisState()
	genesisState[minttypes.ModuleName] = app.AppCodec().MustMarshalJSON(mintGenesis)

	// update distribution genesis state
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

	// update wasm genesis state
	wasmGenesis := &wasmtypes.GenesisState{
		Params: wasmtypes.DefaultParams(),
	}
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
	err := banktestutil.FundAccount(s.App.BankKeeper, s.Ctx, acc, amounts)
	s.Require().NoError(err)
}
