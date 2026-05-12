package keeper

import (
	"context"
	"testing"
	"time"

	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	coretypes "github.com/classic-terra/core/v4/types"
	chronotypes "github.com/classic-terra/core/v4/x/cron/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
)

type fakeAccountKeeper struct {
	moduleAddr sdk.AccAddress
}

func (f fakeAccountKeeper) GetModuleAddress(_ string) sdk.AccAddress {
	return f.moduleAddr
}

type fakeWasmMsgServer struct {
	calls []*chronotypes.MsgExecuteContract
	err   error
}

func (f *fakeWasmMsgServer) ExecuteContract(_ context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	f.calls = append(f.calls, &chronotypes.MsgExecuteContract{
		Contract: msg.Contract,
		Msg:      string(msg.Msg),
	})
	if f.err != nil {
		return nil, f.err
	}
	return &wasmtypes.MsgExecuteContractResponse{}, nil
}

type testEncodingConfig struct {
	Codec codec.Codec
	Amino *codec.LegacyAmino
}

func makeTestEncodingConfig(t *testing.T) testEncodingConfig {
	t.Helper()
	amino := codec.NewLegacyAmino()
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	chronotypes.RegisterLegacyAminoCodec(amino)
	chronotypes.RegisterInterfaces(registry)

	return testEncodingConfig{Codec: cdc, Amino: amino}
}

type testInput struct {
	Ctx        sdk.Context
	Keeper     Keeper
	AccountKey sdk.AccAddress
	StoreKey   storetypes.StoreKey
	MsgServer  *fakeWasmMsgServer
}

func createTestInput(t *testing.T) testInput {
	t.Helper()
	sdk.GetConfig().SetBech32PrefixForAccount(coretypes.Bech32PrefixAccAddr, coretypes.Bech32PrefixAccPub)
	sdk.GetConfig().SetBech32PrefixForValidator(coretypes.Bech32PrefixValAddr, coretypes.Bech32PrefixValPub)
	sdk.GetConfig().SetBech32PrefixForConsensusNode(coretypes.Bech32PrefixConsAddr, coretypes.Bech32PrefixConsPub)

	keyCron := storetypes.NewKVStoreKey(chronotypes.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, sdklog.NewNopLogger())

	enc := makeTestEncodingConfig(t)
	moduleAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	fakeKeeper := fakeAccountKeeper{moduleAddr: moduleAddr}
	msgServer := &fakeWasmMsgServer{}

	ms.MountStoreWithDB(keyCron, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	k := NewKeeper(enc.Codec, keyCron, fakeKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	k.WasmMsgServer = msgServer

	return testInput{
		Ctx:        ctx,
		Keeper:     k,
		AccountKey: moduleAddr,
		StoreKey:   keyCron,
		MsgServer:  msgServer,
	}
}
