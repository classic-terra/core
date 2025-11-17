package keeper

import (
	"testing"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	core "github.com/classic-terra/core/v3/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFeeRewardsForEpoch(t *testing.T) {
	input, _ := setupValidators(t)

	taxAmount := sdkmath.NewInt(1000).MulRaw(core.MicroUnit)

	// Set random prices
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyNewDec(1))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, sdkmath.LegacyNewDec(10))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroGBPDenom, sdkmath.LegacyNewDec(100))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroCNYDenom, sdkmath.LegacyNewDec(1000))

	// Record tax proceeds
	input.TreasuryKeeper.RecordEpochTaxProceeds(input.Ctx, sdk.Coins{
		sdk.NewCoin(core.MicroSDRDenom, taxAmount),
		sdk.NewCoin(core.MicroKRWDenom, taxAmount),
		sdk.NewCoin(core.MicroGBPDenom, taxAmount),
		sdk.NewCoin(core.MicroCNYDenom, taxAmount),
	}.Sort())

	// Update Indicators
	input.TreasuryKeeper.UpdateIndicators(input.Ctx)

	// Get Tax Rawards (TR)
	TR := input.TreasuryKeeper.GetTR(input.Ctx, input.TreasuryKeeper.GetEpoch(input.Ctx))
	require.Equal(t, sdkmath.LegacyNewDec(1111).MulInt64(core.MicroUnit), TR)
}

func TestSeigniorageRewardsForEpoch(t *testing.T) {
	input, _ := setupValidators(t)

	amt := sdkmath.NewInt(1000)
	sdrRate := sdkmath.LegacyNewDec(10)

	// Add seigniorage
	input.TreasuryKeeper.RecordEpochInitialIssuance(input.Ctx)

	// Set random prices
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdrRate)
	input.Ctx = input.Ctx.WithBlockHeight(int64(core.BlocksPerWeek))

	// Add seigniorage
	err := input.BankKeeper.BurnCoins(input.Ctx, faucetAccountName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, amt)))
	require.NoError(t, err)

	// Update Indicators
	input.TreasuryKeeper.UpdateIndicators(input.Ctx)

	// Get seigniorage rewards (SR)
	SR := input.TreasuryKeeper.GetSR(input.Ctx, input.TreasuryKeeper.GetEpoch(input.Ctx))
	miningRewardWeight := input.TreasuryKeeper.GetRewardWeight(input.Ctx)
	require.Equal(t, sdrRate.MulInt(amt).Mul(miningRewardWeight), SR)
}

func TestMiningRewardsForEpoch(t *testing.T) {
	input, _ := setupValidators(t)

	amt := sdkmath.NewInt(1000).MulRaw(core.MicroUnit)
	input.TreasuryKeeper.RecordEpochInitialIssuance(input.Ctx)

	// Set random prices
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyNewDec(1))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, sdkmath.LegacyNewDec(10))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroGBPDenom, sdkmath.LegacyNewDec(100))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroCNYDenom, sdkmath.LegacyNewDec(1000))

	input.Ctx = input.Ctx.WithBlockHeight(int64(core.BlocksPerWeek))

	// Record tax proceeds
	input.TreasuryKeeper.RecordEpochTaxProceeds(input.Ctx, sdk.Coins{
		sdk.NewCoin(core.MicroSDRDenom, amt),
		sdk.NewCoin(core.MicroKRWDenom, amt),
		sdk.NewCoin(core.MicroGBPDenom, amt),
		sdk.NewCoin(core.MicroCNYDenom, amt),
	}.Sort())

	// Add seigniorage
	err := input.BankKeeper.BurnCoins(input.Ctx, faucetAccountName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, amt)))
	require.NoError(t, err)

	input.TreasuryKeeper.UpdateIndicators(input.Ctx)

	epoch := input.TreasuryKeeper.GetEpoch(input.Ctx)

	tProceeds := input.TreasuryKeeper.GetTR(input.Ctx, epoch)
	sProceeds := input.TreasuryKeeper.GetSR(input.Ctx, epoch)
	mProceeds := tProceeds.Add(sProceeds)

	miningRewardWeight := input.TreasuryKeeper.GetRewardWeight(input.Ctx)
	require.Equal(t, sdkmath.LegacyNewDec(1111).MulInt64(core.MicroUnit).Add(miningRewardWeight.MulInt(amt)), mProceeds)
}

