package keeper

import (
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/stretchr/testify/require"

	simappparams "cosmossdk.io/simapp/params"
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

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
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

func MakeEncodingConfig(_ *testing.T) simappparams.EncodingConfig {
	encodingConfig := simappparams.MakeTestEncodingConfig()
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
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

	keyTaxExemption := sdk.NewKVStoreKey(types.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	aKeyParams := sdk.NewKVStoreKey(authtypes.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyTaxExemption, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil, // just added to enable align fee
		govtypes.ModuleName:        {authtypes.Burner},
		wasm.ModuleName:            {authtypes.Burner},
	}

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accountKeeper := authkeeper.NewAccountKeeper(appCodec, aKeyParams,
		authtypes.ProtoBaseAccount,
		maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(), string(authtypes.NewModuleAddress(govtypes.ModuleName)))

	taxexemptionKeeper := NewKeeper(appCodec,
		keyTaxExemption, paramsKeeper.Subspace(types.ModuleName),
		accountKeeper,
		string(accountKeeper.GetModuleAddress(govtypes.ModuleName)),
	)

	return TestInput{ctx, legacyAmino, taxexemptionKeeper}
}
