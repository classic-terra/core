package keeper

//nolint
//DONTCOVER

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storemetrics "cosmossdk.io/store/metrics"
	"github.com/stretchr/testify/require"

	customauth "github.com/classic-terra/core/v3/custom/auth"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	core "github.com/classic-terra/core/v3/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	sdkmath "cosmossdk.io/math"
	store "cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	types "github.com/classic-terra/core/v3/x/dyncomm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkcrypto "github.com/cosmos/cosmos-sdk/crypto/types"
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
	customstaking.AppModuleBasic{},
	customdistr.AppModuleBasic{},
	customparams.AppModuleBasic{},
)

// MakeTestCodec
func MakeTestCodec(t *testing.T) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

// MakeEncodingConfig
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          clientTxConfig
	Amino             *codec.LegacyAmino
}

// minimal alias to avoid importing client interfaces
type clientTxConfig interface{}

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
	PubKeys = simtestutil.CreateTestPubKeys(32)

	InitTokens    = sdk.TokensFromConsensusPower(10_000, sdk.DefaultPowerReduction)
	InitCoins     = sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens))
	DelegateCoins = sdk.NewCoin(core.MicroLunaDenom, InitTokens)

	blackListAddrs = map[string]bool{
		faucetAccountName:              true,
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
	}

	maccPerms = map[string][]string{
		faucetAccountName:              {authtypes.Minter},
		authtypes.FeeCollectorName:     nil,
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          nil,
		types.ModuleName:               {authtypes.Burner, authtypes.Minter},
	}
)

type TestInput struct {
	Ctx           sdk.Context
	Cdc           *codec.LegacyAmino
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	DistrKeeper   distrkeeper.Keeper
	StakingKeeper *stakingkeeper.Keeper
	DyncommKeeper Keeper
}

func CreateTestInput(t *testing.T) TestInput {
	keyAcc := storetypes.NewKVStoreKey(authtypes.StoreKey)
	keyBank := storetypes.NewKVStoreKey(banktypes.StoreKey)
	keyParams := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	keyStaking := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	keyDistr := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	keyDyncomm := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistr, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDyncomm, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

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
	bankKeeper := bankkeeper.NewBaseKeeper(appCodec, runtime.NewKVStoreService(keyBank), accountKeeper, blackListAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String(), log.NewNopLogger())

	totalSupply := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, math.Int(math.LegacyNewDec(1_000_000_000_000))))

	faucetAcc := authtypes.NewEmptyModuleAccount(faucetAccountName, authtypes.Minter)
	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)

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

	err := bankKeeper.MintCoins(ctx, faucetAccountName, totalSupply)
	require.NoError(t, err)

	err = bankKeeper.SendCoinsFromModuleToModule(
		ctx, faucetAccountName, stakingtypes.NotBondedPoolName,
		sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(PubKeys))))),
	)
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

	for idx := range PubKeys {
		baseAcc := authtypes.NewBaseAccountWithAddress(AddrFrom(idx))
		accI := accountKeeper.NewAccount(ctx, baseAcc)
		accountKeeper.SetAccount(ctx, accI)
		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, faucetAccountName, AddrFrom(idx), InitCoins)
		require.NoError(t, err)
	}

	dyncommKeeper := NewKeeper(
		appCodec, keyDyncomm,
		paramsKeeper.Subspace(types.ModuleName),
		stakingKeeper,
	)
	dyncommKeeper.SetParams(
		ctx, types.DefaultParams(),
	)

	return TestInput{ctx, legacyAmino, accountKeeper, bankKeeper, distrKeeper, stakingKeeper, dyncommKeeper}
}

func CallCreateValidatorHooks(ctx sdk.Context, k distrkeeper.Keeper, addr sdk.AccAddress, valAddr sdk.ValAddress) error {
	err := k.Hooks().AfterValidatorCreated(ctx, valAddr)
	if err != nil {
		return err
	}
	err = k.Hooks().BeforeDelegationCreated(ctx, addr, valAddr)
	if err != nil {
		return err
	}
	err = k.Hooks().AfterDelegationModified(ctx, addr, valAddr)
	if err != nil {
		return err
	}
	return nil
}

func CreateValidator(idx int, stake math.Int) (stakingtypes.Validator, error) {
	val, err := stakingtypes.NewValidator(
		ValAddrFrom(idx).String(), PubKeys[idx], stakingtypes.Description{Moniker: "TestValidator"},
	)
	val.Tokens = stake
	val.DelegatorShares = sdkmath.LegacyNewDec(val.Tokens.Int64())
	return val, err
}

func GetPubKey(idx int) (sdkcrypto.PubKey, sdk.AccAddress, sdk.ValAddress) {
	addr := AddrFrom(idx)
	valAddr := ValAddrFrom(idx)
	return PubKeys[idx], addr, valAddr
}

func AddrFrom(idx int) sdk.AccAddress {
	return sdk.AccAddress(PubKeys[idx].Address())
}

func ValAddrFrom(idx int) sdk.ValAddress {
	return sdk.ValAddress(PubKeys[idx].Address())
}
