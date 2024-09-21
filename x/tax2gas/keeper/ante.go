package keeper

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	types "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// this keeper has been taken from the wasmd package

// Tax2GasDecorator ante decorator to limit gas in simulation calls
type Tax2GasDecorator struct {
	gasLimit *sdk.Gas
}

// NewTax2GasDecorator constructor accepts nil value to fallback to block gas limit.
func NewTax2GasDecorator(gasLimit *sdk.Gas) *Tax2GasDecorator {
	if gasLimit != nil && *gasLimit == 0 {
		panic("gas limit must not be zero")
	}
	return &Tax2GasDecorator{gasLimit: gasLimit}
}

// AnteHandle that limits the maximum gas available in simulations only.
// A custom max value can be configured and will be applied when set. The value should not
// exceed the max block gas limit.
// Different values on nodes are not consensus breaking as they affect only
// simulations but may have effect on client user experience.
//
// When no custom value is set then the max block gas is used as default limit.
func (d Tax2GasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate {
		// Wasm code is not executed in checkTX so that we don't need to limit it further.
		// Tendermint rejects the TX afterwards when the tx.gas > max block gas.
		// On deliverTX we rely on the tendermint/sdk mechanics that ensure
		// tx has gas set and gas < max block gas
		if ctx.BlockHeight() == 0 {
			return next(ctx.WithGasMeter(types.NewTax2GasMeter(0, true)), tx, simulate)
		}

		return next(ctx.WithGasMeter(types.NewTax2GasMeter(ctx.GasMeter().Limit(), false)), tx, simulate)
	}

	// apply custom node gas limit
	if d.gasLimit != nil {
		return next(ctx.WithGasMeter(types.NewTax2GasMeter(*d.gasLimit, false)), tx, simulate)
	}

	// default to max block gas when set, to be on the safe side
	if maxGas := ctx.ConsensusParams().GetBlock().MaxGas; maxGas > 0 {
		return next(ctx.WithGasMeter(types.NewTax2GasMeter(sdk.Gas(maxGas), false)), tx, simulate)
	}
	return next(ctx, tx, simulate)
}

// GasRegisterDecorator ante decorator to store gas register in the context
type GasRegisterDecorator struct {
	gasRegister wasmtypes.GasRegister
}

// NewGasRegisterDecorator constructor.
func NewGasRegisterDecorator(gr wasmtypes.GasRegister) *GasRegisterDecorator {
	return &GasRegisterDecorator{gasRegister: gr}
}

// AnteHandle adds the gas register to the context.
func (g GasRegisterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(wasmtypes.WithGasRegister(ctx, g.gasRegister), tx, simulate)
}
