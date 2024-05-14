package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Market errors
var (
	ErrRecursiveSwap    = errorsmod.Register(ModuleName, 2, "recursive swap")
	ErrNoEffectivePrice = errorsmod.Register(ModuleName, 3, "no price registered with oracle")
	ErrZeroSwapCoin     = errorsmod.Register(ModuleName, 4, "zero swap coin")
)
