package wasmbinding

import (
	"testing"

	"github.com/classic-terra/core/app"
	"github.com/classic-terra/core/wasmbinding/bindings"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TERRA_BINDINGS_DIR           = "../testdata/terra_reflect.wasm"
	TERRA_RENOVATED_BINDINGS_DIR = "../testdata/old/bindings_tester.wasm"
)

func TestBindingsAll(t *testing.T) {
	cases := []struct {
		name        string
		dir         string
		executeFunc func(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TerraMsg, funds sdk.Coin) error
		queryFunc   func(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, request bindings.TerraQuery, response interface{})
	}{
		{
			name:        "Terra",
			dir:         TERRA_BINDINGS_DIR,
			executeFunc: executeCustom,
			queryFunc:   queryCustom,
		},
		{
			name:        "Old Terra bindings",
			dir:         TERRA_RENOVATED_BINDINGS_DIR,
			executeFunc: executeOldBindings,
			queryFunc:   queryOldBindings,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Msg
			t.Run("TestSwap", func(t *testing.T) {
				Swap(t, tc.dir, tc.executeFunc)
			})
			t.Run("TestSwapSend", func(t *testing.T) {
				SwapSend(t, tc.dir, tc.executeFunc)
			})

			// Query
			t.Run("TestQuerySwap", func(t *testing.T) {
				QuerySwap(t, tc.dir, tc.queryFunc)
			})
			t.Run("TestQueryExchangeRates", func(t *testing.T) {
				QueryExchangeRates(t, tc.dir, tc.queryFunc)
			})
			t.Run("TestQueryTaxRate", func(t *testing.T) {
				QueryTaxRate(t, tc.dir, tc.queryFunc)
			})
			t.Run("TestQueryTaxCap", func(t *testing.T) {
				QueryTaxCap(t, tc.dir, tc.queryFunc)
			})
		})
	}
}
