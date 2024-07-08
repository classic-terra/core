package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Market errors
var (
	ErrParsing      = errorsmod.Register(ModuleName, 1, "Parsing errors")
	ErrCoinNotFound = errorsmod.Register(ModuleName, 2, "Coin not found")
)
