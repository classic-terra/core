package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyParamField = []byte("TODO: CHANGE ME")
)

// ParamTable for tax2gas module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams() Params {
	return Params{}
}

// DefaultParams are the default tax2gas module parameters.
func DefaultParams() Params {
	return Params{}
}

// Validate validates params.
func (p Params) Validate() error {
	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		// paramtypes.NewParamSetPair(KeyParamField, &p.Field, validateFn),
	}
}
