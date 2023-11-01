package classictax

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/x/classictax/keeper"
	"github.com/classic-terra/core/v2/x/classictax/types"

	core "github.com/classic-terra/core/v2/types"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	// Check epoch last block
	if !core.IsPeriodLastBlock(ctx, 10*core.BlocksPerMinute) {
		return
	}

	ctx.Logger().Info("End Epoch")

}
