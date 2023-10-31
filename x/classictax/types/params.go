package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	KeyBurnTaxRate     = []byte("BurnTaxRate")
	KeyGasFee          = []byte("GasFee")
	KeyTaxableMsgTypes = []byte("TaxableMsgTypes")
)

// Default classictax parameter values
var (
	// todo: correct default values
	DefaultBurnTax         = sdk.NewDecWithPrec(5, 3)     // 0.005 = 0.5%
	DefaultGasFee          = sdk.NewDecWithPrec(28536, 3) // 28.536
	DefaultTaxableMsgTypes = []string{"bank/MsgSend", "bank/MsgMultiSend", "wasm/MsgExecuteContract", "wasm/MsgInstantiateContract", "wasm/MsgStoreCode"}
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default classictax module parameters
func DefaultParams() Params {
	return Params{
		BurnTax:         DefaultBurnTax,
		GasFee:          DefaultGasFee,
		TaxableMsgTypes: DefaultTaxableMsgTypes,
	}
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

// String implements fmt.Stringer interface
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of market module's parameters.
func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyBurnTaxRate, &p.BurnTax, validateBurnTax),
		paramstypes.NewParamSetPair(KeyGasFee, &p.GasFee, validateGasFee),
		paramstypes.NewParamSetPair(KeyTaxableMsgTypes, &p.TaxableMsgTypes, validateTaxableMsgTypes),
	}
}

// Validate a set of params
func (p Params) Validate() error {
	if p.BurnTax.IsNegative() {
		return fmt.Errorf("burn tax must be positive or zero, is %s", p.BurnTax)
	}

	// todo: validate gas fees and taxable msg types

	return nil
}

func validateBurnTax(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("burn tax shall be 0 or positive: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("burn tax shall be less than 1.0: %s", v)
	}

	return nil
}

func validateGasFee(i interface{}) error {
	v, ok := i.(sdk.DecCoin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("gas fees must be positive or zero: %s", v)
	}

	return nil
}

func validateTaxableMsgTypes(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// validate message types
	for _, msgType := range v {
		if len(msgType) == 0 {
			return fmt.Errorf("taxable msg type must not be empty")
		}
	}

	return nil
}
