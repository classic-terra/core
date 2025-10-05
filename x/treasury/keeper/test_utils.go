package keeper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customauth "github.com/classic-terra/core/v3/custom/auth"
	custombank "github.com/classic-terra/core/v3/custom/bank"
	customdistr "github.com/classic-terra/core/v3/custom/distribution"
	customparams "github.com/classic-terra/core/v3/custom/params"
	customstaking "github.com/classic-terra/core/v3/custom/staking"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market"
	marketkeeper "github.com/classic-terra/core/v3/x/market/keeper"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	"github.com/classic-terra/core/v3/x/oracle"
	oraclekeeper "github.com/classic-terra/core/v3/x/oracle/keeper"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/classic-terra/core/v3/x/treasury/types"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	sdklog "cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
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

// Adapter wrappers to satisfy expected interfaces across modules (SDK v0.50)
type oracleAccountAdapter struct{ ak authkeeper.AccountKeeper }

func (a oracleAccountAdapter) GetModuleAddress(name string) sdk.AccAddress {
	return a.ak.GetModuleAddress(name)
}
func (a oracleAccountAdapter) GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI {
	return a.ak.GetModuleAccount(ctx.Context(), moduleName)
}
func (a oracleAccountAdapter) GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	acc := a.ak.GetAccount(ctx.Context(), addr)
	if acc == nil {
		return nil
	}
	if aa, ok := acc.(authtypes.AccountI); ok {
		return aa
	}
	return nil
}

type oracleBankAdapter struct{ bk bankkeeper.BaseKeeper }

func (a oracleBankAdapter) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return a.bk.GetBalance(ctx.Context(), addr, denom)
}
func (a oracleBankAdapter) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return a.bk.GetAllBalances(ctx.Context(), addr)
}
func (a oracleBankAdapter) SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return a.bk.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}
func (a oracleBankAdapter) GetDenomMetaData(ctx sdk.Context, denom string) (banktypes.Metadata, bool) {
	return a.bk.GetDenomMetaData(ctx.Context(), denom)
}
func (a oracleBankAdapter) SetDenomMetaData(ctx sdk.Context, md banktypes.Metadata) {
	a.bk.SetDenomMetaData(ctx.Context(), md)
}
func (a oracleBankAdapter) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return a.bk.SpendableCoins(ctx.Context(), addr)
}

type oracleDistrAdapter struct{ dk distrkeeper.Keeper }

func (a oracleDistrAdapter) AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) {
	_ = a.dk.AllocateTokensToValidator(ctx.Context(), val, tokens)
}
func (a oracleDistrAdapter) GetValidatorOutstandingRewardsCoins(ctx sdk.Context, val sdk.ValAddress) sdk.DecCoins {
	return sdk.DecCoins{}
}

type oracleStakingAdapter struct{ sk *stakingkeeper.Keeper }

func (a oracleStakingAdapter) Validator(ctx context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error) {
	v, _ := a.sk.Validator(ctx, address)
	return v, nil
}
func (a oracleStakingAdapter) TotalBondedTokens(_ context.Context) (sdkmath.Int, error) {
	p, _ := a.sk.TotalBondedTokens(context.Background())
	return p, nil
}
func (a oracleStakingAdapter) Slash(ctx context.Context, cons sdk.ConsAddress, height int64, power int64, frac sdkmath.LegacyDec) (sdkmath.Int, error) {
	return a.sk.Slash(ctx, cons, height, power, frac)
}
func (a oracleStakingAdapter) Jail(ctx context.Context, cons sdk.ConsAddress) error {
	return a.sk.Jail(ctx, cons)
}
func (a oracleStakingAdapter) ValidatorsPowerStoreIterator(ctx context.Context) (storetypes.Iterator, error) {
	return nil, nil
}
func (a oracleStakingAdapter) MaxValidators(ctx context.Context) (uint32, error) {
	mv, _ := a.sk.MaxValidators(ctx)
	return mv, nil
}
func (a oracleStakingAdapter) PowerReduction(ctx context.Context) (res sdkmath.Int) {
	return a.sk.PowerReduction(ctx)
}

type marketAccountAdapter struct{ ak authkeeper.AccountKeeper }

func (a marketAccountAdapter) GetModuleAddress(name string) sdk.AccAddress {
	return a.ak.GetModuleAddress(name)
}
func (a marketAccountAdapter) GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI {
	return a.ak.GetModuleAccount(ctx.Context(), moduleName)
}
func (a marketAccountAdapter) GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	acc := a.ak.GetAccount(ctx.Context(), addr)
	if acc == nil {
		return nil
	}
	if aa, ok := acc.(authtypes.AccountI); ok {
		return aa
	}
	return nil
}

