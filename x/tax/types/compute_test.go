package types

import (
	"testing"

	cosmosmath "cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
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
	taxes := ComputeTaxes(ctx, principal, sdkmath.LegacyNewDecWithPrec(1, 2), false, mockCaps{}) // 1%
	require.True(t, taxes.Empty(), "IBC denom must be excluded from tax")
}

func TestComputeTaxes_NativeDenomTaxWithCap(t *testing.T) {
	ctx := sdk.Context{}
	denom := "uluna"
	principal := sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000_000))
	// taxRate 2% => raw tax 20_000, but cap at 5_000 applies
	caps := mockCaps{caps: map[string]cosmosmath.Int{denom: cosmosmath.NewInt(5_000)}}
	taxes := ComputeTaxes(ctx, principal, sdkmath.LegacyNewDecWithPrec(2, 2), false, caps)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 5_000)), taxes)
}

func TestComputeTaxes_SimulateMinTax(t *testing.T) {
	ctx := sdk.Context{}
	denom := "uluna"
	principal := sdk.NewCoins(sdk.NewInt64Coin(denom, 1))
	// Very small rate -> would compute 0 tax, but simulate=true enforces min 100
	tinyRate := sdkmath.LegacyNewDecWithPrec(1, 10) // 0.0000000001
	caps := mockCaps{caps: map[string]cosmosmath.Int{denom: cosmosmath.NewInt(1_000_000)}}
	taxes := ComputeTaxes(ctx, principal, tinyRate, true, caps)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 100)), taxes)
}

func TestComputeTaxes_SkipBondDenom(t *testing.T) {
	ctx := sdk.Context{}
	bond := sdk.DefaultBondDenom // from SDK, typically "stake"
	principal := sdk.NewCoins(sdk.NewInt64Coin(bond, 1_000_000))
	taxes := ComputeTaxes(ctx, principal, sdkmath.LegacyNewDecWithPrec(1, 2), false, mockCaps{}) // 1%
	require.True(t, taxes.Empty(), "bond denom must be skipped")
}

// TestCommunityTaxAdjustment covers the adjusted community-tax rate used by the
// tax splits. The divisor is communityTax*(1-oracleSplitRate), which is zero
// when communityTax == 1.0 and oracleSplitRate == 0 — both governance-settable
// parameters — and previously caused a decimal division-by-zero panic inside
// every taxable transaction.
func TestCommunityTaxAdjustment(t *testing.T) {
	one := sdkmath.LegacyOneDec()
	zero := sdkmath.LegacyZeroDec()

	t.Run("zero community tax returns zero", func(t *testing.T) {
		require.True(t, CommunityTaxAdjustment(zero, sdkmath.LegacyNewDecWithPrec(2, 1)).IsZero())
	})

	t.Run("zero oracle split yields zero adjustment (original semantics)", func(t *testing.T) {
		communityTax := sdkmath.LegacyNewDecWithPrec(2, 2) // 0.02
		// Original expression: ct*(osr/denominator) = ct*(0/(1-ct)) = 0.
		require.True(t, CommunityTaxAdjustment(communityTax, zero).IsZero())
	})

	t.Run("well-known values match hand-computed result", func(t *testing.T) {
		communityTax := sdkmath.LegacyNewDecWithPrec(2, 2) // 0.02
		oracleSplit := sdkmath.LegacyNewDecWithPrec(5, 1)  // 0.5
		// 0.02*0.5 / (0.02*0.5 + 1 - 0.02) = 0.01 / 0.99
		want := sdkmath.LegacyNewDecWithPrec(1, 2).Quo(sdkmath.LegacyNewDecWithPrec(99, 2))
		got := CommunityTaxAdjustment(communityTax, oracleSplit)
		require.True(t, got.Equal(want), "got %s want %s", got, want)
	})

	t.Run("community tax of one with zero oracle split does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			got := CommunityTaxAdjustment(one, zero)
			require.True(t, got.Equal(one))
		})
	})

	t.Run("community tax of one with positive oracle split does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			// numerator = 1*0.5 = 0.5, denominator = 0.5 + 1 - 1 = 0.5 → 1.0
			got := CommunityTaxAdjustment(one, sdkmath.LegacyNewDecWithPrec(5, 1))
			require.True(t, got.Equal(one))
		})
	})
}
