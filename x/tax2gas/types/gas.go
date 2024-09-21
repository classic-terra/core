package types

import (
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ErrorNegativeGasConsumed struct {
	Descriptor string
}

type Tax2GasMeter struct {
	limit         sdk.Gas
	consumed      sdk.Gas
	taxConsumed   sdkmath.Int
	limitEnforced bool
}

// NewGasMeter returns a reference to a new Tax2GasMeter.
func NewTax2GasMeter(limit sdk.Gas) sdk.GasMeter {
	return &Tax2GasMeter{
		limit:         limit,
		consumed:      0,
		taxConsumed:   sdkmath.ZeroInt(),
		limitEnforced: true,
	}
}

// GasConsumed returns the gas consumed from the GasMeter.
func (g *Tax2GasMeter) GasConsumed() sdk.Gas {
	return g.consumed
}

// TaxConsumed returns the tax consumed from the GasMeter.
func (g *Tax2GasMeter) TaxConsumed() sdkmath.Int {
	return g.taxConsumed
}

// GasRemaining returns the gas left in the GasMeter.
func (g *Tax2GasMeter) GasRemaining() sdk.Gas {
	if g.IsPastLimit() {
		return 0
	}
	return g.limit - g.consumed
}

// Limit returns the gas limit of the GasMeter.
func (g *Tax2GasMeter) Limit() sdk.Gas {
	return g.limit
}

// GasConsumedToLimit returns the gas limit if gas consumed is past the limit,
// otherwise it returns the consumed gas.
//
// NOTE: This behavior is only called when recovering from panic when
// BlockGasMeter consumes gas past the limit.
func (g *Tax2GasMeter) GasConsumedToLimit() sdk.Gas {
	if g.limitEnforced {
		if g.IsPastLimit() {
			return g.limit
		}
		return g.consumed
	}

	// When limit enforcement is disabled, return the limit to prevent panics in consumeBlockGas
	return g.limit
}

// addUint64Overflow performs the addition operation on two uint64 integers and
// returns a boolean on whether or not the result overflows.
func addUint64Overflow(a, b uint64) (uint64, bool) {
	if math.MaxUint64-a < b {
		return 0, true
	}

	return a + b, false
}

// ConsumeGas adds the given amount of gas to the gas consumed and panics if it overflows the limit or out of gas.
func (g *Tax2GasMeter) ConsumeGas(amount sdk.Gas, descriptor string) {
	var overflow bool
	g.consumed, overflow = addUint64Overflow(g.consumed, amount)
	if overflow {
		g.consumed = math.MaxUint64
		panic(sdk.ErrorGasOverflow{Descriptor: descriptor})
	}

	if g.consumed > g.limit {
		panic(sdk.ErrorOutOfGas{Descriptor: descriptor})
	}
}

func (g *Tax2GasMeter) ConsumeTax(amount sdkmath.Int, descriptor string) {
	if amount.IsNegative() {
		panic(ErrorNegativeGasConsumed{Descriptor: descriptor})
	}

	g.taxConsumed = g.taxConsumed.Add(amount)
}

// RefundGas will deduct the given amount from the gas consumed. If the amount is greater than the
// gas consumed, the function will panic.
//
// Use case: This functionality enables refunding gas to the transaction or block gas pools so that
// EVM-compatible chains can fully support the go-ethereum StateDb interface.
// See https://github.com/cosmos/cosmos-sdk/pull/9403 for reference.
func (g *Tax2GasMeter) RefundGas(amount sdk.Gas, descriptor string) {
	if g.consumed < amount {
		panic(ErrorNegativeGasConsumed{Descriptor: descriptor})
	}

	g.consumed -= amount
}

// IsPastLimit returns true if gas consumed is past limit, otherwise it returns false.
func (g *Tax2GasMeter) IsPastLimit() bool {
	return g.consumed > g.limit
}

// IsOutOfGas returns true if gas consumed is greater than or equal to gas limit, otherwise it returns false.
func (g *Tax2GasMeter) IsOutOfGas() bool {
	return g.consumed >= g.limit
}

// String returns the Tax2GasMeter's gas limit and gas consumed.
func (g *Tax2GasMeter) String() string {
	return fmt.Sprintf("Tax2GasMeter:\n  limit: %d\n  consumed: %d", g.limit, g.consumed)
}

// EnableGasLimitEnforcement enables the gas limit enforcement.
func (g *Tax2GasMeter) EnableGasLimitEnforcement() {
	g.limitEnforced = true
}

// DisableGasLimitEnforcement disables the gas limit enforcement.
func (g *Tax2GasMeter) DisableGasLimitEnforcement() {
	g.limitEnforced = false
}
