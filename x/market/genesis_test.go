package market

import (
	"testing"

	"github.com/classic-terra/core/v3/x/market/keeper"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
)

func TestExportInitGenesis(t *testing.T) {
	input := keeper.CreateTestInput(t)
	input.MarketKeeper.SetTerraPoolDelta(input.Ctx, sdkmath.LegacyNewDec(1123))
	genesis := ExportGenesis(input.Ctx, input.MarketKeeper)

	newInput := keeper.CreateTestInput(t)
	InitGenesis(newInput.Ctx, newInput.MarketKeeper, genesis)
	newGenesis := ExportGenesis(newInput.Ctx, newInput.MarketKeeper)

	require.Equal(t, genesis, newGenesis)
}
