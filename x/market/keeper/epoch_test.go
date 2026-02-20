package keeper

import (
	"testing"

	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestEpoch_BurnAndRefill(t *testing.T) {
	input := CreateTestInput(t)

	marketAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	accumAddr := input.AccountKeeper.GetModuleAddress(types.AccumulatorModuleName)

	// Seed balances: market has 1_000_000 uusd; accumulator has 5_000_000 uusd
	preMarket := sdk.NewCoins(sdk.NewInt64Coin(core.MicroUSDDenom, 1_000_000))
	preAccum := sdk.NewCoins(sdk.NewInt64Coin(core.MicroUSDDenom, 5_000_000))

	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, preMarket))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(input.Ctx, faucetAccountName, types.ModuleName, preMarket))
	require.Equal(t, preMarket, input.BankKeeper.GetAllBalances(input.Ctx, marketAddr))

	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, preAccum))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(input.Ctx, faucetAccountName, types.AccumulatorModuleName, preAccum))
	require.Equal(t, preAccum, input.BankKeeper.GetAllBalances(input.Ctx, accumAddr))

	// Set non-zero height and trigger epoch processing: since last epoch is 0, it should process now
	input.Ctx = input.Ctx.WithBlockHeight(1)
	input.MarketKeeper.ProcessEpochIfDue(input.Ctx)
	input.MarketKeeper.ReplenishPools(input.Ctx)

	// Market balance should equal pre-accumulator (burned its own pre balance then refilled)
	require.Equal(t, preAccum, input.BankKeeper.GetAllBalances(input.Ctx, marketAddr))
	// Accumulator should be empty
	require.True(t, input.BankKeeper.GetAllBalances(input.Ctx, accumAddr).Empty())
}

func TestEpoch_NoProcessBeforeEpoch(t *testing.T) {
	input := CreateTestInput(t)

	marketAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	accumAddr := input.AccountKeeper.GetModuleAddress(types.AccumulatorModuleName)

	// First processing to set last epoch height at height 1
	initial := sdk.NewCoins(sdk.NewInt64Coin(core.MicroUSDDenom, 100_000))
	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, initial))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(input.Ctx, faucetAccountName, types.AccumulatorModuleName, initial))
	input.Ctx = input.Ctx.WithBlockHeight(1)
	input.MarketKeeper.ProcessEpochIfDue(input.Ctx)
	input.MarketKeeper.ReplenishPools(input.Ctx)
	require.Equal(t, initial, input.BankKeeper.GetAllBalances(input.Ctx, marketAddr))
	require.True(t, input.BankKeeper.GetAllBalances(input.Ctx, accumAddr).Empty())

	// Mint new amounts to both accounts
	moreMarket := sdk.NewCoins(sdk.NewInt64Coin(core.MicroUSDDenom, 222_222))
	moreAccum := sdk.NewCoins(sdk.NewInt64Coin(core.MicroUSDDenom, 333_333))
	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, moreMarket))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(input.Ctx, faucetAccountName, types.ModuleName, moreMarket))
	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, moreAccum))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(input.Ctx, faucetAccountName, types.AccumulatorModuleName, moreAccum))

	// Advance height but not enough for epoch: should NOT process epoch
	input.Ctx = input.Ctx.WithBlockHeight(2)
	input.MarketKeeper.ProcessEpochIfDue(input.Ctx)
	input.MarketKeeper.ReplenishPools(input.Ctx)

	// Balances remain unchanged
	expectedMarket := initial.Add(moreMarket...)
	require.Equal(t, expectedMarket, input.BankKeeper.GetAllBalances(input.Ctx, marketAddr))
	require.Equal(t, moreAccum, input.BankKeeper.GetAllBalances(input.Ctx, accumAddr))
}
