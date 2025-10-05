package keeper

//nolint
//DONTCOVER

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customauth "github.com/classic-terra/core/v3/custom/auth"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/types"
	"github.com/classic-terra/core/v3/x/oracle"
	oraclekeeper "github.com/classic-terra/core/v3/x/oracle/keeper"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/runtime"
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
	customstaking.AppModuleBasic{},
	customdistr.AppModuleBasic{},
	customparams.AppModuleBasic{},
	oracle.AppModuleBasic{},
)

// MakeTestCodec
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

// MakeEncodingConfig
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

// Test Account
var (
	PubKeys = []crypto.PubKey{
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
	}

	Addrs = []sdk.AccAddress{
		sdk.AccAddress(PubKeys[0].Address()),
		sdk.AccAddress(PubKeys[1].Address()),
		sdk.AccAddress(PubKeys[2].Address()),
	}

	ValAddrs = []sdk.ValAddress{
		sdk.ValAddress(PubKeys[0].Address()),
		sdk.ValAddress(PubKeys[1].Address()),
		sdk.ValAddress(PubKeys[2].Address()),
	}

	InitTokens = sdk.TokensFromConsensusPower(200, sdk.DefaultPowerReduction)
	InitCoins  = sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens))
)

type TestInput struct {
	Ctx           sdk.Context
	Cdc           *codec.LegacyAmino
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	OracleKeeper  types.OracleKeeper
	MarketKeeper  Keeper
}

func CreateTestInput(t *testing.T) TestInput {
	keyAcc := storetypes.NewKVStoreKey(authtypes.StoreKey)
	keyBank := storetypes.NewKVStoreKey(banktypes.StoreKey)
	keyParams := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	keyOracle := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	keyStaking := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	keyDistr := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	keyMarket := storetypes.NewKVStoreKey(types.StoreKey)

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
	ms.MountStoreWithDB(keyMarket, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	blackListAddrs := map[string]bool{
		faucetAccountName:              true,
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
	}

	maccPerms := map[string][]string{
		faucetAccountName:              {authtypes.Minter},
		authtypes.FeeCollectorName:     nil,
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          nil,
		oracletypes.ModuleName:         nil,
		types.ModuleName:               {authtypes.Burner, authtypes.Minter},
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
	err := bankKeeper.MintCoins(ctx, faucetAccountName, totalSupply)
	require.NoError(t, err)

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
	distrParams.CommunityTax = sdkmath.LegacyNewDecWithPrec(2, 2)
	distrParams.BaseProposerReward = sdkmath.LegacyNewDecWithPrec(1, 2)
	distrParams.BonusProposerReward = sdkmath.LegacyNewDecWithPrec(4, 2)
	distrKeeper.Params.Set(ctx, distrParams)
	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(distrKeeper.Hooks()))

	faucetAcc := authtypes.NewEmptyModuleAccount(faucetAccountName, authtypes.Minter)
	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	oracleAcc := authtypes.NewEmptyModuleAccount(oracletypes.ModuleName)
	marketAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Burner, authtypes.Minter)

	// assign unique account numbers and set module accounts first
	faucetAccI := accountKeeper.NewAccount(ctx, faucetAcc)
	accountKeeper.SetModuleAccount(ctx, faucetAccI.(authtypes.ModuleAccountI))
	feeCollectorAccI := accountKeeper.NewAccount(ctx, feeCollectorAcc)
	accountKeeper.SetModuleAccount(ctx, feeCollectorAccI.(authtypes.ModuleAccountI))
	bondPoolAccI := accountKeeper.NewAccount(ctx, bondPool)
	accountKeeper.SetModuleAccount(ctx, bondPoolAccI.(authtypes.ModuleAccountI))
	notBondedPoolAccI := accountKeeper.NewAccount(ctx, notBondedPool)
	accountKeeper.SetModuleAccount(ctx, notBondedPoolAccI.(authtypes.ModuleAccountI))
	distrAccI := accountKeeper.NewAccount(ctx, distrAcc)
	accountKeeper.SetModuleAccount(ctx, distrAccI.(authtypes.ModuleAccountI))
	oracleAccI := accountKeeper.NewAccount(ctx, oracleAcc)
	accountKeeper.SetModuleAccount(ctx, oracleAccI.(authtypes.ModuleAccountI))
	marketAccI := accountKeeper.NewAccount(ctx, marketAcc)
	accountKeeper.SetModuleAccount(ctx, marketAccI.(authtypes.ModuleAccountI))

	err = bankKeeper.SendCoinsFromModuleToModule(ctx, faucetAccountName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(Addrs))))))
	require.NoError(t, err)

	for _, addr := range Addrs {
		baseAcc := authtypes.NewBaseAccountWithAddress(addr)
		accI := accountKeeper.NewAccount(ctx, baseAcc)
		accountKeeper.SetAccount(ctx, accI)
		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, faucetAccountName, addr, InitCoins)
		require.NoError(t, err)
		require.Equal(t, bankKeeper.GetAllBalances(ctx, addr), InitCoins)
	}

	oracleKeeper := oraclekeeper.NewKeeper(
		appCodec,
		keyOracle,
		paramsKeeper.Subspace(oracletypes.ModuleName),
		accountKeeper,
		bankKeeper,
		distrKeeper,
		stakingKeeper,
		distrtypes.ModuleName,
	)
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleKeeper.SetParams(ctx, oracleDefaultParams)

	for _, denom := range oracleDefaultParams.Whitelist {
		oracleKeeper.SetTobinTax(ctx, denom.Name, denom.TobinTax)
	}

	keeper := NewKeeper(
		appCodec,
		keyMarket, paramsKeeper.Subspace(types.ModuleName),
		accountKeeper,
		bankKeeper,
		oracleKeeper,
	)
	keeper.SetParams(ctx, types.DefaultParams())

	return TestInput{ctx, legacyAmino, accountKeeper, bankKeeper, oracleKeeper, keeper}
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
