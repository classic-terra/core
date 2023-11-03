package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	expectedkeeper "github.com/classic-terra/core/v2/custom/auth/keeper"
	core "github.com/classic-terra/core/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	marketexported "github.com/classic-terra/core/v2/x/market/exported"
)

// FilterMsgAndComputeStabilityTax computes the stability tax on messages.
func FilterMsgAndComputeStabilityTax(ctx sdk.Context, tk expectedkeeper.TreasuryKeeper, msgs ...sdk.Msg) sdk.Coins {
	taxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			taxes = taxes.Add(computeStabilityTax(ctx, tk, msg.Amount)...)

		case *banktypes.MsgMultiSend:
			for _, input := range msg.Inputs {
				taxes = taxes.Add(computeStabilityTax(ctx, tk, input.Coins)...)
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(computeStabilityTax(ctx, tk, sdk.NewCoins(msg.OfferCoin))...)

		case *wasm.MsgInstantiateContract:
			taxes = taxes.Add(computeStabilityTax(ctx, tk, msg.Funds)...)

		case *wasm.MsgInstantiateContract2:
			taxes = taxes.Add(computeStabilityTax(ctx, tk, msg.Funds)...)

		case *wasm.MsgExecuteContract:
			taxes = taxes.Add(computeStabilityTax(ctx, tk, msg.Funds)...)

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				taxes = taxes.Add(FilterMsgAndComputeStabilityTax(ctx, tk, messages...)...)
			}
		}
	}

	return taxes.Sort()
}

// computes the stability tax according to tax-rate and tax-cap
func computeStabilityTax(ctx sdk.Context, tk expectedkeeper.TreasuryKeeper, principal sdk.Coins) sdk.Coins {
	taxRate := tk.GetTaxRate(ctx)

	if taxRate.Equal(sdk.ZeroDec()) {
		return sdk.Coins{}
	}

	taxes := sdk.Coins{}

	for _, coin := range principal {
		// reverted to stability tax behavior of not taxing uluna
		if coin.Denom == sdk.DefaultBondDenom || coin.Denom == core.MicroLunaDenom {
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
