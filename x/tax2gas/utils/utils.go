package utils

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	oracleexported "github.com/classic-terra/core/v3/x/oracle/exported"
)

// GetTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it opens potential attack vectors
// where txs with multiple coins could not be prioritize as expected.
func GetTxPriority(fee sdk.Coins, gas int64) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		gasPrice := c.Amount.QuoRaw(gas)
		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}

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
