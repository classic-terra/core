package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	core "github.com/classic-terra/core/v2/types"
)

// Parameter keys
var (
	// Terra liquidity pool(usdr unit) made available per ${PoolRecoveryPeriod} (usdr unit)
	KeyBasePool = []byte("BasePool")
	// The period required to recover BasePool
	KeyPoolRecoveryPeriod = []byte("PoolRecoveryPeriod")
	// Min spread
	KeyMinStabilitySpread            = []byte("MinStabilitySpread")
	KeyMaxSupplyCoin                 = []byte("MaxSupplyCoin")
	KeyPercentageSupplyMaxDescending = []byte("PercentageSupplyMaxDescending")
)

// Default parameter values
var (
	DefaultBasePool           = sdk.NewDec(1000000 * core.MicroUnit) // 1000,000sdr = 1000,000,000,000usdr
	DefaultPoolRecoveryPeriod = core.BlocksPerDay                    // 14,400
	DefaultMinStabilitySpread = sdk.NewDecWithPrec(2, 2)
	// ATTENTION: The list of modes must be in alphabetical order, otherwise an error occurs in validateMaxSupplyCoin => !v.IsValid()
	DefaultMaxSupplyCoin = sdk.Coins{
		{Denom: "uaud", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ucad", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uchf", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ucny", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "udkk", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ueur", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ugbp", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uhkd", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uidr", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uinr", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ujpy", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "ukrw", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uluna", Amount: sdk.NewInt(6842077281678221687)},
		{Denom: "umnt", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "unok", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uphp", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "usdr", Amount: sdk.NewInt(700380430867725)},
		{Denom: "usek", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "usgd", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uthb", Amount: sdk.NewInt(500000000000000000)},
		{Denom: "uusd", Amount: sdk.NewInt(9797700474387101)},
		{Denom: "uarb", Amount: sdk.NewInt(50000000000000)},
	}
	DefaultPercentageSupplyMaxDescending = sdk.NewDecWithPrec(30, 2) // 30%
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default market module parameters
func DefaultParams() Params {
	return Params{
		BasePool:                      DefaultBasePool,
		PoolRecoveryPeriod:            DefaultPoolRecoveryPeriod,
		MinStabilitySpread:            DefaultMinStabilitySpread,
		MaxSupplyCoin:                 DefaultMaxSupplyCoin,
		PercentageSupplyMaxDescending: DefaultPercentageSupplyMaxDescending,
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
		paramstypes.NewParamSetPair(KeyBasePool, &p.BasePool, validateBasePool),
		paramstypes.NewParamSetPair(KeyPoolRecoveryPeriod, &p.PoolRecoveryPeriod, validatePoolRecoveryPeriod),
		paramstypes.NewParamSetPair(KeyMinStabilitySpread, &p.MinStabilitySpread, validateMinStabilitySpread),
		paramstypes.NewParamSetPair(KeyMaxSupplyCoin, &p.MaxSupplyCoin, validateMaxSupplyCoin),
		paramstypes.NewParamSetPair(KeyPercentageSupplyMaxDescending, &p.PercentageSupplyMaxDescending, validatePercentageSupplyMaxDescending),
	}
}

// Validate a set of params
func (p Params) Validate() error {
	if p.BasePool.IsNegative() {
		return fmt.Errorf("mint base pool should be positive or zero, is %s", p.BasePool)
	}
	if p.PoolRecoveryPeriod == 0 {
		return fmt.Errorf("pool recovery period should be positive, is %d", p.PoolRecoveryPeriod)
	}
	if p.MinStabilitySpread.IsNegative() || p.MinStabilitySpread.GT(sdk.OneDec()) {
		return fmt.Errorf("market minimum stability spead should be a value between [0,1], is %s", p.MinStabilitySpread)
	}
	if len(p.MaxSupplyCoin) == 0 {
		return fmt.Errorf("max supplay cannot be empty %s", p.MaxSupplyCoin)
	}
	if p.PercentageSupplyMaxDescending.IsNegative() {
		return fmt.Errorf("mint base pool should be positive or zero, is %s", p.PercentageSupplyMaxDescending)
	}
	return nil
}

func validateBasePool(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("mint base pool must be positive or zero: %s", v)
	}

	return nil
}

func validatePoolRecoveryPeriod(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("pool recovery period must be positive: %d", v)
	}

	return nil
}

func validateMinStabilitySpread(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("min spread must be positive or zero: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("min spread is too large: %s", v)
	}

	return nil
}

func validateMaxSupplyCoin(i interface{}) error {
	// v, ok := i.(sdk.Coins)
	// if !ok {
	// 	return fmt.Errorf("invalid parameter type: %T", i)
	// }
	// if !v.IsValid() {
	// 	return fmt.Errorf("invalid max supply: %s", v)
	// }

	return nil
}

func validatePercentageSupplyMaxDescending(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("mint Percentage Supply Max Descending must be positive or zero: %s", v)
	}

	return nil
}
