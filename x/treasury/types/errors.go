package types

import (
	errorsmod "cosmossdk.io/errors"
)

// ErrNoSuchBurnTaxExemptionAddress is deprecated and no longer returned; it
// stays registered so error code 1 is not reused.
var ErrNoSuchBurnTaxExemptionAddress = errorsmod.Register(ModuleName, 1, "no such address in extemption list")