type marketBankAdapter struct{ bk bankkeeper.BaseKeeper }

func (a marketBankAdapter) SendCoinsFromModuleToModule(ctx sdk.Context, s, r string, amt sdk.Coins) error {
	return a.bk.SendCoinsFromModuleToModule(ctx, s, r, amt)
}
func (a marketBankAdapter) SendCoinsFromModuleToAccount(ctx sdk.Context, s string, r sdk.AccAddress, amt sdk.Coins) error {
	return a.bk.SendCoinsFromModuleToAccount(ctx, s, r, amt)
}
func (a marketBankAdapter) SendCoinsFromAccountToModule(ctx sdk.Context, s sdk.AccAddress, r string, amt sdk.Coins) error {
	return a.bk.SendCoinsFromAccountToModule(ctx, s, r, amt)
}
func (a marketBankAdapter) BurnCoins(ctx sdk.Context, name string, amt sdk.Coins) error {
	return a.bk.BurnCoins(ctx.Context(), name, amt)
}
func (a marketBankAdapter) MintCoins(ctx sdk.Context, name string, amt sdk.Coins) error {
	return a.bk.MintCoins(ctx.Context(), name, amt)
}
func (a marketBankAdapter) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return a.bk.SpendableCoins(ctx.Context(), addr)
}
func (a marketBankAdapter) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return a.bk.GetBalance(ctx.Context(), addr, denom)
}
func (a marketBankAdapter) IsSendEnabledCoin(ctx sdk.Context, coin sdk.Coin) bool {
	return a.bk.IsSendEnabledCoin(ctx.Context(), coin)
}

type treasuryAccountAdapter struct{ ak authkeeper.AccountKeeper }

func (a treasuryAccountAdapter) GetModuleAddress(name string) sdk.AccAddress {
	return a.ak.GetModuleAddress(name)
}
func (a treasuryAccountAdapter) GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI {
	return a.ak.GetModuleAccount(ctx, moduleName)
}

func (a treasuryAccountAdapter) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return a.ak.GetAccount(ctx, addr)
}

type treasuryBankAdapter struct{ bk bankkeeper.BaseKeeper }

func (a treasuryBankAdapter) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return a.bk.MintCoins(ctx, moduleName, amt)
}
func (a treasuryBankAdapter) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return a.bk.BurnCoins(ctx, moduleName, amt)
}
func (a treasuryBankAdapter) SendCoinsFromModuleToAccount(ctx context.Context, s string, r sdk.AccAddress, amt sdk.Coins) error {
	return a.bk.SendCoinsFromModuleToAccount(ctx, s, r, amt)
}
func (a treasuryBankAdapter) SendCoinsFromAccountToModule(ctx context.Context, s sdk.AccAddress, r string, amt sdk.Coins) error {
	return a.bk.SendCoinsFromAccountToModule(ctx, s, r, amt)
}
func (a treasuryBankAdapter) SendCoinsFromModuleToModule(ctx context.Context, s, r string, amt sdk.Coins) error {
	return a.bk.SendCoinsFromModuleToModule(ctx, s, r, amt)
}
func (a treasuryBankAdapter) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return a.bk.GetAllBalances(ctx, addr)
}
func (a treasuryBankAdapter) GetSupply(ctx context.Context, denom string) sdk.Coin {
	return a.bk.GetSupply(ctx, denom)
}
func (a treasuryBankAdapter) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return a.bk.GetBalance(ctx, addr, denom)
}
func (a treasuryBankAdapter) GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool) {
	return a.bk.GetDenomMetaData(ctx, denom)
}
func (a treasuryBankAdapter) SetDenomMetaData(ctx context.Context, md banktypes.Metadata) {
	a.bk.SetDenomMetaData(ctx, md)
}
func (a treasuryBankAdapter) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return a.bk.SpendableCoins(ctx, addr)
}
func (a treasuryBankAdapter) IsSendEnabledCoin(ctx context.Context, coin sdk.Coin) bool {
	return a.bk.IsSendEnabledCoin(ctx, coin)
}

type treasuryDistrAdapter struct {
	dk   distrkeeper.Keeper
	pool distrtypes.FeePool
}

func (a *treasuryDistrAdapter) GetFeePool(_ sdk.Context) (feePool distrtypes.FeePool) {
	return a.pool
}
func (a *treasuryDistrAdapter) SetFeePool(_ sdk.Context, feePool distrtypes.FeePool) {
	a.pool = feePool
}

