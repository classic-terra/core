package utils

import (
	"regexp"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	marketexported "github.com/classic-terra/core/v3/x/market/exported"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

func isIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}

// FilterMsgAndComputeTax computes the stability tax on messages.
func FilterMsgAndComputeTax(ctx sdk.Context, tk types.TreasuryKeeper, msgs ...sdk.Msg) sdk.Coins {
	taxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !tk.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Amount)...)
			}

		case *banktypes.MsgMultiSend:
			tainted := 0

			for _, input := range msg.Inputs {
				if tk.HasBurnTaxExemptionAddress(ctx, input.Address) {
					tainted++
				}
			}

			for _, output := range msg.Outputs {
				if tk.HasBurnTaxExemptionAddress(ctx, output.Address) {
					tainted++
				}
			}

			if tainted != len(msg.Inputs)+len(msg.Outputs) {
				for _, input := range msg.Inputs {
					taxes = taxes.Add(computeTax(ctx, tk, input.Coins)...)
				}
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(computeTax(ctx, tk, sdk.NewCoins(msg.OfferCoin))...)

		case *wasmtypes.MsgInstantiateContract:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)

		case *wasmtypes.MsgInstantiateContract2:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)

		case *wasmtypes.MsgExecuteContract:
			if !tk.HasBurnTaxExemptionContract(ctx, msg.Contract) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)
			}

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err == nil {
				taxes = taxes.Add(FilterMsgAndComputeTax(ctx, tk, messages...)...)
			}
		}
	}

	return taxes
}

// computes the stability tax according to tax-rate and tax-cap
func computeTax(ctx sdk.Context, tk types.TreasuryKeeper, principal sdk.Coins) sdk.Coins {
	taxRate := tk.GetTaxRate(ctx)
	if taxRate.Equal(sdk.ZeroDec()) {
		return sdk.Coins{}
	}

	taxes := sdk.Coins{}

	for _, coin := range principal {
		if coin.Denom == sdk.DefaultBondDenom {
			continue
		}

		if isIBCDenom(coin.Denom) {
			continue
		}

		taxDue := sdk.NewDecFromInt(coin.Amount).Mul(taxRate).TruncateInt()

		// If tax due is greater than the tax cap, cap!
		taxCap := tk.GetTaxCap(ctx, coin.Denom)
		if taxDue.GT(taxCap) {
			taxDue = taxCap
		}

		if taxDue.Equal(sdk.ZeroInt()) {
			continue
		}

		taxes = taxes.Add(sdk.NewCoin(coin.Denom, taxDue))
	}

	return taxes
}

func ComputeGas(gasPrices sdk.DecCoins, taxes sdk.Coins) (uint64, error) {
	taxes = taxes.Sort()
	tax2gas := sdkmath.ZeroInt()
	// Convert to gas
	i, j := 0, 0
	for i < len(gasPrices) && j < len(taxes) {
		switch {
		case gasPrices[i].Denom == taxes[j].Denom:
			tax2gas = tax2gas.Add(sdk.NewDec(taxes[j].Amount.Int64()).Quo((gasPrices[i].Amount)).Ceil().RoundInt())
			i++
			j++
		case gasPrices[i].Denom < taxes[j].Denom:
			i++
		default:
			j++
		}
	}

	return tax2gas.Uint64(), nil
}

func ComputeFeesOnGasConsumed(tx sdk.Tx, gasPrices sdk.DecCoins, gas uint64) (sdk.Coins, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	isOracleTx := IsOracleTx(feeTx.GetMsgs())

	gasFees := make(sdk.Coins, len(gasPrices))
	if !isOracleTx && len(gasPrices) != 0 {
		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
		glDec := sdk.NewDec(int64(gas))
		for i, gp := range gasPrices {
			fee := gp.Amount.Mul(glDec)
			gasFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}
	}

	return gasFees, nil
}