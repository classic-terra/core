package keeper

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestOracleFreshnessCheck tests that swaps are denied when oracle data is stale
func TestOracleFreshnessCheck(t *testing.T) {
	input := CreateTestInput(t)

	// Set oracle prices
	lunaPriceInUSD := sdkmath.LegacyNewDecWithPrec(5, 0) // 5 USD per LUNC
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, lunaPriceInUSD)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyOneDec())
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, lunaPriceInUSD)

	// Set up pool liquidity
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(10000000)),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(50000000)),
	)
	require.NoError(t, FundModuleAccount(input, types.ModuleName, poolCoins))

	// Set initial tally time
	initialTime := time.Unix(1000000, 0)
	input.Ctx = input.Ctx.WithBlockTime(initialTime)
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, initialTime.Unix())

	// Test 1: Fresh oracle data - swap should succeed
	input.Ctx = input.Ctx.WithBlockTime(initialTime.Add(30 * time.Second))
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1000000))
	_, _, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroUSDDenom)
	require.NoError(t, err, "swap should succeed with fresh oracle data")

	// Test 2: Oracle data at max age (75s) - should still work
	input.Ctx = input.Ctx.WithBlockTime(initialTime.Add(75 * time.Second))
	_, _, err = input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroUSDDenom)
	require.NoError(t, err, "swap should succeed at max oracle age")

	// Test 3: Stale oracle data (76s) - swap should fail
	input.Ctx = input.Ctx.WithBlockTime(initialTime.Add(76 * time.Second))
	trader := Addrs[0]
	msgServer := NewMsgServerImpl(input.MarketKeeper)

	swapMsg := types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err, "swap should fail with stale oracle data")
	require.ErrorIs(t, err, types.ErrOraclePriceStale)

	// Test 4: Update tally time - swap should succeed again
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, input.Ctx.BlockTime().Unix())
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err, "swap should succeed after oracle update")
}

// TestTWAPTracking tests TWAP price snapshot tracking
func TestTWAPTracking(t *testing.T) {
	input := CreateTestInput(t)

	denom := oracletypes.MetaUSDDenom // Use MetaUSDDenom as that's what the code tracks

	// Test 1: No TWAP data initially
	_, err := input.MarketKeeper.ComputeTWAP(input.Ctx, denom)
	require.Error(t, err, "should error when no TWAP data exists")

	// Test 2: Add price snapshots
	prices := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDecWithPrec(100, 2), // 1.00
		sdkmath.LegacyNewDecWithPrec(105, 2), // 1.05
		sdkmath.LegacyNewDecWithPrec(110, 2), // 1.10
		sdkmath.LegacyNewDecWithPrec(95, 2),  // 0.95
		sdkmath.LegacyNewDecWithPrec(100, 2), // 1.00
	}

	for i, price := range prices {
		input.Ctx = input.Ctx.WithBlockHeight(int64(i + 1))
		input.MarketKeeper.AddTWAPPrice(input.Ctx, denom, price)
	}

	// Test 3: Compute TWAP (simple average)
	twap, err := input.MarketKeeper.ComputeTWAP(input.Ctx, denom)
	require.NoError(t, err)

	expectedTWAP := sdkmath.LegacyNewDecWithPrec(102, 2) // (1.00 + 1.05 + 1.10 + 0.95 + 1.00) / 5 = 1.02
	require.True(t, twap.Sub(expectedTWAP).Abs().LTE(sdkmath.LegacyNewDecWithPrec(1, 3)),
		"TWAP should be approximately %s, got %s", expectedTWAP, twap)

	// Test 4: Old snapshots are pruned
	lookbackWindow := input.MarketKeeper.TwapLookbackWindow(input.Ctx)
	input.Ctx = input.Ctx.WithBlockHeight(int64(lookbackWindow) + 100)
	input.MarketKeeper.AddTWAPPrice(input.Ctx, denom, sdkmath.LegacyNewDecWithPrec(200, 2))

	// Get TWAP should still work with the new snapshot
	twap2, err := input.MarketKeeper.ComputeTWAP(input.Ctx, denom)
	require.NoError(t, err)
	require.Equal(t, sdkmath.LegacyNewDecWithPrec(200, 2), twap2, "TWAP should be the new price after pruning")
}

