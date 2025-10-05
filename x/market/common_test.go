package market

import (
	"testing"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/keeper"
	"github.com/classic-terra/core/v3/x/market/types"

	sdkmath "cosmossdk.io/math"
)

var randomPrice = sdkmath.LegacyNewDec(1700)

func setup(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	input.MarketKeeper.SetParams(input.Ctx, params)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, randomPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, randomPrice)
	h := keeper.NewMsgServerImpl(input.MarketKeeper)

	return input, h
}
