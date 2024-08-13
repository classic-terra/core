package utils

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	oracleexported "github.com/classic-terra/core/v3/x/oracle/exported"
)

func IsOracleTx(msgs []sdk.Msg) bool {
	for _, msg := range msgs {
		switch msg.(type) {
		case *oracleexported.MsgAggregateExchangeRatePrevote:
			continue
		case *oracleexported.MsgAggregateExchangeRateVote:
			continue
		default:
			return false
		}
	}

	return true
}

// Find returns true and Dec amount if the denom exists in gasPrices. Otherwise it returns false
// and a zero dec. Uses binary search.
// CONTRACT: gasPrices must be valid (sorted).
func GetGasPriceByDenom(gasPrices sdk.DecCoins, denom string) (bool, sdk.Dec) {
	switch len(gasPrices) {
	case 0:
		return false, sdk.ZeroDec()

	case 1:
		gasPrice := gasPrices[0]
		if gasPrice.Denom == denom {
			return true, gasPrice.Amount
		}
		return false, sdk.ZeroDec()

	default:
		midIdx := len(gasPrices) / 2 // 2:1, 3:1, 4:2
		gasPrice := gasPrices[midIdx]
		switch {
		case denom < gasPrice.Denom:
			return GetGasPriceByDenom(gasPrices[:midIdx], denom)
		case denom == gasPrice.Denom:
			return true, gasPrice.Amount
		default:
			return GetGasPriceByDenom(gasPrices[midIdx+1:], denom)
		}
	}
}

func CalculateTaxesAndPayableFee(gasPrices sdk.DecCoins, feeCoins sdk.Coins, taxGas sdkmath.Int, totalGasRemaining sdkmath.Int) (taxes, payableFees sdk.Coins, gasRemaining sdkmath.Int) {
	taxGasRemaining := taxGas
	taxes = sdk.NewCoins()
	payableFees = sdk.NewCoins()
	gasRemaining = totalGasRemaining
	for _, feeCoin := range feeCoins {
		found, gasPrice := GetGasPriceByDenom(gasPrices, feeCoin.Denom)
		if !found {
			continue
		}
		taxFeeRequired := sdk.NewCoin(feeCoin.Denom, gasPrice.MulInt(taxGasRemaining).Ceil().RoundInt())
		totalFeeRequired := sdk.NewCoin(feeCoin.Denom, gasPrice.MulInt(gasRemaining).Ceil().RoundInt())

		switch {
		case taxGasRemaining.IsPositive():
			switch {
			case feeCoin.IsGTE(totalFeeRequired):
				taxes = taxes.Add(taxFeeRequired)
				payableFees = payableFees.Add(totalFeeRequired)
				gasRemaining = sdkmath.ZeroInt()
				return taxes, payableFees, gasRemaining
			case feeCoin.IsGTE(taxFeeRequired):
				taxes = taxes.Add(taxFeeRequired)
				taxGasRemaining = sdkmath.ZeroInt()
				payableFees = payableFees.Add(feeCoin)
				totalFeeRemaining := sdk.NewDecCoinFromCoin(totalFeeRequired.Sub(feeCoin))
				gasRemaining = totalFeeRemaining.Amount.Quo(gasPrice).Ceil().RoundInt()
			default:
				taxes = taxes.Add(feeCoin)
				payableFees = payableFees.Add(feeCoin)
				taxFeeRemaining := sdk.NewDecCoinFromCoin(taxFeeRequired.Sub(feeCoin))
				taxGasRemaining = taxFeeRemaining.Amount.Quo(gasPrice).Ceil().RoundInt()
				gasRemaining = gasRemaining.Sub(taxGas.Sub(taxGasRemaining))
			}
		case gasRemaining.IsPositive():
			if feeCoin.IsGTE(totalFeeRequired) {
				payableFees = payableFees.Add(totalFeeRequired)
				gasRemaining = sdkmath.ZeroInt()
				return taxes, payableFees, gasRemaining
			}
			payableFees = payableFees.Add(feeCoin)
			totalFeeRemaining := sdk.NewDecCoinFromCoin(totalFeeRequired.Sub(feeCoin))
			gasRemaining = totalFeeRemaining.Amount.Quo(gasPrice).Ceil().RoundInt()
		default:
			return taxes, payableFees, gasRemaining
		}
	}
	return taxes, payableFees, gasRemaining
}
