package keeper

import (
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/treasury/types"
)

func TestRewardWeight(t *testing.T) {
	input := CreateTestInput(t)

	// See that we can get and set reward weights
	for i := int64(0); i < 10; i++ {
		input.TreasuryKeeper.SetRewardWeight(input.Ctx, sdkmath.LegacyNewDecWithPrec(i, 2))
		require.Equal(t, sdkmath.LegacyNewDecWithPrec(i, 2), input.TreasuryKeeper.GetRewardWeight(input.Ctx))
	}
}

func TestTaxRate(t *testing.T) {
	input := CreateTestInput(t)

	// See that we can get and set tax rate
	for i := int64(0); i < 10; i++ {
		input.TreasuryKeeper.SetTaxRate(input.Ctx, sdkmath.LegacyNewDecWithPrec(i, 2))
		require.Equal(t, sdkmath.LegacyNewDecWithPrec(i, 2), input.TreasuryKeeper.GetTaxRate(input.Ctx))
	}
}

func TestTaxCap(t *testing.T) {
	input := CreateTestInput(t)

	for i := int64(0); i < 10; i++ {
		input.TreasuryKeeper.SetTaxCap(input.Ctx, core.MicroCNYDenom, sdkmath.NewInt(i))
		require.Equal(t, sdkmath.NewInt(i), input.TreasuryKeeper.GetTaxCap(input.Ctx, core.MicroCNYDenom))
	}
}

func TestIterateTaxCap(t *testing.T) {
	input := CreateTestInput(t)

	cnyCap := sdkmath.NewInt(123)
	usdCap := sdkmath.NewInt(13)
	krwCap := sdkmath.NewInt(1300)
	input.TreasuryKeeper.SetTaxCap(input.Ctx, core.MicroCNYDenom, cnyCap)
	input.TreasuryKeeper.SetTaxCap(input.Ctx, core.MicroUSDDenom, usdCap)
	input.TreasuryKeeper.SetTaxCap(input.Ctx, core.MicroKRWDenom, krwCap)

	input.TreasuryKeeper.IterateTaxCap(input.Ctx, func(denom string, taxCap math.Int) bool {
		switch denom {
		case core.MicroCNYDenom:
			require.Equal(t, cnyCap, taxCap)
		case core.MicroUSDDenom:
			require.Equal(t, usdCap, taxCap)
		case core.MicroKRWDenom:
			require.Equal(t, krwCap, taxCap)
		}

		return false
	})
}

func TestTaxProceeds(t *testing.T) {
	input := CreateTestInput(t)

	for i := int64(0); i < 10; i++ {
		proceeds := sdk.NewCoins(sdk.NewCoin(core.MicroSDRDenom, sdkmath.NewInt(100+i)))
		input.TreasuryKeeper.RecordEpochTaxProceeds(input.Ctx, proceeds)
		input.TreasuryKeeper.RecordEpochTaxProceeds(input.Ctx, proceeds)
		input.TreasuryKeeper.RecordEpochTaxProceeds(input.Ctx, proceeds)

		require.Equal(t, proceeds.Add(proceeds...).Add(proceeds...), input.TreasuryKeeper.PeekEpochTaxProceeds(input.Ctx))

		input.TreasuryKeeper.SetEpochTaxProceeds(input.Ctx, sdk.Coins{})
		require.True(t, input.TreasuryKeeper.PeekEpochTaxProceeds(input.Ctx).IsZero())
	}
}

func TestMicroLunaIssuance(t *testing.T) {
	input := CreateTestInput(t)

	initialSupply := input.BankKeeper.GetSupply(input.Ctx, core.MicroLunaDenom)
	// See that we can get and set luna issuance
	blocksPerEpoch := core.BlocksPerWeek
	for i := int64(0); i < 10; i++ {
		input.Ctx = input.Ctx.WithBlockHeight(i * int64(blocksPerEpoch))

		input.TreasuryKeeper.RecordEpochInitialIssuance(input.Ctx)
		require.Equal(t, initialSupply.Amount.Add(sdkmath.NewInt(i)), input.TreasuryKeeper.GetEpochInitialIssuance(input.Ctx).AmountOf(core.MicroLunaDenom))

		input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, sdkmath.OneInt())))
	}
}

func TestPeekEpochSeigniorage(t *testing.T) {
	input := CreateTestInput(t)

	for i := int64(0); i < 10; i++ {
		input.Ctx = input.Ctx.WithBlockHeight(i * int64(core.BlocksPerWeek))
		faucetBalance := input.BankKeeper.GetBalance(input.Ctx, input.AccountKeeper.GetModuleAddress(faucetAccountName), core.MicroLunaDenom)

		input.TreasuryKeeper.RecordEpochInitialIssuance(input.Ctx)

		issueAmount := sdkmath.NewInt(rand.Int63()%1000000 + 1)
		err := input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, issueAmount)))
		require.NoError(t, err)

		burnAmount := sdkmath.NewInt(rand.Int63()%(faucetBalance.Amount.Int64()+issueAmount.Int64()) + 1)
		err = input.BankKeeper.BurnCoins(input.Ctx, faucetAccountName, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, burnAmount)))
		require.NoError(t, err)

		targetSeigniorage := burnAmount.Sub(issueAmount)
		if targetSeigniorage.IsNegative() {
			targetSeigniorage = sdkmath.ZeroInt()
		}

		require.Equal(t, targetSeigniorage, input.TreasuryKeeper.PeekEpochSeigniorage(input.Ctx))
	}
}

func TestIndicatorGetterSetter(t *testing.T) {
	input := CreateTestInput(t)

	for e := int64(0); e < 10; e++ {
		randomVal := sdkmath.LegacyNewDec(rand.Int63() + 1)
		input.TreasuryKeeper.SetTR(input.Ctx, e, randomVal)
		require.Equal(t, randomVal, input.TreasuryKeeper.GetTR(input.Ctx, e))
		input.TreasuryKeeper.SetSR(input.Ctx, e, randomVal)
		require.Equal(t, randomVal, input.TreasuryKeeper.GetSR(input.Ctx, e))
		input.TreasuryKeeper.SetTSL(input.Ctx, e, randomVal.TruncateInt())
		require.Equal(t, randomVal.TruncateInt(), input.TreasuryKeeper.GetTSL(input.Ctx, e))
	}

	input.TreasuryKeeper.ClearTRs(input.Ctx)
	input.TreasuryKeeper.ClearSRs(input.Ctx)
	input.TreasuryKeeper.ClearTSLs(input.Ctx)

	for e := int64(0); e < 10; e++ {
		require.Equal(t, sdkmath.LegacyZeroDec(), input.TreasuryKeeper.GetTR(input.Ctx, e))
		require.Equal(t, sdkmath.LegacyZeroDec(), input.TreasuryKeeper.GetSR(input.Ctx, e))
		require.Equal(t, sdkmath.ZeroInt(), input.TreasuryKeeper.GetTSL(input.Ctx, e))
	}
}

func TestParams(t *testing.T) {
	input := CreateTestInput(t)

	defaultParams := types.DefaultParams()
	input.TreasuryKeeper.SetParams(input.Ctx, defaultParams)

	retrievedParams := input.TreasuryKeeper.GetParams(input.Ctx)
	require.Equal(t, defaultParams, retrievedParams)
}
