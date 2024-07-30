package taxexemption

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState() *types.GenesisState {
	return &types.GenesisState{}
}

// DefaultGenesisState gets raw genesis raw message for testing
func DefaultGenesisState() *types.GenesisState {
	return &types.GenesisState{}
}

// InitGenesis initializes default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) (data *types.GenesisState) {
	return types.NewGenesisState()
}
