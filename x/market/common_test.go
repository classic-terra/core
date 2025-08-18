package market

import (
	"testing"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/keeper"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var randomPrice = sdk.NewDec(1700)

func setup(t *testing.T) (keeper.TestInput, sdk.Handler) {
	input := keeper.CreateTestInput(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	input.MarketKeeper.SetParams(input.Ctx, params)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, randomPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, randomPrice)
	// Set required meta USD rate for oracle guard in market swaps
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, randomPrice)

	// Seed market module pool with liquidity for ask denoms used in tests
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroUSDDenom, sdk.NewInt(1_000_000_000)),
		sdk.NewCoin(core.MicroSDRDenom, sdk.NewInt(1_000_000_000)),
	)
	_ = input.BankKeeper.MintCoins(input.Ctx, markettypes.ModuleName, poolCoins)

	h := NewHandler(input.MarketKeeper)

	return input, h
}
