package wasmbinding

import (
	"testing"
)

const (
	TERRA_BINDINGS_DIR     = "../testdata/terra_reflect.wasm"
	TERRA_OLD_BINDINGS_DIR = "../testdata/old/bindings_tester.wasm"
)

func TestCustom(t *testing.T) {
	// Msg
	t.Run("TestSwap", func(t *testing.T) {
		Swap(t, TERRA_BINDINGS_DIR)
	})
	t.Run("TestSwapSend", func(t *testing.T) {
		SwapSend(t, TERRA_BINDINGS_DIR)
	})

	// Query
	t.Run("TestQuerySwap", func(t *testing.T) {
		QuerySwap(t, TERRA_BINDINGS_DIR)
	})
	t.Run("TestQueryExchangeRates", func(t *testing.T) {
		QueryExchangeRates(t, TERRA_BINDINGS_DIR)
	})
	t.Run("TestQueryTaxRate", func(t *testing.T) {
		QueryTaxRate(t, TERRA_BINDINGS_DIR)
	})
	t.Run("TestQueryTaxCap", func(t *testing.T) {
		QueryTaxCap(t, TERRA_BINDINGS_DIR)
	})
}

// go test -v -run ^TestOldCustom$ github.com/classic-terra/core/wasmbinding/test
func TestOldCustom(t *testing.T) {
	// Msg
	t.Run("TestSwap", func(t *testing.T) {
		Swap(t, TERRA_OLD_BINDINGS_DIR)
	})
	t.Run("TestSwapSend", func(t *testing.T) {
		SwapSend(t, TERRA_OLD_BINDINGS_DIR)
	})

	// Query
	t.Run("TestQuerySwap", func(t *testing.T) {
		QuerySwap(t, TERRA_OLD_BINDINGS_DIR)
	})
	t.Run("TestQueryExchangeRates", func(t *testing.T) {
		QueryExchangeRates(t, TERRA_OLD_BINDINGS_DIR)
	})
	t.Run("TestQueryTaxRate", func(t *testing.T) {
		QueryTaxRate(t, TERRA_OLD_BINDINGS_DIR)
	})
	t.Run("TestQueryTaxCap", func(t *testing.T) {
		QueryTaxCap(t, TERRA_OLD_BINDINGS_DIR)
	})
}