func TestLoadIndicatorByEpoch(t *testing.T) {
	input := CreateTestInput(t)

	TRArr := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDec(100),
		sdkmath.LegacyNewDec(200),
		sdkmath.LegacyNewDec(300),
		sdkmath.LegacyNewDec(400),
	}

	for epoch, TR := range TRArr {
		input.TreasuryKeeper.SetTR(input.Ctx, int64(epoch), TR)
	}

	SRArr := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDec(10),
		sdkmath.LegacyNewDec(20),
		sdkmath.LegacyNewDec(30),
		sdkmath.LegacyNewDec(40),
	}

	for epoch, SR := range SRArr {
		input.TreasuryKeeper.SetSR(input.Ctx, int64(epoch), SR)
	}

	TSLArr := []math.Int{
		sdkmath.NewInt(1000000),
		sdkmath.NewInt(2000000),
		sdkmath.NewInt(3000000),
		sdkmath.NewInt(4000000),
	}

	for epoch, TSL := range TSLArr {
		input.TreasuryKeeper.SetTSL(input.Ctx, int64(epoch), TSL)
	}

	for epoch := int64(0); epoch < 4; epoch++ {
		require.Equal(t, TRArr[epoch].QuoInt(TSLArr[epoch]), TRL(input.Ctx, epoch, input.TreasuryKeeper))
		require.Equal(t, SRArr[epoch], SR(input.Ctx, epoch, input.TreasuryKeeper))
		require.Equal(t, TRArr[epoch].Add(SRArr[epoch]), MR(input.Ctx, epoch, input.TreasuryKeeper))
	}

	// empty epoch load test
	require.Equal(t, sdkmath.LegacyZeroDec(), TRL(input.Ctx, 5, input.TreasuryKeeper))
	require.Equal(t, sdkmath.LegacyZeroDec(), SR(input.Ctx, 5, input.TreasuryKeeper))
	require.Equal(t, sdkmath.LegacyZeroDec(), MR(input.Ctx, 5, input.TreasuryKeeper))
}

func TestSumIndicator(t *testing.T) {
	input := CreateTestInput(t)

	SRArr := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDec(100),
		sdkmath.LegacyNewDec(200),
		sdkmath.LegacyNewDec(300),
		sdkmath.LegacyNewDec(400),
		sdkmath.LegacyNewDec(500),
		sdkmath.LegacyNewDec(600),
	}

	for epoch, SR := range SRArr {
		input.TreasuryKeeper.SetSR(input.Ctx, int64(epoch), SR)
	}

	// Case 1: at epoch 0 and summing over 0 epochs
	rval := input.TreasuryKeeper.sumIndicator(input.Ctx, 0, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 2: at epoch 0 and summing over negative epochs
	rval = input.TreasuryKeeper.sumIndicator(input.Ctx, -1, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 3: at epoch 3 and summing over 3, 4, 5 epochs; all should have the same rval
	input.Ctx = input.Ctx.WithBlockHeight(int64(core.BlocksPerWeek * 3))
	rval = input.TreasuryKeeper.sumIndicator(input.Ctx, 4, SR)
	rval2 := input.TreasuryKeeper.sumIndicator(input.Ctx, 5, SR)
	rval3 := input.TreasuryKeeper.sumIndicator(input.Ctx, 6, SR)
	require.Equal(t, sdkmath.LegacyNewDec(1000), rval)
	require.Equal(t, rval, rval2)
	require.Equal(t, rval2, rval3)

	// Case 4: at epoch 3 and summing over 0 epochs
	rval = input.TreasuryKeeper.sumIndicator(input.Ctx, 0, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 5. Sum up to 6
	input.Ctx = input.Ctx.WithBlockHeight(int64(core.BlocksPerWeek * 5))
	rval = input.TreasuryKeeper.sumIndicator(input.Ctx, 6, SR)
	require.Equal(t, sdkmath.LegacyNewDec(2100), rval)
}

func TestRollingAverageIndicator(t *testing.T) {
	input := CreateTestInput(t)
	SRArr := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDec(100),
		sdkmath.LegacyNewDec(200),
		sdkmath.LegacyNewDec(300),
		sdkmath.LegacyNewDec(400),
	}

	for epoch, SR := range SRArr {
		input.TreasuryKeeper.SetSR(input.Ctx, int64(epoch), SR)
	}

	// Case 1: at epoch 0 and averaging over 0 epochs
	rval := input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 0, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 2: at epoch 0 and averaging over negative epochs
	rval = input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, -1, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 3: at epoch 3 and averaging over 3, 4, 5 epochs; all should have the same rval
	input.Ctx = input.Ctx.WithBlockHeight(int64(core.BlocksPerWeek * 3))
	rval = input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 4, SR)
	rval2 := input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 5, SR)
	rval3 := input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 6, SR)
	require.Equal(t, sdkmath.LegacyNewDec(250), rval)
	require.Equal(t, rval, rval2)
	require.Equal(t, rval2, rval3)

	// Case 4: at epoch 3 and averaging over 0 epochs
	rval = input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 0, SR)
	require.Equal(t, sdkmath.LegacyZeroDec(), rval)

	// Case 5: at epoch 3 and averaging over 1 epoch
	rval = input.TreasuryKeeper.rollingAverageIndicator(input.Ctx, 1, SR)
	require.Equal(t, sdkmath.LegacyNewDec(400), rval)
}
