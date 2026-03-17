package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGasPrices is set at runtime to the staking token with zero amount i.e. "0uatom"
// see DefaultZeroGlobalFee method in gaia/x/globalfee/ante/fee.go.
var DefaultGasPrices = sdk.NewDecCoins(
	sdk.NewDecCoinFromDec("uluna", sdkmath.LegacyNewDecWithPrec(28325, 3)),
	sdk.NewDecCoinFromDec("usdr", sdkmath.LegacyNewDecWithPrec(52469, 5)),
	sdk.NewDecCoinFromDec("uusd", sdkmath.LegacyNewDecWithPrec(75, 2)),
	sdk.NewDecCoinFromDec("ukrw", sdkmath.LegacyNewDecWithPrec(850, 0)),
	sdk.NewDecCoinFromDec("umnt", sdkmath.LegacyNewDecWithPrec(2142855, 3)),
	sdk.NewDecCoinFromDec("ueur", sdkmath.LegacyNewDecWithPrec(625, 3)),
	sdk.NewDecCoinFromDec("ucny", sdkmath.LegacyNewDecWithPrec(49, 1)),
	sdk.NewDecCoinFromDec("ujpy", sdkmath.LegacyNewDecWithPrec(8185, 2)),
	sdk.NewDecCoinFromDec("ugbp", sdkmath.LegacyNewDecWithPrec(55, 2)),
	sdk.NewDecCoinFromDec("uinr", sdkmath.LegacyNewDecWithPrec(544, 1)),
	sdk.NewDecCoinFromDec("ucad", sdkmath.LegacyNewDecWithPrec(95, 2)),
	sdk.NewDecCoinFromDec("uchf", sdkmath.LegacyNewDecWithPrec(7, 1)),
	sdk.NewDecCoinFromDec("uaud", sdkmath.LegacyNewDecWithPrec(95, 2)),
	sdk.NewDecCoinFromDec("usgd", sdkmath.LegacyNewDec(1)),
	sdk.NewDecCoinFromDec("uthb", sdkmath.LegacyNewDecWithPrec(231, 1)),
	sdk.NewDecCoinFromDec("usek", sdkmath.LegacyNewDecWithPrec(625, 2)),
	sdk.NewDecCoinFromDec("unok", sdkmath.LegacyNewDecWithPrec(625, 2)),
	sdk.NewDecCoinFromDec("udkk", sdkmath.LegacyNewDecWithPrec(45, 1)),
	sdk.NewDecCoinFromDec("uidr", sdkmath.LegacyNewDecWithPrec(10900, 0)),
	sdk.NewDecCoinFromDec("uphp", sdkmath.LegacyNewDecWithPrec(38, 0)),
	sdk.NewDecCoinFromDec("uhkd", sdkmath.LegacyNewDecWithPrec(585, 2)),
	sdk.NewDecCoinFromDec("umyr", sdkmath.LegacyNewDecWithPrec(3, 0)),
	sdk.NewDecCoinFromDec("utwd", sdkmath.LegacyNewDecWithPrec(20, 0)),
)

func NewParams() Params {
	return Params{}
}

// DefaultParams are the default tax2gas module parameters.
func DefaultParams() Params {
	return Params{
		GasPrices:   DefaultGasPrices,
		BurnTaxRate: sdkmath.LegacyNewDecWithPrec(5, 3),
	}
}

// Validate validates params.
func (p Params) Validate() error {
	/*if len(p.GasPrices) == 0 {
		return fmt.Errorf("must provide at least 1 gas prices")
	}*/
	// gas prices can be empty in case of 0 gas price

	return nil
}
