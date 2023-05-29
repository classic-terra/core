package wasmbinding

import (
	"os"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/app"
	testhelpers "github.com/classic-terra/core/app/helpers"
	core "github.com/classic-terra/core/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CreateTestInput(t *testing.T) (*app.TerraApp, sdk.Context) {
	t.Helper()

	app := testhelpers.Setup(t, false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Height: 1, Time: time.Now().UTC()})
	return app, ctx
}

func InstantiateContract(t *testing.T, ctx sdk.Context, app *app.TerraApp, addr sdk.AccAddress, contractDir string) sdk.AccAddress {
	t.Helper()

	wasmKeeper := app.WasmKeeper

	codeId := storeReflectCode(t, ctx, app, addr, contractDir)

	cInfo := wasmKeeper.GetCodeInfo(ctx, codeId)
	require.NotNil(t, cInfo)

	contractAddr := instantiateContract(t, ctx, app, addr, codeId)

	// check if contract is instantiated
	info := wasmKeeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, info)

	return contractAddr
}

// we need to make this deterministic (same every test run), as content might affect gas costs
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func storeReflectCode(t *testing.T, ctx sdk.Context, app *app.TerraApp, addr sdk.AccAddress, contractDir string) uint64 {
	t.Helper()

	wasmCode, err := os.ReadFile(contractDir)
	require.NoError(t, err)

	codeId, _, err := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper).Create(ctx, addr, wasmCode, &wasmtypes.AllowEverybody)
	require.NoError(t, err)

	return codeId
}

func instantiateContract(t *testing.T, ctx sdk.Context, app *app.TerraApp, funder sdk.AccAddress, codeId uint64) sdk.AccAddress {
	t.Helper()

	initMsgBz := []byte("{}")
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	addr, _, err := contractKeeper.Instantiate(ctx, codeId, funder, funder, initMsgBz, nil)
	require.NoError(t, err)

	return addr
}

func RandomAccountAddress() sdk.AccAddress {
	_, _, addr := keyPubAddr()
	return addr
}

func FundAccount(t *testing.T, ctx sdk.Context, app *app.TerraApp, acct sdk.AccAddress) {
	t.Helper()
	err := simapp.FundAccount(app.BankKeeper, ctx, acct, sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(10000000000)),
	))
	require.NoError(t, err)
}
