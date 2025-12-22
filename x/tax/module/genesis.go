package module

import (
	"github.com/classic-terra/core/v4/x/tax/keeper"
	"github.com/classic-terra/core/v4/x/tax/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {
	keeper.SetParams(ctx, data.Params)
}
