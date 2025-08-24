package wasm

import (
	"context"
	"reflect"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	v13 "github.com/classic-terra/core/v3/app/upgrades/v13"
	legacyupgrade "github.com/classic-terra/core/v3/custom/upgrade/legacy"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	LegacyParamStoreKeyUploadAccess      = []byte("uploadAccess")
	LegacyParamStoreKeyInstantiateAccess = []byte("instantiateAccess")
)

// LegacyWasmParams is a wrapper around wasmtypes.Params that implements ParamSet
type LegacyWasmParams struct {
	wasmtypes.Params
}

// LegacyQueryServer wraps the wasm QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	wasmtypes.QueryServer
	keeper *wasmkeeper.Keeper
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer wasmtypes.QueryServer,
	keeper *wasmkeeper.Keeper,
) wasmtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer: originalServer,
		keeper:      keeper,
	}
}

func (q *LegacyQueryServer) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if legacyMode == legacyupgrade.LegacyHandlingNone {
		return q.QueryServer.SmartContractState(ctx, req)
	}

	var result []byte
	var queryErr error

	hasTimeIssue := sdkCtx.BlockTime().IsZero() || sdkCtx.BlockTime().Unix() <= 0
	modifiedCtx := sdk.UnwrapSDKContext(ctx)

	// If we fixed the block time, apply it to the new context, it is not the correct historic time
	if hasTimeIssue {
		baseTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		defaultTime := baseTime.Add(time.Duration(sdkCtx.BlockHeight()) * time.Minute)
		modifiedCtx = modifiedCtx.WithBlockTime(defaultTime)
	}

	// Use direct query with keeper for all pre-upgrade heights
	result, queryErr = q.keeper.QuerySmart(modifiedCtx, sdk.MustAccAddressFromBech32(req.Address), req.QueryData)
	// If the direct query was successful, return the result
	if queryErr == nil {
		return &wasmtypes.QuerySmartContractStateResponse{Data: result}, nil
	}

	return nil, queryErr
}

// ContractHistory gets the contract code history with legacy fallback support
func (q *LegacyQueryServer) ContractHistory(ctx context.Context, req *wasmtypes.QueryContractHistoryRequest) (*wasmtypes.QueryContractHistoryResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Try normal path first
	resp, err := q.QueryServer.ContractHistory(ctx, req)
	if err == nil && resp != nil && len(resp.Entries) > 0 {
		return resp, nil
	}

	// Legacy handling: fall back for archived pre-upgrade data
	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if legacyMode == legacyupgrade.LegacyHandlingNone {
		return resp, err
	}

	// Reflect storeKey and codec from keeper (private fields)
	keVal := reflect.ValueOf(q.keeper).Elem()
	storeKeyField := keVal.FieldByName("storeKey")
	cdcField := keVal.FieldByName("cdc")
	if !storeKeyField.IsValid() || !cdcField.IsValid() {
		// Cannot access internals; return original result
		return resp, err
	}

	storeKey, ok := storeKeyField.Interface().(storetypes.StoreKey)
	if !ok {
		return resp, err
	}
	cdc, ok := cdcField.Interface().(codecMarshaler)
	if !ok {
		return resp, err
	}

	kv := sdkCtx.KVStore(storeKey)
	contractAddr := sdk.MustAccAddressFromBech32(req.Address)

	var entries []wasmtypes.ContractCodeHistoryEntry
	v13.IterateContractHistoryWithFallback(kv, func(addr []byte, history []byte) bool {
		if !sdk.AccAddress(addr).Equals(contractAddr) {
			return true
		}
		var entry wasmtypes.ContractCodeHistoryEntry
		if err := cdc.Unmarshal(history, &entry); err == nil {
			entries = append(entries, entry)
		}
		return true
	})

	// If we found entries via fallback, return them
	if len(entries) > 0 {
		return &wasmtypes.QueryContractHistoryResponse{Entries: entries}, nil
	}

	// Otherwise, return the original response/error
	return resp, err
}

// codecMarshaler defines just the Unmarshal API we need from the keeper codec
type codecMarshaler interface {
	Unmarshal(bz []byte, ptr interface{}) error
}
