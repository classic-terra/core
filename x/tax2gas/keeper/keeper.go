package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/classic-terra/core/v3/x/tax2gas/types"
)


type Keeper struct {
	storeKey storetypes.StoreKey

	paramSpace paramtypes.Subspace
}

func NewKeeper(storeKey storetypes.StoreKey, paramSpace paramtypes.Subspace) *Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{storeKey: storeKey, paramSpace: paramSpace}
}

// GetParams returns the total set of testmodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of testmodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

// InitGenesis initializes the testmodule module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the testmodule module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}
