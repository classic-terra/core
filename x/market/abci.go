package market

import (
	"github.com/classic-terra/core/v4/x/market/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Replenishes each pools towards equilibrium
	k.ReplenishPools(ctx)
}