// TestTWAPDeviationCheck tests that swaps are denied when price deviates too much from TWAP
func TestTWAPDeviationCheck(t *testing.T) {
	input := CreateTestInput(t)

	trader := Addrs[0]
	msgServer := NewMsgServerImpl(input.MarketKeeper)

	// Set up TWAP with stable price around 1.00 USD
	denom := oracletypes.MetaUSDDenom
	basePrice := sdkmath.LegacyNewDecWithPrec(100, 2) // 1.00 USD

	for i := 0; i < 10; i++ {
		input.Ctx = input.Ctx.WithBlockHeight(int64(i + 1))
		// Add slight variations around 1.00
		variation := sdkmath.LegacyNewDecWithPrec(int64(i%3-1), 2) // -0.01, 0, 0.01
		input.MarketKeeper.AddTWAPPrice(input.Ctx, denom, basePrice.Add(variation))
	}

	// Set oracle prices (need SDR for swap calculations)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyOneDec())
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, basePrice)

	// Set oracle tally time
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, input.Ctx.BlockTime().Unix())

	// Set up pool liquidity
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(10000000)),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(10000000)),
	)
	require.NoError(t, FundModuleAccount(input, types.ModuleName, poolCoins))

	// Test 1: Current price within deviation (5% from TWAP) - should succeed
	currentPrice := sdkmath.LegacyNewDecWithPrec(104, 2) // 1.04 USD (4% deviation)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, denom, currentPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, currentPrice)

	offerCoin := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1000000))
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))

	swapMsg := types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom)
	_, err := msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err, "swap should succeed with 4%% price deviation")

	// Test 2: Current price exceeds max deviation (11% from TWAP) - should fail
	currentPrice = sdkmath.LegacyNewDecWithPrec(112, 2) // 1.12 USD (12% deviation from 1.00)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, denom, currentPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, currentPrice)

	// Fund trader again
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))

	swapMsg = types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	if err == nil {
		t.Logf("Expected error but swap succeeded - checking TWAP data")
		twap, twapErr := input.MarketKeeper.ComputeTWAP(input.Ctx, oracletypes.MetaUSDDenom)
		t.Logf("TWAP: %v, err: %v", twap, twapErr)
		t.Logf("Current price: %v", currentPrice)
	}
	require.Error(t, err, "swap should fail with 12%% price deviation")
	require.ErrorIs(t, err, types.ErrTWAPDeviation)

	// Test 3: Price drops below TWAP by >10% - should also fail
	currentPrice = sdkmath.LegacyNewDecWithPrec(88, 2) // 0.88 USD (12% deviation downward)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, denom, currentPrice)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, currentPrice)

	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))

	swapMsg = types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err, "swap should fail with 12%% downward deviation")
	require.ErrorIs(t, err, types.ErrTWAPDeviation)
}

// TestDailyCapBasicTracking tests basic daily cap tracking functionality
func TestDailyCapBasicTracking(t *testing.T) {
	input := CreateTestInput(t)

	denom := core.MicroLunaDenom
	baseline := sdkmath.NewInt(1000000) // 1M LUNC baseline

	// Test 1: Set and get baseline
	input.MarketKeeper.SetDailyCapBaseline(input.Ctx, denom, baseline)
	retrievedBaseline := input.MarketKeeper.GetDailyCapBaseline(input.Ctx, denom)
	require.Equal(t, baseline, retrievedBaseline)

	// Test 2: Initial usage is zero
	usage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, denom)
	require.True(t, usage.IsZero())

	// Test 3: Set and get usage
	usageAmount := sdkmath.NewInt(50000) // 50k used
	input.MarketKeeper.SetDailyCapUsage(input.Ctx, denom, usageAmount)
	retrievedUsage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, denom)
	require.Equal(t, usageAmount, retrievedUsage)

	// Test 4: Daily reset clears usage
	input.MarketKeeper.SetDailyCapResetHeight(input.Ctx, 100)
	input.Ctx = input.Ctx.WithBlockHeight(100 + int64(core.BlocksPerDay) + 1)

	input.MarketKeeper.ResetDailyCapIfNeeded(input.Ctx)

	// Usage should be cleared
	usage = input.MarketKeeper.GetDailyCapUsage(input.Ctx, denom)
	require.True(t, usage.IsZero(), "usage should be reset after a day")
}

