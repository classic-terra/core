package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var ErrNoSuchWhitelist = sdkerrors.Register(ModuleName, 1, "no such whitelist address in substore")
