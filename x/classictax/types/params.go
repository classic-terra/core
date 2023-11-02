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
	KeyGasPrices       = []byte("GasPrices")
	KeyTaxableMsgTypes = []byte("TaxableMsgTypes")
)

// Default classictax parameter values
var (
	// todo: correct default values
	DefaultBurnTax   = sdk.NewDecWithPrec(5, 3) // 0.005 = 0.5%
	DefaultGasPrices = []sdk.DecCoin{
		sdk.NewDecCoinFromDec("uluna", sdk.NewDecWithPrec(28325, 3)),
		sdk.NewDecCoinFromDec("usdr", sdk.NewDecWithPrec(52469, 5)),
		sdk.NewDecCoinFromDec("uusd", sdk.NewDecWithPrec(75, 2)),
		sdk.NewDecCoinFromDec("ukrw", sdk.NewDecWithPrec(850, 0)),
		sdk.NewDecCoinFromDec("umnt", sdk.NewDecWithPrec(2142855, 3)),
		sdk.NewDecCoinFromDec("ueur", sdk.NewDecWithPrec(625, 3)),
		sdk.NewDecCoinFromDec("ucny", sdk.NewDecWithPrec(49, 1)),
		sdk.NewDecCoinFromDec("ujpy", sdk.NewDecWithPrec(8185, 2)),
		sdk.NewDecCoinFromDec("ugbp", sdk.NewDecWithPrec(55, 2)),
		sdk.NewDecCoinFromDec("uinr", sdk.NewDecWithPrec(544, 1)),
		sdk.NewDecCoinFromDec("ucad", sdk.NewDecWithPrec(95, 2)),
		sdk.NewDecCoinFromDec("uchf", sdk.NewDecWithPrec(7, 1)),
		sdk.NewDecCoinFromDec("uaud", sdk.NewDecWithPrec(95, 2)),
		sdk.NewDecCoinFromDec("usgd", sdk.NewDec(1)),
		sdk.NewDecCoinFromDec("uthb", sdk.NewDecWithPrec(231, 1)),
		sdk.NewDecCoinFromDec("usek", sdk.NewDecWithPrec(625, 2)),
		sdk.NewDecCoinFromDec("unok", sdk.NewDecWithPrec(625, 2)),
		sdk.NewDecCoinFromDec("udkk", sdk.NewDecWithPrec(45, 1)),
		sdk.NewDecCoinFromDec("uidr", sdk.NewDecWithPrec(10900, 0)),
		sdk.NewDecCoinFromDec("uphp", sdk.NewDecWithPrec(38, 0)),
		sdk.NewDecCoinFromDec("uhkd", sdk.NewDecWithPrec(585, 2)),
		sdk.NewDecCoinFromDec("umyr", sdk.NewDecWithPrec(3, 0)),
		sdk.NewDecCoinFromDec("utwd", sdk.NewDecWithPrec(20, 0)),
	}
	DefaultTaxableMsgTypes = []string{"types.MsgSend", "types.MsgMultiSend", "types.MsgExecuteContract", "types.MsgInstantiateContract", "types.MsgStoreCode"}
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default classictax module parameters
func DefaultParams() Params {
	return Params{
		BurnTax:         DefaultBurnTax,
		GasPrices:       DefaultGasPrices,
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
		paramstypes.NewParamSetPair(KeyGasPrices, &p.GasPrices, validateGasPrices),
		paramstypes.NewParamSetPair(KeyTaxableMsgTypes, &p.TaxableMsgTypes, validateTaxableMsgTypes),
	}
}

// Validate a set of params
func (p Params) Validate() error {
	if p.BurnTax.IsNegative() {
		return fmt.Errorf("burn tax must be positive or zero, is %s", p.BurnTax)
	}

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

func validateGasPrices(i interface{}) error {
	v, ok := i.([]sdk.DecCoin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, coin := range v {
		if coin.IsNegative() {
			return fmt.Errorf("gas prices must be positive or zero: %s", v)
		}
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