// TestDailyCapEnforcement tests daily cap enforcement during swaps
func TestDailyCapEnforcement(t *testing.T) {
	input := CreateTestInput(t)

	// Set up oracle prices
	lunaPriceInUSD := sdkmath.LegacyNewDecWithPrec(5, 0) // 5 USD per LUNC
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, lunaPriceInUSD)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyOneDec())
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, lunaPriceInUSD)
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, input.Ctx.BlockTime().Unix())

	// Set up pool with baseline
	lunaBaseline := sdkmath.NewInt(1000000) // 1M LUNC
	usdBaseline := sdkmath.NewInt(5000000)  // 5M USD (equivalent value)

	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, lunaBaseline),
		sdk.NewCoin(core.MicroUSDDenom, usdBaseline),
	)
	require.NoError(t, FundModuleAccount(input, types.ModuleName, poolCoins))

	// Set block height and baselines (simulating epoch change)
	input.Ctx = input.Ctx.WithBlockHeight(100)
	input.MarketKeeper.SetDailyCapBaseline(input.Ctx, core.MicroLunaDenom, lunaBaseline)
	input.MarketKeeper.SetDailyCapBaseline(input.Ctx, core.MicroUSDDenom, usdBaseline)
	input.MarketKeeper.SetDailyCapResetHeight(input.Ctx, input.Ctx.BlockHeight())

	trader := Addrs[0]
	msgServer := NewMsgServerImpl(input.MarketKeeper)

	// Daily cap is 10% of baseline = 100k LUNC or 500k USD

	// Test 1: Drain 80k LUNC - should succeed (80k USD at 1:1 ratio = 80k LUNC)
	offerCoin := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(80000)) // 80k USD -> 80k LUNC (1:1 in pool)
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))

	swapMsg := types.NewMsgSwap(trader, offerCoin, core.MicroLunaDenom)
	_, err := msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err, "first swap should succeed (80k LUNC)")

	// Check usage was updated
	usage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, core.MicroLunaDenom)
	require.True(t, usage.GT(sdkmath.ZeroInt()), "usage should be tracked")

	// Test 2: Try to drain another 30k LUNC - should fail (total 110k > 100k cap)
	offerCoin2 := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(30000)) // 30k USD -> 30k LUNC
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin2)))

	swapMsg2 := types.NewMsgSwap(trader, offerCoin2, core.MicroLunaDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg2)
	require.Error(t, err, "second swap should fail (exceeds daily cap: 80k + 30k = 110k > 100k)")
	require.ErrorIs(t, err, types.ErrDailyCapExceeded)

	// Test 3: Swap back (add LUNC to pool) - should reduce usage
	lunaToSwapBack := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(40000)) // 40k LUNC back
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(lunaToSwapBack)))

	swapBackMsg := types.NewMsgSwap(trader, lunaToSwapBack, core.MicroUSDDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapBackMsg)
	require.NoError(t, err, "swap back should succeed")

	// Usage should be reduced
	newUsage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, core.MicroLunaDenom)
	require.True(t, newUsage.LT(usage), "usage should decrease after swapping back")

	// Test 4: After daily reset, can drain again
	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + int64(core.BlocksPerDay) + 1)
	input.MarketKeeper.ResetDailyCapIfNeeded(input.Ctx)

	// Should be able to drain 80k LUNC again
	offerCoin3 := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(80000))
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin3)))

	swapMsg3 := types.NewMsgSwap(trader, offerCoin3, core.MicroLunaDenom)
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg3)
	require.NoError(t, err, "swap should succeed after daily reset")
}

