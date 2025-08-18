package market

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/market/keeper"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Epoch processing: burn leftover and refill market pool if epoch elapsed
	k.ProcessEpochIfDue(ctx)

	// Replenishes virtual pools towards equilibrium
	k.ReplenishPools(ctx)
}
