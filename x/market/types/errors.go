package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Market errors
var (
	ErrRecursiveSwap         = errorsmod.Register(ModuleName, 2, "recursive swap")
	ErrNoEffectivePrice      = errorsmod.Register(ModuleName, 3, "no price registered with oracle")
	ErrZeroSwapCoin          = errorsmod.Register(ModuleName, 4, "zero swap coin")
	ErrInvalidSwapPair       = errorsmod.Register(ModuleName, 5, "invalid swap pair; not allowed")
	ErrInsufficientLiquidity = errorsmod.Register(ModuleName, 6, "insufficient pool liquidity")
	ErrOraclePriceStale      = errorsmod.Register(ModuleName, 7, "oracle price too old; swap denied")
	ErrTWAPDeviation         = errorsmod.Register(ModuleName, 8, "price deviates too much from TWAP")
	ErrDailyCapExceeded      = errorsmod.Register(ModuleName, 9, "daily swap cap exceeded")
)