// TestEpochBaselineSetup tests that baselines are set correctly at epoch change
func TestEpochBaselineSetup(t *testing.T) {
	input := CreateTestInput(t)

	// Set up accumulator with funds (mint to faucet first, then send to accumulator)
	accumCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(2000000)),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(10000000)),
	)
	require.NoError(t, FundModuleAccount(input, types.AccumulatorModuleName, accumCoins))

	// Set epoch length and trigger epoch processing
	params := input.MarketKeeper.GetParams(input.Ctx)
	params.EpochLengthBlocks = 100
	input.MarketKeeper.SetParams(input.Ctx, params)

	input.Ctx = input.Ctx.WithBlockHeight(101)
	input.MarketKeeper.ProcessEpochIfDue(input.Ctx)

	// Check that baselines were set to the refilled amounts
	lunaBaseline := input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroLunaDenom)
	usdBaseline := input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroUSDDenom)

	require.Equal(t, sdkmath.NewInt(2000000), lunaBaseline, "LUNC baseline should match refilled amount")
	require.Equal(t, sdkmath.NewInt(10000000), usdBaseline, "USD baseline should match refilled amount")

	// Check that daily reset height was initialized
	resetHeight := input.MarketKeeper.GetDailyCapResetHeight(input.Ctx)
	require.Equal(t, int64(101), resetHeight, "reset height should be set to epoch height")
}

// TestAfterOracleTallyHook tests that the oracle tally hook updates TWAP and timestamp
func TestAfterOracleTallyHook(t *testing.T) {
	input := CreateTestInput(t)

	// Set block time and height first
	tallyTime := time.Unix(2000000, 0)
	input.Ctx = input.Ctx.WithBlockTime(tallyTime).WithBlockHeight(100)

	// Set oracle price AFTER setting context
	ustcPrice := sdkmath.LegacyNewDecWithPrec(102, 2) // 1.02 USD
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, ustcPrice)

	// Verify price was set
	retrievedPrice, err := input.OracleKeeper.GetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom)
	require.NoError(t, err, "should be able to retrieve oracle price")
	require.Equal(t, ustcPrice, retrievedPrice, "retrieved price should match")

	// Call the hook
	input.MarketKeeper.AfterOracleTally(input.Ctx)

	// Test 1: Tally timestamp was updated
	lastTallyTime := input.MarketKeeper.GetLastOracleTallyTime(input.Ctx)
	require.Equal(t, tallyTime.Unix(), lastTallyTime, "tally timestamp should be updated")

	// Test 2: TWAP price was added
	snapshotsMeta := input.MarketKeeper.GetTWAPPrices(input.Ctx, oracletypes.MetaUSDDenom)

	require.Equal(t, 1, len(snapshotsMeta), "should have one TWAP snapshot for MetaUSDDenom")
	require.Equal(t, ustcPrice, snapshotsMeta[0].Price, "TWAP snapshot should have correct price")
	require.Equal(t, int64(100), snapshotsMeta[0].Height, "TWAP snapshot should have correct height")
}

// TestMultipleDenomDailyCap tests daily cap tracking for multiple denoms
func TestMultipleDenomDailyCap(t *testing.T) {
	input := CreateTestInput(t)

	// Set baselines for multiple denoms
	denoms := []string{core.MicroLunaDenom, core.MicroUSDDenom, core.MicroSDRDenom}
	baselines := []sdkmath.Int{
		sdkmath.NewInt(1000000),
		sdkmath.NewInt(5000000),
		sdkmath.NewInt(3000000),
	}

	for i, denom := range denoms {
		input.MarketKeeper.SetDailyCapBaseline(input.Ctx, denom, baselines[i])
	}

	// Set different usage amounts
	usages := []sdkmath.Int{
		sdkmath.NewInt(50000),
		sdkmath.NewInt(250000),
		sdkmath.NewInt(100000),
	}

	for i, denom := range denoms {
		input.MarketKeeper.SetDailyCapUsage(input.Ctx, denom, usages[i])
	}

	// Verify all are tracked independently
	for i, denom := range denoms {
		baseline := input.MarketKeeper.GetDailyCapBaseline(input.Ctx, denom)
		usage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, denom)

		require.Equal(t, baselines[i], baseline, "baseline for %s should match", denom)
		require.Equal(t, usages[i], usage, "usage for %s should match", denom)
	}

	// Reset and verify all are cleared
	input.MarketKeeper.SetDailyCapResetHeight(input.Ctx, 100)
	input.Ctx = input.Ctx.WithBlockHeight(100 + int64(core.BlocksPerDay) + 1)
	input.MarketKeeper.ResetDailyCapIfNeeded(input.Ctx)

	for _, denom := range denoms {
		usage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, denom)
		require.True(t, usage.IsZero(), "usage for %s should be reset", denom)
	}
}

