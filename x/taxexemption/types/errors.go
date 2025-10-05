package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrNoSuchTaxExemptionZone    = errors.Register(ModuleName, 1, "no such zone in exemption list")
	ErrNoSuchTaxExemptionAddress = errors.Register(ModuleName, 2, "no such address in exemption list")
	ErrZoneNotExist              = errors.Register(ModuleName, 3, "zone not exist")
	ErrZoneLengthInvalid         = errors.Register(ModuleName, 4, "length of zone list and addresses by zone must be equal")
)
