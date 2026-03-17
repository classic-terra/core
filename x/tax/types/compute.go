package types

import (
	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TaxCapProvider abstracts access to per-denom tax caps.
// Both the custom ante TreasuryKeeper and the tax module's treasury keeper implement this method.
type TaxCapProvider interface {
	GetTaxCap(ctx sdk.Context, denom string) math.Int
}

// ComputeTaxes returns the taxes for the given principal according to the provided parameters.
// It unifies tax handling across ante and keeper:
// - Skips bond denom (staking token)
// - Skips IBC denoms (ibc/<64-hex>)
// - Applies burn tax rate
// - In simulate mode, enforces a minimum tax of 100 to allow split simulation
// - Applies per-denom tax caps
func ComputeTaxes(ctx sdk.Context, principal sdk.Coins, taxRate sdkmath.LegacyDec, simulate bool, caps TaxCapProvider) sdk.Coins {
	if taxRate.IsZero() {
		return sdk.Coins{}
	}

	taxes := sdk.Coins{}
	for _, coin := range principal {
		if coin.Denom == sdk.DefaultBondDenom {
			continue
		}
		if IsIBCDenom(coin.Denom) {
			continue
		}

		if coin.Amount.IsZero() {
			continue
		}

		taxDue := sdkmath.LegacyNewDecFromInt(coin.Amount).Mul(taxRate).TruncateInt()
		if simulate && taxDue.LT(sdkmath.NewInt(100)) {
			taxDue = sdkmath.NewInt(100)
		}

		if caps != nil {
			taxCap := caps.GetTaxCap(ctx, coin.Denom)
			if taxDue.GT(taxCap) {
				taxDue = taxCap
			}
		}

		if !taxDue.IsZero() {
			taxes = taxes.Add(sdk.NewCoin(coin.Denom, taxDue))
		}
	}

	return taxes
}