// setupSwapEnv seeds oracle rates, tally time and pool liquidity for swap tests
func setupSwapEnv(t *testing.T, input TestInput, poolCoins sdk.Coins) types.MsgServer {
	t.Helper()

	// UST meta rate (USD per USTC) and the legacy uusd rate (USD per LUNA).
	// usdr is the reference denom ComputeSwap prices through.
	// Both USD rates are 5 so uusd/UST resolves to a 1:1 LUNC<->USTC pool ratio,
	// which keeps the cap arithmetic in these tests readable.
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, sdkmath.LegacyNewDec(5))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyOneDec())
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, sdkmath.LegacyNewDec(5))
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, input.Ctx.BlockTime().Unix())

	require.NoError(t, FundModuleAccount(input, types.ModuleName, poolCoins))

	return NewMsgServerImpl(input.MarketKeeper)
}

// TestDailyCapCountsSwapFee ensures the recorded drain covers payout plus fee.
// The fee is carved out of the trader's payout, so the trader pays nothing extra,
// but it still leaves the pool and must count against the daily cap.
func TestDailyCapCountsSwapFee(t *testing.T) {
	input := CreateTestInput(t)

	lunaBaseline := sdkmath.NewInt(1000000)
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, lunaBaseline),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(5000000)),
	)
	msgServer := setupSwapEnv(t, input, poolCoins)

	input.Ctx = input.Ctx.WithBlockHeight(100)
	input.MarketKeeper.SetDailyCapBaseline(input.Ctx, core.MicroLunaDenom, lunaBaseline)
	input.MarketKeeper.SetDailyCapResetHeight(input.Ctx, input.Ctx.BlockHeight())

	trader := Addrs[0]
	offerCoin := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(50000))
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))

	res, err := msgServer.Swap(sdk.WrapSDKContext(input.Ctx), types.NewMsgSwap(trader, offerCoin, core.MicroLunaDenom))
	require.NoError(t, err)
	require.True(t, res.SwapFee.Amount.IsPositive(), "test needs a positive spread fee to be meaningful")

	usage := input.MarketKeeper.GetDailyCapUsage(input.Ctx, core.MicroLunaDenom)
	require.Equal(t, res.SwapCoin.Amount.Add(res.SwapFee.Amount), usage,
		"daily usage must count the gross drain (payout + fee), not just the payout")
}

// TestDailyCapSeedsMissingBaseline ensures a denom funded mid-epoch is still capped
// instead of being drainable without limit for the rest of the epoch.
func TestDailyCapSeedsMissingBaseline(t *testing.T) {
	input := CreateTestInput(t)

	// Pool is funded but no baselines exist (denom appeared after the epoch boundary)
	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1000000)),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(5000000)),
	)
	msgServer := setupSwapEnv(t, input, poolCoins)

	input.Ctx = input.Ctx.WithBlockHeight(100)
	input.MarketKeeper.SetDailyCapResetHeight(input.Ctx, input.Ctx.BlockHeight())
	require.True(t, input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroLunaDenom).IsZero())

	trader := Addrs[0]

	// Daily cap is 10% of the seeded baseline (1M LUNC) = 100k
	offerCoin := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(80000))
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))
	_, err := msgServer.Swap(sdk.WrapSDKContext(input.Ctx), types.NewMsgSwap(trader, offerCoin, core.MicroLunaDenom))
	require.NoError(t, err, "first swap seeds the baseline from the pool balance and fits the cap")

	baseline := input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroLunaDenom)
	require.Equal(t, sdkmath.NewInt(1000000), baseline, "baseline should be seeded from the pool balance")

	// Second swap pushes past the cap and must be rejected
	offerCoin2 := sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(30000))
	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin2)))
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), types.NewMsgSwap(trader, offerCoin2, core.MicroLunaDenom))
	require.ErrorIs(t, err, types.ErrDailyCapExceeded)
}

