package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	DefaultLimit           = uint64(5)
	DefaultMaxExecutionGas = uint64(5_000_000)
)

// ParamKeyTable returns the cron module parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams(limit uint64, maxExecutionGas ...uint64) Params {
	gasLimit := DefaultMaxExecutionGas
	if len(maxExecutionGas) > 0 {
		gasLimit = maxExecutionGas[0]
	}
	return Params{Limit: limit, MaxExecutionGas: gasLimit}
}

// DefaultParams returns the default cron module parameters.
func DefaultParams() Params {
	return NewParams(DefaultLimit)
}

// ParamSetPairs returns the cron module parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("Limit"), &p.Limit, validateLimit),
		paramtypes.NewParamSetPair([]byte("MaxExecutionGas"), &p.MaxExecutionGas, validateMaxExecutionGas),
	}
}

// Validate validates the cron module parameters.
func (p Params) Validate() error {
	if err := validateLimit(p.Limit); err != nil {
		return err
	}
	return validateMaxExecutionGas(p.MaxExecutionGas)
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

func validateMaxExecutionGas(i interface{}) error {
	g, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if g == 0 {
		return fmt.Errorf("max execution gas must be positive")
	}
	return nil
}

func validateAuthority(authority string) error {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return ErrInvalidAuthority
	}
	return nil
}
