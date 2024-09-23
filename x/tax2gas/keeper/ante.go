package keeper

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	types "github.com/classic-terra/core/v3/x/tax2gas/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ante "github.com/cosmos/cosmos-sdk/x/auth/ante"
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
func (d Tax2GasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	gasTx, ok := tx.(ante.GasTx)
	if !ok {
		// Set a gas meter with limit 0 as to prevent an infinite gas meter attack
		// during runTx.
		newCtx = SetGasMeter(simulate, ctx, 0)
		return newCtx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be GasTx")
	}

	ctx.Logger().Debug("Tax2GasDecorator.AnteHandle", "simulate", simulate, "gaswanted", gasTx.GetGas())

	newCtx = ctx // default to old context for defer handling

	// gas := gasTx.GetGas()
	// newCtx = SetGasMeter(simulate, ctx, gas)

	// Decorator will catch an OutOfGasPanic caused in the next antehandler
	// AnteHandlers must have their own defer/recover in order for the BaseApp
	// to know how much gas was used! This is because the GasMeter is created in
	// the AnteHandler, but if it panics the context won't be set properly in
	// runTx's recover call.
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				log := fmt.Sprintf(
					"out of gas in location: %v; gasWanted: %d, gasUsed: %d",
					rType.Descriptor, gasTx.GetGas(), newCtx.GasMeter().GasConsumed())

				err = sdkerrors.Wrap(sdkerrors.ErrOutOfGas, log)
			default:
				panic(r)
			}
		}
	}()

	if !simulate {
		// Wasm code is not executed in checkTX so that we don't need to limit it further.
		// Tendermint rejects the TX afterwards when the tx.gas > max block gas.
		// On deliverTX we rely on the tendermint/sdk mechanics that ensure
		// tx has gas set and gas < max block gas
		if ctx.BlockHeight() == 0 {
			return next(ctx.WithGasMeter(types.NewTax2GasMeter(0, true)), tx, simulate)
		}

		return next(ctx.WithGasMeter(types.NewTax2GasMeter(gasTx.GetGas(), false)), tx, simulate)
	}

	// apply custom node gas limit
	if d.gasLimit != nil {
		return next(ctx.WithGasMeter(types.NewTax2GasMeter(*d.gasLimit, false)), tx, simulate)
	}

	// default to max block gas when set, to be on the safe side
	if maxGas := ctx.ConsensusParams().GetBlock().MaxGas; maxGas > 0 {
		return next(ctx.WithGasMeter(types.NewTax2GasMeter(sdk.Gas(maxGas), false)), tx, simulate)
	}

	return next(ctx.WithGasMeter(types.NewTax2GasMeter(gasTx.GetGas(), false)), tx, simulate)
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

// SetGasMeter returns a new context with a gas meter set from a given context.
func SetGasMeter(simulate bool, ctx sdk.Context, gasLimit uint64) sdk.Context {
	// In various cases such as simulation and during the genesis block, we do not
	// meter any gas utilization.
	if simulate || ctx.BlockHeight() == 0 {
		return ctx.WithGasMeter(types.NewTax2GasMeter(0, true))
	}

	return ctx.WithGasMeter(types.NewTax2GasMeter(gasLimit, false))
}
