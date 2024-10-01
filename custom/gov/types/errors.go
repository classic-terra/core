package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// x/gov module sentinel errors
var (
	ErrQueryExchangeRateUusdFail = sdkerrors.Register(govtypes.ModuleName, 17, "Get exchange rate lunc-uusd from oracle failed")
)
