package keeper

import (
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	customauth "github.com/classic-terra/core/v3/custom/auth"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market"
	"github.com/classic-terra/core/v3/x/oracle"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
)

var ModuleBasics = module.NewBasicManager(
	customauth.AppModuleBasic{},
	custombank.AppModuleBasic{},
	customdistr.AppModuleBasic{},
	customstaking.AppModuleBasic{},
	customparams.AppModuleBasic{},
	oracle.AppModuleBasic{},
	market.AppModuleBasic{},
)

func MakeTestCodec(t *testing.T) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	Amino             *codec.LegacyAmino
}

func MakeEncodingConfig(_ *testing.T) EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             cdc,
		Amino:             amino,
	}
}

var (
	ValPubKeys = simtestutil.CreateTestPubKeys(5)

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
	Ctx                sdk.Context
	Cdc                *codec.LegacyAmino
	TaxExemptionKeeper Keeper
}

func CreateTestInput(t *testing.T) TestInput {
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	accAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	keyTaxExemption := storetypes.NewKVStoreKey(types.StoreKey)
	keyParams := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	aKeyParams := storetypes.NewKVStoreKey(authtypes.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, sdklog.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyTaxExemption, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(aKeyParams, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil, // just added to enable align fee
		govtypes.ModuleName:        {authtypes.Burner},
		wasmtypes.ModuleName:       {authtypes.Burner},
	}

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(aKeyParams),
		authtypes.ProtoBaseAccount,
		maccPerms,
		accAddrCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	taxexemptionKeeper := NewKeeper(appCodec,
		keyTaxExemption, paramsKeeper.Subspace(types.ModuleName),
		accountKeeper,
		string(accountKeeper.GetModuleAddress(govtypes.ModuleName)),
	)

	return TestInput{ctx, legacyAmino, taxexemptionKeeper}
}
