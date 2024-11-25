package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// expected TreasuryKeeper
type TreasuryKeeper interface {
	HasBurnTaxExemptionAddress(ctx sdk.Context, addresses ...string) bool
}
