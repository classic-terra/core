package ante

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	marketexported "github.com/classic-terra/core/v4/x/market/exported"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	taxexemptionkeeper "github.com/classic-terra/core/v4/x/taxexemption/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

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
	return taxtypes.ComputeTaxes(ctx, principal, th.GetBurnTaxRate(ctx), simulate, tk)
}
