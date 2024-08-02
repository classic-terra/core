package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Tax2Gas errors
var (
	ErrParsing       = errorsmod.Register(ModuleName, 1, "Parsing errors")
	ErrCoinNotFound  = errorsmod.Register(ModuleName, 2, "Coin not found")
	ErrDenomNotFound = errorsmod.Register(ModuleName, 3, "Denom not found")
)
