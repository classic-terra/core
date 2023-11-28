package ante

import (
	"regexp"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	marketexported "github.com/classic-terra/core/v2/x/market/exported"
	oracleexported "github.com/classic-terra/core/v2/x/oracle/exported"
	taxexemptionkeeper "github.com/classic-terra/core/v2/x/taxexemption/keeper"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

func isIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}

// FilterMsgAndComputeTax computes the stability tax on messages.
func FilterMsgAndComputeTax(ctx sdk.Context, te taxexemptionkeeper.Keeper, tk TreasuryKeeper, msgs ...sdk.Msg) sdk.Coins {
	taxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !te.IsExemptedFromTax(ctx, msg.FromAddress, msg.ToAddress) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Amount)...)
			}

		case *banktypes.MsgMultiSend:
			tainted := 0

			// make list of output addresses
			outputAddresses := make([]string, len(msg.Outputs))
			for i, output := range msg.Outputs {
				outputAddresses[i] = output.Address
			}

			for _, input := range msg.Inputs {
				if te.IsExemptedFromTax(ctx, input.Address, outputAddresses...) {
					tainted++
				}
			}

			if tainted != len(msg.Inputs) {
				for _, input := range msg.Inputs {
					taxes = taxes.Add(computeTax(ctx, tk, input.Coins)...)
				}
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(computeTax(ctx, tk, sdk.NewCoins(msg.OfferCoin))...)

		case *wasm.MsgInstantiateContract:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)

		case *wasm.MsgInstantiateContract2:
			taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)

		case *wasm.MsgExecuteContract:
			if !te.IsExemptedFromTax(ctx, msg.Sender, msg.Contract) {
				taxes = taxes.Add(computeTax(ctx, tk, msg.Funds)...)
			}

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				taxes = taxes.Add(FilterMsgAndComputeTax(ctx, te, tk, messages...)...)
			}
		}
	}

	return taxes
}

// computes the stability tax according to tax-rate and tax-cap
func computeTax(ctx sdk.Context, tk TreasuryKeeper, principal sdk.Coins) sdk.Coins {
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

func isOracleTx(msgs []sdk.Msg) bool {
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
