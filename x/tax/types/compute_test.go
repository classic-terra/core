package types

import (
	"testing"

	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// mockCaps implements TaxCapProvider for tests.
type mockCaps struct{ caps map[string]cosmosmath.Int }

func (m mockCaps) GetTaxCap(_ sdk.Context, denom string) cosmosmath.Int {
	if m.caps == nil {
		return cosmosmath.NewInt(0)
	}
	if v, ok := m.caps[denom]; ok {
		return v
	}
	return cosmosmath.NewInt(0)
}

func TestComputeTaxes_IBCDenomExcluded(t *testing.T) {
	ctx := sdk.Context{}
	// ibc denom hash (64 hex)
	ibcDenom := "ibc/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	principal := sdk.NewCoins(sdk.NewInt64Coin(ibcDenom, 1_000_000))
	taxes := ComputeTaxes(ctx, principal, sdk.NewDecWithPrec(1, 2), false, mockCaps{}) // 1%
	require.True(t, taxes.Empty(), "IBC denom must be excluded from tax")
}

func TestComputeTaxes_NativeDenomTaxWithCap(t *testing.T) {
	ctx := sdk.Context{}
	denom := "uluna"
	principal := sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000_000))
	// taxRate 2% => raw tax 20_000, but cap at 5_000 applies
	caps := mockCaps{caps: map[string]cosmosmath.Int{denom: cosmosmath.NewInt(5_000)}}
	taxes := ComputeTaxes(ctx, principal, sdk.NewDecWithPrec(2, 2), false, caps)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 5_000)), taxes)
}

func TestComputeTaxes_SimulateMinTax(t *testing.T) {
	ctx := sdk.Context{}
	denom := "uluna"
	principal := sdk.NewCoins(sdk.NewInt64Coin(denom, 1))
	// Very small rate -> would compute 0 tax, but simulate=true enforces min 100
	tinyRate := sdk.NewDecWithPrec(1, 10) // 0.0000000001
	caps := mockCaps{caps: map[string]cosmosmath.Int{denom: cosmosmath.NewInt(1_000_000)}}
	taxes := ComputeTaxes(ctx, principal, tinyRate, true, caps)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 100)), taxes)
}

func TestComputeTaxes_SkipBondDenom(t *testing.T) {
	ctx := sdk.Context{}
	bond := sdk.DefaultBondDenom // from SDK, typically "stake"
	principal := sdk.NewCoins(sdk.NewInt64Coin(bond, 1_000_000))
	taxes := ComputeTaxes(ctx, principal, sdk.NewDecWithPrec(1, 2), false, mockCaps{}) // 1%
	require.True(t, taxes.Empty(), "bond denom must be skipped")
}

// computeTaxOldBrokenWay simulates the old keeper.ComputeTax logic that had the regression
func computeTaxOldBrokenWay(amount sdk.Coins, burnTaxRate sdk.Dec) sdk.Coins {
	taxes := sdk.Coins{}
	for _, coin := range amount {
		taxAmount := sdk.NewDecFromInt(coin.Amount).Mul(burnTaxRate).TruncateInt()
		if taxAmount.IsPositive() {
			taxes = taxes.Add(sdk.NewCoin(coin.Denom, taxAmount))
		}
	}
	return taxes
}

// TestComputeTaxes_ContractRegressionFix verifies that IBC tokens are not taxed
// in contract reverse charge scenarios (the original regression case)
func TestComputeTaxes_ContractRegressionFix(t *testing.T) {
	ctx := sdk.Context{}
	ibcDenom := "ibc/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	taxRate := sdk.NewDecWithPrec(1, 2) // 1%
	
	// Test the scenario that was broken: contract interaction with IBC tokens
	contractFunds := sdk.NewCoins(sdk.NewInt64Coin(ibcDenom, 1_000_000))
	
	// Demonstrate the old broken behavior would have taxed IBC tokens
	oldBrokenTaxes := computeTaxOldBrokenWay(contractFunds, taxRate)
	require.False(t, oldBrokenTaxes.Empty(), "old logic incorrectly taxed IBC tokens - this confirms the regression existed")
	
	// Verify the fix: new unified logic excludes IBC tokens
	newFixedTaxes := ComputeTaxes(ctx, contractFunds, taxRate, false, mockCaps{})
	require.True(t, newFixedTaxes.Empty(), "IBC tokens must not be taxed in contract interactions")
	
	// Verify regular tokens are still taxed correctly
	regularFunds := sdk.NewCoins(sdk.NewInt64Coin("uluna", 1_000_000))
	caps := mockCaps{caps: map[string]cosmosmath.Int{"uluna": cosmosmath.NewInt(1_000_000)}}
	regularTaxes := ComputeTaxes(ctx, regularFunds, taxRate, false, caps)
	expectedTax := sdk.NewCoins(sdk.NewInt64Coin("uluna", 10_000)) // 1% of 1M
	require.Equal(t, expectedTax, regularTaxes, "regular tokens should be taxed")
}
