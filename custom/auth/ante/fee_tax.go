package ante

import (
	"regexp"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	marketexported "github.com/classic-terra/core/v3/x/market/exported"
	taxexemptionkeeper "github.com/classic-terra/core/v3/x/taxexemption/keeper"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

func isIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}

// FilterMsgAndComputeTax computes the stability tax on messages.
func FilterMsgAndComputeTax(ctx sdk.Context, te taxexemptionkeeper.Keeper, tk TreasuryKeeper, th TaxKeeper, simulate bool, msgs ...sdk.Msg) (sdk.Coins, sdk.Coins) {
	taxes := sdk.Coins{}
	nonTaxableTaxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !te.IsExemptedFromTax(ctx, msg.FromAddress, msg.ToAddress) {
				taxes = taxes.Add(computeTax(ctx, tk, th, msg.Amount, simulate)...)
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
					taxes = taxes.Add(computeTax(ctx, tk, th, input.Coins, simulate)...)
				}
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(computeTax(ctx, tk, th, sdk.NewCoins(msg.OfferCoin), simulate)...)

		// The contract messages were disabled to remove double-taxation
		// whenever a contract sends funds to a wallet, it is taxed (deducted from sent amount)
		case *wasmtypes.MsgInstantiateContract:
			nonTaxableTaxes = nonTaxableTaxes.Add(computeTax(ctx, tk, th, msg.Funds, simulate)...)

		case *wasmtypes.MsgInstantiateContract2:
			nonTaxableTaxes = nonTaxableTaxes.Add(computeTax(ctx, tk, th, msg.Funds, simulate)...)

		case *wasmtypes.MsgExecuteContract:
			if !te.IsExemptedFromTax(ctx, msg.Sender, msg.Contract) {
				nonTaxableTaxes = nonTaxableTaxes.Add(computeTax(ctx, tk, th, msg.Funds, simulate)...)
			}
		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err == nil {
				execTaxes, execNonTaxable := FilterMsgAndComputeTax(ctx, te, tk, th, simulate, messages...)
				taxes = taxes.Add(execTaxes...)
				nonTaxableTaxes = nonTaxableTaxes.Add(execNonTaxable...)
			}
		}
	}

	return taxes, nonTaxableTaxes
}

// computes the stability tax according to tax-rate and tax-cap
func computeTax(ctx sdk.Context, tk TreasuryKeeper, th TaxKeeper, principal sdk.Coins, simulate bool) sdk.Coins {
	taxRate := th.GetBurnTaxRate(ctx)
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
