package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	return Params{
		GasPrices: sdk.NewDecCoins(
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
		),
	}
}

// Validate validates params.
func (p Params) Validate() error {
	if len(p.GasPrices) == 0 {
		return fmt.Errorf("must provide at least 1 gas prices")
	}
	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		// paramtypes.NewParamSetPair(KeyParamField, &p.Field, validateFn),
	}
}
