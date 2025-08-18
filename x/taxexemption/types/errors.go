package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrNoSuchTaxExemptionZone    = sdkerrors.Register(ModuleName, 1, "no such zone in exemption list")
	ErrNoSuchTaxExemptionAddress = sdkerrors.Register(ModuleName, 2, "no such address in exemption list")
	ErrZoneNotExist              = sdkerrors.Register(ModuleName, 3, "zone not exist")
	ErrZoneLengthInvalid         = sdkerrors.Register(ModuleName, 4, "length of zone list and addresses by zone must be equal")
)
