package wasm

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LegacyQueryHandler wraps contract queries to apply legacy store translation for historical heights
type LegacyQueryHandler struct {
	next     wasmkeeper.WasmVMQueryHandler
	storeKey storetypes.StoreKey
}

// NewLegacyQueryHandler creates a query handler that wraps contract queries with legacy store support
func NewLegacyQueryHandler(next wasmkeeper.WasmVMQueryHandler, storeKey storetypes.StoreKey) wasmkeeper.WasmVMQueryHandler {
	return &LegacyQueryHandler{
		next:     next,
		storeKey: storeKey,
	}
}

// HandleQuery intercepts contract queries and wraps the context with legacy store if needed
func (h *LegacyQueryHandler) HandleQuery(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
	// Check if we need legacy translation for this height
	preMigration := isPreWasmKeyMigration(ctx.ChainID(), ctx.BlockHeight())

	if preMigration {
		// Wrap context with legacy store
		wrappedCtx, ok := prepareLegacyWasmContext(ctx, h.storeKey)
		if ok {
			ctx = wrappedCtx
			ctx.Logger().Debug("contract query using legacy wasm store",
				"caller", caller.String(),
				"height", ctx.BlockHeight())
		}
	}

	// Execute the query with the (possibly wrapped) context
	return h.next.HandleQuery(ctx, caller, request)
}
