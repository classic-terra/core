package simulation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// Simulation operation weights constants
//
//nolint:gosec
const (
	OpWeightMsgStoreCode           = "op_weight_msg_store_code"
	OpWeightMsgInstantiateContract = "op_weight_msg_instantiate_contract"
	OpWeightMsgExecuteContract     = "op_weight_msg_execute_contract"
	OpWeightMsgUpdateAdmin         = "op_weight_msg_update_admin"
	OpWeightMsgClearAdmin          = "op_weight_msg_clear_admin"
	OpWeightMsgMigrateContract     = "op_weight_msg_migrate_contract"
	OpReflectContractPath          = "op_reflect_contract_path"
)

// WasmKeeper is a subset of the wasm keeper used by simulations
type WasmKeeper interface {
	GetParams(ctx sdk.Context) types.Params
	GetAuthority() string
	IterateCodeInfos(ctx sdk.Context, cb func(uint64, types.CodeInfo) bool)
	IterateContractInfo(ctx sdk.Context, cb func(sdk.AccAddress, types.ContractInfo) bool)
	QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
	PeekAutoIncrementID(ctx sdk.Context, lastIDKey []byte) uint64
}
type BankKeeper interface {
	simulation.BankKeeper
	IsSendEnabledCoin(ctx sdk.Context, coin sdk.Coin) bool
}

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	simstate *module.SimulationState,
	ak types.AccountKeeper,
	bk BankKeeper,
	wasmKeeper WasmKeeper,
) simulation.WeightedOperations {
	// SDK v0.50 migration: disable custom wasm simulation ops for now
	// to keep the application buildable. Re-enable with updated APIs later.
	return simulation.WeightedOperations{}
}