func (a *treasuryDistrAdapter) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
	return a.dk.AllocateTokensToValidator(ctx, val, tokens)
}
func (a *treasuryDistrAdapter) GetValidatorOutstandingRewardsCoins(_ context.Context, _ sdk.ValAddress) (sdk.DecCoins, error) {
	return sdk.DecCoins{}, nil
}

const faucetAccountName = "faucet"

var ModuleBasics = module.NewBasicManager(
	customauth.AppModuleBasic{},
	custombank.AppModuleBasic{},
	customdistr.AppModuleBasic{},
	customstaking.AppModuleBasic{},
	customparams.AppModuleBasic{},
	oracle.AppModuleBasic{},
	market.AppModuleBasic{},
)

// EncodingConfig mirrors the fields needed in tests; avoids importing simapp.
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	Amino             *codec.LegacyAmino
}

func MakeTestCodec(t *testing.T) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

func MakeEncodingConfig(_ *testing.T) EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
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
	Ctx            sdk.Context
	Cdc            *codec.LegacyAmino
	TreasuryKeeper Keeper
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	StakingKeeper  *stakingkeeper.Keeper
	MarketKeeper   types.MarketKeeper
	OracleKeeper   types.OracleKeeper
}

func CreateTestInput(t *testing.T) TestInput {
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	sdk.GetConfig().SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	sdk.GetConfig().SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)

	keyAcc := storetypes.NewKVStoreKey(authtypes.StoreKey)
	keyBank := storetypes.NewKVStoreKey(banktypes.StoreKey)
	keyParams := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	keyOracle := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	keyStaking := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	keyDistr := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	keyMarket := storetypes.NewKVStoreKey(markettypes.StoreKey)
	keyTreasury := storetypes.NewKVStoreKey(types.StoreKey)

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
	ms.MountStoreWithDB(keyTreasury, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	blackListAddrs := map[string]bool{
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
		oracletypes.ModuleName:         true,
		faucetAccountName:              true,
	}

	maccPerms := map[string][]string{
		faucetAccountName:              {authtypes.Minter, authtypes.Burner},
		authtypes.FeeCollectorName:     nil,
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		markettypes.ModuleName:         {authtypes.Burner, authtypes.Minter},
		distrtypes.ModuleName:          nil,
		oracletypes.ModuleName:         nil,
		types.ModuleName:               {authtypes.Burner, authtypes.Minter},
		types.BurnModuleName:           {authtypes.Burner},
	}

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	accountKeeper := authkeeper.NewAccountKeeper(appCodec, runtime.NewKVStoreService(keyAcc), authtypes.ProtoBaseAccount, maccPerms, accAddrCodec, sdk.GetConfig().GetBech32AccountAddrPrefix(), authtypes.NewModuleAddress(govtypes.ModuleName).String())
	bankKeeper := bankkeeper.NewBaseKeeper(appCodec, runtime.NewKVStoreService(keyBank), accountKeeper, blackListAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String(), sdklog.NewNopLogger())

	totalSupply := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(Addrs)*10))))

	stakingKeeper := stakingkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keyStaking), accountKeeper, bankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(), accAddrCodec, valAddrCodec)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = core.MicroLunaDenom
	stakingKeeper.SetParams(ctx, stakingParams)

	distrKeeper := distrkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keyDistr), accountKeeper, bankKeeper, stakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	distrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())

	// Note: default params are set; adjust in tests as needed.
	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(distrKeeper.Hooks()))

	faucetAcc := authtypes.NewEmptyModuleAccount(faucetAccountName, authtypes.Minter, authtypes.Burner)
	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	oracleAcc := authtypes.NewEmptyModuleAccount(oracletypes.ModuleName)
	marketAcc := authtypes.NewEmptyModuleAccount(markettypes.ModuleName, authtypes.Burner, authtypes.Minter)
	treasuryAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Burner, authtypes.Minter)
	burnAcc := authtypes.NewEmptyModuleAccount(types.BurnModuleName, authtypes.Burner)

	// create module accounts with unique account numbers
	faucetAccI := accountKeeper.NewAccount(ctx, faucetAcc)
	accountKeeper.SetModuleAccount(ctx, faucetAccI.(authtypes.ModuleAccountI))
	feeCollectorAccI := accountKeeper.NewAccount(ctx, feeCollectorAcc)
	accountKeeper.SetModuleAccount(ctx, feeCollectorAccI.(authtypes.ModuleAccountI))
	bondPoolI := accountKeeper.NewAccount(ctx, bondPool)
	accountKeeper.SetModuleAccount(ctx, bondPoolI.(authtypes.ModuleAccountI))
	notBondedPoolI := accountKeeper.NewAccount(ctx, notBondedPool)
	accountKeeper.SetModuleAccount(ctx, notBondedPoolI.(authtypes.ModuleAccountI))
	distrAccI := accountKeeper.NewAccount(ctx, distrAcc)
	accountKeeper.SetModuleAccount(ctx, distrAccI.(authtypes.ModuleAccountI))
	oracleAccI := accountKeeper.NewAccount(ctx, oracleAcc)
	accountKeeper.SetModuleAccount(ctx, oracleAccI.(authtypes.ModuleAccountI))
	marketAccI := accountKeeper.NewAccount(ctx, marketAcc)
	accountKeeper.SetModuleAccount(ctx, marketAccI.(authtypes.ModuleAccountI))
	treasuryAccI := accountKeeper.NewAccount(ctx, treasuryAcc)
	accountKeeper.SetModuleAccount(ctx, treasuryAccI.(authtypes.ModuleAccountI))
	burnAccI := accountKeeper.NewAccount(ctx, burnAcc)
	accountKeeper.SetModuleAccount(ctx, burnAccI.(authtypes.ModuleAccountI))

	// now mint faucet supply and seed staking not bonded pool
	require.NoError(t, bankKeeper.MintCoins(ctx, faucetAccountName, totalSupply))
	require.NoError(t, bankKeeper.SendCoinsFromModuleToModule(ctx, faucetAccountName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, InitTokens.MulRaw(int64(len(Addrs)+1))))))

	for _, addr := range Addrs {
		base := authtypes.NewBaseAccountWithAddress(addr)
		acc := accountKeeper.NewAccount(ctx, base)
		accountKeeper.SetAccount(ctx, acc)
		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, faucetAccountName, addr, InitCoins)
		require.NoError(t, err)
	}

	// to test burn module account
	err := bankKeeper.SendCoinsFromModuleToModule(ctx, faucetAccountName, types.BurnModuleName, InitCoins)
	require.NoError(t, err)

	// wasm not required for these unit tests; pass nil into treasury keeper

	oracleKeeper := oraclekeeper.NewKeeper(
		appCodec,
		keyOracle,
		paramsKeeper.Subspace(oracletypes.ModuleName),
		treasuryAccountAdapter{accountKeeper},
		treasuryBankAdapter{bankKeeper},
		&treasuryDistrAdapter{dk: distrKeeper},
		oracleStakingAdapter{stakingKeeper},
		distrtypes.ModuleName,
	)
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleKeeper.SetParams(ctx, oracleDefaultParams)

	for _, denom := range oracleDefaultParams.Whitelist {
		oracleKeeper.SetTobinTax(ctx, denom.Name, denom.TobinTax)
	}

	marketKeeper := marketkeeper.NewKeeper(
		appCodec,
		keyMarket, paramsKeeper.Subspace(markettypes.ModuleName),
		treasuryAccountAdapter{accountKeeper},
		treasuryBankAdapter{bankKeeper},
		oracleKeeper,
	)
	marketKeeper.SetParams(ctx, markettypes.DefaultParams())

	treasuryKeeper := NewKeeper(
		appCodec,
		keyTreasury, paramsKeeper.Subspace(types.ModuleName),
		treasuryAccountAdapter{accountKeeper},
		treasuryBankAdapter{bankKeeper},
		marketKeeper,
		oracleKeeper,
		stakingKeeper,
		distrKeeper,
		nil,
		distrtypes.ModuleName,
	)

	treasuryKeeper.SetParams(ctx, types.DefaultParams())

	return TestInput{ctx, legacyAmino, treasuryKeeper, accountKeeper, bankKeeper, distrKeeper, stakingKeeper, marketKeeper, oracleKeeper}
}

// NewTestMsgCreateValidator test msg creator
func NewTestMsgCreateValidator(address sdk.ValAddress, pubKey cryptotypes.PubKey, amt sdkmath.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec())
	msg, _ := stakingtypes.NewMsgCreateValidator(
		sdk.AccAddress(address).String(), pubKey, sdk.NewCoin(core.MicroLunaDenom, amt),
		stakingtypes.Description{Moniker: "TestValidator"}, commission, sdkmath.NewInt(1),
	)

	return msg
}

func setupValidators(t *testing.T) (TestInput, stakingtypes.MsgServer) {
	input := CreateTestInput(t)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)

	// Create Validators
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	_, err := stakingMsgSvr.CreateValidator(input.Ctx, NewTestMsgCreateValidator(addr, val, amt))

	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(input.Ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)

	return input, stakingMsgSvr
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
