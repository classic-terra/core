package ante

import (
	"regexp"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	marketexported "github.com/classic-terra/core/v3/x/market/exported"
	oracleexported "github.com/classic-terra/core/v3/x/oracle/exported"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

func isIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}

// FilterMsgAndComputeTax computes the stability tax on messages.
func FilterMsgAndComputeTax(ctx sdk.Context, tk TreasuryKeeper, simulate bool, msgs ...sdk.Msg) sdk.Coins {
	taxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !tk.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Amount, simulate)...)
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
					taxes = taxes.Add(computeTax(ctx, tk, input.Coins, simulate)...)
				}
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(computeTax(ctx, tk, sdk.NewCoins(msg.OfferCoin), simulate)...)

		case *wasmtypes.MsgInstantiateContract:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds, simulate)...)

		case *wasmtypes.MsgInstantiateContract2:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds, simulate)...)

		case *wasmtypes.MsgExecuteContract:
			if !tk.HasBurnTaxExemptionContract(ctx, msg.Contract) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Funds, simulate)...)
			}

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err == nil {
				taxes = taxes.Add(FilterMsgAndComputeTax(ctx, tk, simulate, messages...)...)
			}
		}
	}

	return taxes
}

// computes the stability tax according to tax-rate and tax-cap
func computeTax(ctx sdk.Context, tk TreasuryKeeper, principal sdk.Coins, simulate bool) sdk.Coins {
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
		// we need to check all taxes if they are GTE 100 because otherwise we will not be able to
		// simulate the split processes (i.e. BurnTaxSplit and OracleSplit)
		// if they are less than 100, we will set them to 100
		if simulate && taxDue.LT(sdk.NewInt(100)) {
			taxDue = sdk.NewInt(100)
		}

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

func IsOracleTx(msgs []sdk.Msg) bool {
	if len(msgs) == 0 {
		return false
	}
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
