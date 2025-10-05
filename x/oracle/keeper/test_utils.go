package keeper

import (
	"testing"
	"time"

	customauth "github.com/classic-terra/core/v3/custom/auth"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	sdklog "cosmossdk.io/log"
	"cosmossdk.io/math"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/std"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const faucetAccountName = "faucet"

var ModuleBasics = module.NewBasicManager(
	customauth.AppModuleBasic{},
	custombank.AppModuleBasic{},
	customdistr.AppModuleBasic{},
	customstaking.AppModuleBasic{},
	customparams.AppModuleBasic{},
)

func MakeTestCodec(t *testing.T) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

// EncodingConfig mirrors the fields needed in tests; avoids importing simapp.
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func MakeEncodingConfig(_ *testing.T) EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(codec, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	types.RegisterLegacyAminoCodec(amino)
	types.RegisterInterfaces(interfaceRegistry)
	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// Test addresses
var (
	ValPubKeys = simtestutil.CreateTestPubKeys(5)

	pubKeys = []crypto.PubKey{
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
	}

	Addrs = []sdk.AccAddress{
		sdk.AccAddress(pubKeys[0].Address()),
		sdk.AccAddress(pubKeys[1].Address()),
		sdk.AccAddress(pubKeys[2].Address()),
		sdk.AccAddress(pubKeys[3].Address()),
		sdk.AccAddress(pubKeys[4].Address()),
	}

	ValAddrs = []sdk.ValAddress{
		sdk.ValAddress(pubKeys[0].Address()),
		sdk.ValAddress(pubKeys[1].Address()),
		sdk.ValAddress(pubKeys[2].Address()),
		sdk.ValAddress(pubKeys[3].Address()),
		sdk.ValAddress(pubKeys[4].Address()),
	}

	InitTokens = sdk.TokensFromConsensusPower(200, sdk.DefaultPowerReduction)
	InitCoins  = sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens))

	OracleDecPrecision = 8
)

type TestInput struct {
	Ctx           sdk.Context
	Cdc           *codec.LegacyAmino
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	OracleKeeper  Keeper
	StakingKeeper *stakingkeeper.Keeper
	DistrKeeper   distrkeeper.Keeper
}

func CreateTestInput(t *testing.T) TestInput {
	// Ensure Terra bech32 prefixes are set for address codecs used by keepers
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	sdk.GetConfig().SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	sdk.GetConfig().SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	keyAcc := storetypes.NewKVStoreKey(authtypes.StoreKey)
	keyBank := storetypes.NewKVStoreKey(banktypes.StoreKey)
	keyParams := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	keyOracle := storetypes.NewKVStoreKey(types.StoreKey)
	keyStaking := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	keyDistr := storetypes.NewKVStoreKey(distrtypes.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, sdklog.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyOracle, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistr, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	blackListAddrs := map[string]bool{
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
		faucetAccountName:              true,
	}

	maccPerms := map[string][]string{
		faucetAccountName:              {authtypes.Minter},
		authtypes.FeeCollectorName:     nil,
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          nil,
		types.ModuleName:               nil,
	}

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keyAcc),
		authtypes.ProtoBaseAccount,
		maccPerms,
		accAddrCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(appCodec, runtime.NewKVStoreService(keyBank), accountKeeper, blackListAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String(), sdklog.NewNopLogger())

	totalSupply := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(Addrs)*10))))

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keyStaking),
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = core.MicroLunaDenom
	stakingKeeper.SetParams(ctx, stakingParams)

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keyDistr),
		accountKeeper, bankKeeper, stakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	distrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = math.LegacyNewDecWithPrec(2, 2)
	distrParams.BaseProposerReward = math.LegacyNewDecWithPrec(1, 2)
	distrParams.BonusProposerReward = math.LegacyNewDecWithPrec(4, 2)
	distrKeeper.Params.Set(ctx, distrParams)
	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(distrKeeper.Hooks()))

	faucetAcc := authtypes.NewEmptyModuleAccount(faucetAccountName, authtypes.Minter)
	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	oracleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter)

	// assign unique account numbers for module accounts
	faucetAccI := accountKeeper.NewAccount(ctx, faucetAcc)
	accountKeeper.SetModuleAccount(ctx, faucetAccI.(sdk.ModuleAccountI))
	feeCollectorAccI := accountKeeper.NewAccount(ctx, feeCollectorAcc)
	accountKeeper.SetModuleAccount(ctx, feeCollectorAccI.(sdk.ModuleAccountI))
	bondPoolAccI := accountKeeper.NewAccount(ctx, bondPool)
	accountKeeper.SetModuleAccount(ctx, bondPoolAccI.(sdk.ModuleAccountI))
	notBondedPoolAccI := accountKeeper.NewAccount(ctx, notBondedPool)
	accountKeeper.SetModuleAccount(ctx, notBondedPoolAccI.(sdk.ModuleAccountI))
	distrAccI := accountKeeper.NewAccount(ctx, distrAcc)
	accountKeeper.SetModuleAccount(ctx, distrAccI.(sdk.ModuleAccountI))
	oracleAccI := accountKeeper.NewAccount(ctx, oracleAcc)
	accountKeeper.SetModuleAccount(ctx, oracleAccI.(sdk.ModuleAccountI))

	// now that module accounts exist, mint faucet supply
	require.NoError(t, bankKeeper.MintCoins(ctx, faucetAccountName, totalSupply))
	bankKeeper.SendCoinsFromModuleToModule(ctx, faucetAccountName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(Addrs))))))

	for _, addr := range Addrs {
		// allocate a unique account number for base accounts too
		baseAcc := authtypes.NewBaseAccountWithAddress(addr)
		accI := accountKeeper.NewAccount(ctx, baseAcc)
		accountKeeper.SetAccount(ctx, accI)
		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, faucetAccountName, addr, InitCoins)
		require.NoError(t, err)
	}

	keeper := NewKeeper(
		appCodec,
		keyOracle,
		paramsKeeper.Subspace(types.ModuleName),
		accountKeeper,
		bankKeeper,
		distrKeeper,
		stakingKeeper,
		distrtypes.ModuleName,
	)

	defaults := types.DefaultParams()
	keeper.SetParams(ctx, defaults)

	for _, denom := range defaults.Whitelist {
		keeper.SetTobinTax(ctx, denom.Name, denom.TobinTax)
	}

	return TestInput{ctx, legacyAmino, accountKeeper, bankKeeper, keeper, stakingKeeper, distrKeeper}
}

// NewTestMsgCreateValidator test msg creator
func NewTestMsgCreateValidator(address sdk.ValAddress, pubKey cryptotypes.PubKey, amt math.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())
	desc := stakingtypes.Description{Moniker: "TestValidator"}
	msg, _ := stakingtypes.NewMsgCreateValidator(
		address.String(), pubKey, sdk.NewCoin(core.MicroLunaDenom, amt),
		desc, commission, math.OneInt(),
	)

	return msg
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
func FundAccount(input TestInput, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, amounts); err != nil {
		return err
	}

	return input.BankKeeper.SendCoinsFromModuleToAccount(input.Ctx, faucetAccountName, addr, amounts)
}
