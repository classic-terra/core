package wasmbinding

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/classic-terra/core/app"
	core "github.com/classic-terra/core/types"
	"github.com/classic-terra/core/wasmbinding/bindings"
	markettypes "github.com/classic-terra/core/x/market/types"
	"github.com/stretchr/testify/require"
)

// go test -v -run ^TestSwap$ github.com/classic-terra/core/wasmbinding/test
// oracle rate: 1 uluna = 1.7 usdr
// 1000 uluna from trader goes to contract
// 1666 usdr (after 2% tax) is swapped into which goes back to contract
func Swap(t *testing.T, contractDir string, executeFunc func(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TerraMsg, funds sdk.Coin) error) {
	t.Helper()

	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateContract(t, ctx, app, actor, contractDir)
	require.NotEmpty(t, contractAddr)

	// setup swap environment
	// Set Oracle Price
	lunaPriceInSDR := sdk.NewDecWithPrec(17, 1)
	app.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroSDRDenom, lunaPriceInSDR)

	actorBeforeSwap := app.BankKeeper.GetAllBalances(ctx, actor)
	contractBeforeSwap := app.BankKeeper.GetAllBalances(ctx, contractAddr)

	// Calculate expected swapped SDR
	expectedSwappedSDR := sdk.NewDec(1000).Mul(lunaPriceInSDR)
	tax := markettypes.DefaultMinStabilitySpread.Mul(expectedSwappedSDR)
	expectedSwappedSDR = expectedSwappedSDR.Sub(tax)

	// execute custom Msg
	msg := bindings.TerraMsg{
		Swap: &bindings.Swap{
			OfferCoin: sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1000)),
			AskDenom:  core.MicroSDRDenom,
		},
	}

	err := executeFunc(t, ctx, app, contractAddr, actor, msg, sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1000)))
	require.NoError(t, err)

	// check result after swap
	actorAfterSwap := app.BankKeeper.GetAllBalances(ctx, actor)
	contractAfterSwap := app.BankKeeper.GetAllBalances(ctx, contractAddr)

	require.Equal(t, actorBeforeSwap.AmountOf(core.MicroLunaDenom).Sub(sdk.NewInt(1000)), actorAfterSwap.AmountOf(core.MicroLunaDenom))
	require.Equal(t, contractBeforeSwap.AmountOf(core.MicroSDRDenom).Add(expectedSwappedSDR.TruncateInt()), contractAfterSwap.AmountOf(core.MicroSDRDenom))
}

// go test -v -run ^TestSwapSend$ github.com/classic-terra/core/wasmbinding/test
// oracle rate: 1 uluna = 1.7 usdr
// 1000 uluna from trader goes to contract
// 1666 usdr (after 2% tax) is swapped into which goes back to contract
// 1666 usdr is sent to trader
func SwapSend(t *testing.T, contractDir string, executeFunc func(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TerraMsg, funds sdk.Coin) error) {
	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateContract(t, ctx, app, actor, contractDir)
	require.NotEmpty(t, contractAddr)

	// setup swap environment
	// Set Oracle Price
	lunaPriceInSDR := sdk.NewDecWithPrec(17, 1)
	app.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroSDRDenom, lunaPriceInSDR)

	actorBeforeSwap := app.BankKeeper.GetAllBalances(ctx, actor)

	// Calculate expected swapped SDR
	expectedSwappedSDR := sdk.NewDec(1000).Mul(lunaPriceInSDR)
	tax := markettypes.DefaultMinStabilitySpread.Mul(expectedSwappedSDR)
	expectedSwappedSDR = expectedSwappedSDR.Sub(tax)

	// execute custom Msg
	msg := bindings.TerraMsg{
		SwapSend: &bindings.SwapSend{
			ToAddress: actor.String(),
			OfferCoin: sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1000)),
			AskDenom:  core.MicroSDRDenom,
		},
	}

	err := executeFunc(t, ctx, app, contractAddr, actor, msg, sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1000)))
	require.NoError(t, err)

	// check result after swap
	actorAfterSwap := app.BankKeeper.GetAllBalances(ctx, actor)
	expectedActorAfterSwap := actorBeforeSwap.Sub(sdk.NewCoins(sdk.NewInt64Coin(core.MicroLunaDenom, 1000)))
	expectedActorAfterSwap = expectedActorAfterSwap.Add(sdk.NewCoin(core.MicroSDRDenom, expectedSwappedSDR.TruncateInt()))

	require.Equal(t, expectedActorAfterSwap, actorAfterSwap)
}

type ReflectExec struct {
	ReflectMsg    *ReflectMsgs    `json:"reflect_msg,omitempty"`
	ReflectSubMsg *ReflectSubMsgs `json:"reflect_sub_msg,omitempty"`
}

type ReflectMsgs struct {
	Msgs []wasmvmtypes.CosmosMsg `json:"msgs"`
}

type ReflectSubMsgs struct {
	Msgs []wasmvmtypes.SubMsg `json:"msgs"`
}

func executeCustom(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TerraMsg, funds sdk.Coin) error {
	t.Helper()

	customBz, err := json.Marshal(msg)
	require.NoError(t, err)
	reflectMsg := ReflectExec{
		ReflectMsg: &ReflectMsgs{
			Msgs: []wasmvmtypes.CosmosMsg{{
				Custom: customBz,
			}},
		},
	}
	reflectBz, err := json.Marshal(reflectMsg)
	require.NoError(t, err)

	// no funds sent if amount is 0
	var coins sdk.Coins
	if !funds.Amount.IsNil() {
		coins = sdk.Coins{funds}
	}

	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	_, err = contractKeeper.Execute(ctx, contract, sender, reflectBz, coins)
	return err
}

type customSwap struct {
	Swap *bindings.Swap `json:"swap"`
}

type customSwapSend struct {
	SwapSend *bindings.SwapSend `json:"swap_send"`
}

func executeOldBindings(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TerraMsg, funds sdk.Coin) error {
	t.Helper()

	var reflectBz []byte
	switch {
	case msg.Swap != nil:
		customSwap := customSwap{
			Swap: msg.Swap,
		}
		var err error
		reflectBz, err = json.Marshal(customSwap)
		require.NoError(t, err)
	case msg.SwapSend != nil:
		customSwapSend := customSwapSend{
			SwapSend: msg.SwapSend,
		}
		var err error
		reflectBz, err = json.Marshal(customSwapSend)
		require.NoError(t, err)
	}

	// no funds sent if amount is 0
	var coins sdk.Coins
	if !funds.Amount.IsNil() {
		coins = sdk.Coins{funds}
	}

	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	_, err := contractKeeper.Execute(ctx, contract, sender, reflectBz, coins)
	return err
}
