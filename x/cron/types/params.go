package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	DefaultLimit = uint64(5)
)

// ParamKeyTable returns the cron module parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams(limit uint64) Params {
	return Params{Limit: limit}
}

// DefaultParams returns the default cron module parameters.
func DefaultParams() Params {
	return NewParams(DefaultLimit)
}

// ParamSetPairs returns the cron module parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("Limit"), &p.Limit, validateLimit),
	}
}

// Validate validates the cron module parameters.
func (p Params) Validate() error {
	return validateLimit(p.Limit)
}

func validateLimit(i interface{}) error {
	l, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if l == 0 {
		return ErrInvalidLimit
	}
	return nil
}

func validateAuthority(authority string) error {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return ErrInvalidAuthority
	}
	return nil
}
