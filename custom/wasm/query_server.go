package wasm

import (
	"context"
	"reflect"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v3/custom/upgrade/legacy"
	core "github.com/classic-terra/core/v3/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
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

	legacyMode := legacy.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	if legacyMode == legacy.LegacyHandlingNone {
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
	// Try normal path first
	resp, err := q.QueryServer.ContractHistory(ctx, req)
	if err == nil && resp != nil && len(resp.Entries) > 0 {
		return resp, nil
	}

	// Legacy handling: fall back for archived pre-upgrade data
	if !q.shouldUseLegacyFallback(ctx) {
		return resp, err
	}

	storeKey, cdc, ok := q.getStoreKeyAndCodec()
	if !ok {
		return resp, err
	}

	kv := sdk.UnwrapSDKContext(ctx).KVStore(storeKey)
	contractAddr := sdk.MustAccAddressFromBech32(req.Address)

	var entries []wasmtypes.ContractCodeHistoryEntry
	core.IterateContractHistoryWithFallback(kv, func(addr []byte, history []byte) bool {
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

// ContractInfo gets contract info with legacy fallback support
func (q *LegacyQueryServer) ContractInfo(ctx context.Context, req *wasmtypes.QueryContractInfoRequest) (*wasmtypes.QueryContractInfoResponse, error) {
	resp, err := q.QueryServer.ContractInfo(ctx, req)
	if err == nil && resp != nil && resp.Address != "" {
		return resp, nil
	}

	if !q.shouldUseLegacyFallback(ctx) {
		return resp, err
	}

	storeKey, cdc, ok := q.getStoreKeyAndCodec()
	if !ok {
		return resp, err
	}
	kv := sdk.UnwrapSDKContext(ctx).KVStore(storeKey)
	addr := sdk.MustAccAddressFromBech32(req.Address)

	bz, found := core.ReadContractInfoWithFallback(kv, addr)
	if !found {
		return resp, err
	}
	var info wasmtypes.ContractInfo
	if e := cdc.Unmarshal(bz, &info); e != nil {
		return resp, err
	}

	return &wasmtypes.QueryContractInfoResponse{Address: req.Address, ContractInfo: info}, nil
}

// RawContractState reads a specific key with legacy fallback
func (q *LegacyQueryServer) RawContractState(ctx context.Context, req *wasmtypes.QueryRawContractStateRequest) (*wasmtypes.QueryRawContractStateResponse, error) {
	resp, err := q.QueryServer.RawContractState(ctx, req)
	if err == nil && resp != nil && len(resp.Data) > 0 {
		return resp, nil
	}

	if !q.shouldUseLegacyFallback(ctx) {
		return resp, err
	}

	storeKey, _, ok := q.getStoreKeyAndCodec()
	if !ok {
		return resp, err
	}
	kv := sdk.UnwrapSDKContext(ctx).KVStore(storeKey)
	addr := sdk.MustAccAddressFromBech32(req.Address)

	bz, found := core.ReadRawContractStateWithFallback(kv, addr, req.QueryData)
	if !found {
		return resp, err
	}
	return &wasmtypes.QueryRawContractStateResponse{Data: bz}, nil
}

// AllContractState lists all state entries with legacy fallback
func (q *LegacyQueryServer) AllContractState(ctx context.Context, req *wasmtypes.QueryAllContractStateRequest) (*wasmtypes.QueryAllContractStateResponse, error) {
	resp, err := q.QueryServer.AllContractState(ctx, req)
	if err == nil && resp != nil && len(resp.Models) > 0 {
		return resp, nil
	}

	if !q.shouldUseLegacyFallback(ctx) {
		return resp, err
	}

	storeKey, _, ok := q.getStoreKeyAndCodec()
	if !ok {
		return resp, err
	}
	kv := sdk.UnwrapSDKContext(ctx).KVStore(storeKey)
	addr := sdk.MustAccAddressFromBech32(req.Address)

	var models []wasmtypes.Model
	core.IterateAllContractStateWithFallback(kv, addr, func(k, v []byte) bool {
		models = append(models, wasmtypes.Model{Key: append([]byte{}, k...), Value: append([]byte{}, v...)})
		return true
	})
	if len(models) == 0 {
		return resp, err
	}
	return &wasmtypes.QueryAllContractStateResponse{Models: models}, nil
}

// ContractsByCode falls back to old secondary index if needed
func (q *LegacyQueryServer) ContractsByCode(ctx context.Context, req *wasmtypes.QueryContractsByCodeRequest) (*wasmtypes.QueryContractsByCodeResponse, error) {
	resp, err := q.QueryServer.ContractsByCode(ctx, req)
	if err == nil && resp != nil && len(resp.Contracts) > 0 {
		return resp, nil
	}

	if !q.shouldUseLegacyFallback(ctx) {
		return resp, err
	}

	storeKey, _, ok := q.getStoreKeyAndCodec()
	if !ok {
		return resp, err
	}
	kv := sdk.UnwrapSDKContext(ctx).KVStore(storeKey)

	// New secondary index after migration: 0x06 | codeID(8) | addr
	// Old secondary index: 0x10 | codeID(8) | addr
	codeIDBz := sdk.Uint64ToBigEndian(req.CodeId)

	// Try new
	newIndexPrefix := append([]byte{0x06}, codeIDBz...)
	newStore := prefix.NewStore(kv, newIndexPrefix)
	newIter := newStore.Iterator(nil, nil)
	contracts := make([]string, 0)
	for ; newIter.Valid(); newIter.Next() {
		addr := sdk.AccAddress(newIter.Value())
		contracts = append(contracts, addr.String())
	}
	newIter.Close()
	if len(contracts) > 0 {
		return &wasmtypes.QueryContractsByCodeResponse{Contracts: contracts}, nil
	}

	// Try old
	oldIndexPrefix := append([]byte{0x10}, codeIDBz...)
	oldStore := prefix.NewStore(kv, oldIndexPrefix)
	oldIter := oldStore.Iterator(nil, nil)
	for ; oldIter.Valid(); oldIter.Next() {
		addr := sdk.AccAddress(oldIter.Value())
		contracts = append(contracts, addr.String())
	}
	oldIter.Close()
	if len(contracts) > 0 {
		return &wasmtypes.QueryContractsByCodeResponse{Contracts: contracts}, nil
	}

	return resp, err
}

// codecMarshaler defines just the Unmarshal API we need from the keeper codec
type codecMarshaler interface {
	Unmarshal(bz []byte, ptr interface{}) error
}

// shouldUseLegacyFallback checks if legacy fallback should be used for the current context
func (q *LegacyQueryServer) shouldUseLegacyFallback(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	legacyMode := legacy.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	return legacyMode != legacy.LegacyHandlingNone
}

// helper to reflect store key and codec
func (q *LegacyQueryServer) getStoreKeyAndCodec() (storetypes.StoreKey, codecMarshaler, bool) {
	keVal := reflect.ValueOf(q.keeper).Elem()
	storeKeyField := keVal.FieldByName("storeKey")
	cdcField := keVal.FieldByName("cdc")
	if !storeKeyField.IsValid() || !cdcField.IsValid() {
		return nil, nil, false
	}
	sk, ok1 := storeKeyField.Interface().(storetypes.StoreKey)
	cdc, ok2 := cdcField.Interface().(codecMarshaler)
	if !ok1 || !ok2 {
		return nil, nil, false
	}
	return sk, cdc, true
}