// TestEpochClearsStaleCapState ensures epoch processing drops baselines for denoms
// that left the pool and does not carry usage into the new epoch.
func TestEpochClearsStaleCapState(t *testing.T) {
	input := CreateTestInput(t)

	// Stale state from the previous epoch
	input.MarketKeeper.SetDailyCapBaseline(input.Ctx, core.MicroSDRDenom, sdkmath.NewInt(3000000))
	input.MarketKeeper.SetDailyCapUsage(input.Ctx, core.MicroLunaDenom, sdkmath.NewInt(90000))

	accumCoins := sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(2000000)))
	require.NoError(t, FundModuleAccount(input, types.AccumulatorModuleName, accumCoins))

	params := input.MarketKeeper.GetParams(input.Ctx)
	params.EpochLengthBlocks = 100
	input.MarketKeeper.SetParams(input.Ctx, params)

	input.Ctx = input.Ctx.WithBlockHeight(101)
	input.MarketKeeper.ProcessEpochIfDue(input.Ctx)

	require.True(t, input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroSDRDenom).IsZero(),
		"baseline for a denom no longer in the pool must be cleared")
	require.True(t, input.MarketKeeper.GetDailyCapUsage(input.Ctx, core.MicroLunaDenom).IsZero(),
		"usage must not carry into the new epoch")
	require.Equal(t, sdkmath.NewInt(2000000), input.MarketKeeper.GetDailyCapBaseline(input.Ctx, core.MicroLunaDenom))
}

// TestTWAPDeviationUsesMetaUSTRate pins which rate guards a uusd leg.
//
// The UST meta rate is the real USD price of 1 USTC and is the guard input. The
// legacy uusd rate is a LUNC price recorded under the old 1 USTC = 1 USD
// assumption - the very thing the meta rate exists to correct - so a move in it
// alone is not treated as a USTC price deviation.
func TestTWAPDeviationUsesMetaUSTRate(t *testing.T) {
	input := CreateTestInput(t)

	poolCoins := sdk.NewCoins(
		sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(10000000)),
		sdk.NewCoin(core.MicroUSDDenom, sdkmath.NewInt(10000000)),
	)
	msgServer := setupSwapEnv(t, input, poolCoins)

	// Build a TWAP history with both rates stable at 5
	for h := int64(100); h < 105; h++ {
		input.Ctx = input.Ctx.WithBlockHeight(h)
		input.MarketKeeper.AfterOracleTally(input.Ctx)
	}

	metaTWAP, err := input.MarketKeeper.ComputeTWAP(input.Ctx, oracletypes.MetaUSDDenom)
	require.NoError(t, err)
	require.Equal(t, sdkmath.LegacyNewDec(5), metaTWAP)

	trader := Addrs[0]
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(1000))

	// Moving only the legacy uusd rate 20% does not trip the guard
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, sdkmath.LegacyNewDec(6))
	input.MarketKeeper.SetLastOracleTallyTime(input.Ctx, input.Ctx.BlockTime().Unix())

	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom))
	require.NoError(t, err, "legacy uusd rate is not the USTC price and must not gate the swap")

	// Moving the UST meta rate 20% above its TWAP does trip the guard
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, oracletypes.MetaUSDDenom, sdkmath.LegacyNewDec(6))

	require.NoError(t, FundAccount(input, trader, sdk.NewCoins(offerCoin)))
	_, err = msgServer.Swap(sdk.WrapSDKContext(input.Ctx), types.NewMsgSwap(trader, offerCoin, core.MicroUSDDenom))
	require.ErrorIs(t, err, types.ErrTWAPDeviation)
}
