package market

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/keeper"
	"github.com/classic-terra/core/v4/x/market/types"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var randomPrice = sdkmath.LegacyNewDec(1700)

func setup(t *testing.T) (keeper.TestInput, types.MsgServer) {
	input := keeper.CreateTestInput(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	input.MarketKeeper.SetParams(input.Ctx, params)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, randomPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, randomPrice)
	// Set required meta USD rate for oracle guard in market swaps
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, randomPrice)

	// Seed market module pool with liquidity for ask denoms used in tests.
	// Use the faucet (which holds the Minter permission) rather than minting into
	// the market module directly: production grants the market module only Burner.
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(1_000_000_000)),
		sdk.NewCoin(core.MicroSDRDenom, sdkmath.NewInt(1_000_000_000)),
	)
	require.NoError(t, keeper.FundModuleAccount(input, markettypes.ModuleName, poolCoins))

	h := keeper.NewMsgServerImpl(input.MarketKeeper)

	return input, h
}
