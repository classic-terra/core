package keeper

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm"
	expectedkeeper "github.com/classic-terra/core/v2/custom/auth/keeper"
	core "github.com/classic-terra/core/v2/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	marketexported "github.com/classic-terra/core/v2/x/market/exported"
	oracleexported "github.com/classic-terra/core/v2/x/oracle/exported"
	oraclekeeper "github.com/classic-terra/core/v2/x/oracle/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

func isIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}

func (k Keeper) ContainsDenom(coins sdk.Coins, denom string) bool {
	return coins.AmountOf(denom).GT(sdk.ZeroInt())
}

func (k Keeper) CalculateTaxGas(ctx sdk.Context, taxes sdk.Coins, gasPrice sdk.Dec) uint64 {
	taxGas := uint64(0)

	for _, tax := range taxes {
		// ensure that gasPrice isn't zero
		if !gasPrice.IsZero() {
			// calculate tax gas
			taxForGas := sdk.NewDecFromBigInt(tax.Amount.BigInt()).Quo(gasPrice)
			taxGasAmount := taxForGas.TruncateInt64()

			if taxGasAmount > 0 {
				taxGas += uint64(taxGasAmount)
			}
		}
	}

	return taxGas
}

func (k Keeper) ComputeBurnTax(ctx sdk.Context, principal sdk.Coins) sdk.Coins {
	taxRate := k.GetBurnTaxRate(ctx)

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
		if taxDue.Equal(sdk.ZeroInt()) {
			continue
		}

		taxes = taxes.Add(sdk.NewCoin(coin.Denom, taxDue))
	}

	return taxes
}

func (k Keeper) GetFeeCoins(ctx sdk.Context, gas uint64, taxes sdk.Coins, ok oraclekeeper.Keeper) (sdk.Coins, sdk.Coin) {
	requiredGasFees := sdk.Coins{}
	requiredGasFeesUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	minGasPrices := ctx.MinGasPrices()
	if !minGasPrices.IsZero() {
		requiredGasFees = make(sdk.Coins, len(minGasPrices))

		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
		glDec := sdk.NewDec(int64(gas))
		for i, gp := range minGasPrices {
			fee := gp.Amount.Mul(glDec)
			requiredGasFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			if gp.Denom == core.MicroLunaDenom {
				requiredGasFeesUluna = sdk.NewCoin(core.MicroLunaDenom, fee.Ceil().RoundInt())
				break
			} else {
				inUluna := k.CoinToMicroLuna(ctx, ok, sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt()))
				if requiredGasFeesUluna.IsLT(inUluna) {
					requiredGasFeesUluna = inUluna
				}
			}
		}
	}

	requiredFees := requiredGasFees.Add(taxes...)

	return requiredFees, requiredGasFeesUluna
}

func (k Keeper) GetTaxCoins(ctx sdk.Context, tk expectedkeeper.TreasuryKeeper, ok oraclekeeper.Keeper, msgs ...sdk.Msg) (sdk.Coins, sdk.Coin) {
	// define empty coins list
	taxes := sdk.NewCoins()
	taxesUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	// read taxable message types from params
	taxableMsgTypes := k.GetTaxableMsgTypes(ctx)

	// read taxable message types from params
	for _, msg := range msgs {
		taxable := false
		for _, msgType := range taxableMsgTypes {
			tp := strings.TrimLeft(reflect.TypeOf(msg).String(), "*")
			if tp == msgType {
				taxable = true
				break
			}
		}

		if !taxable {
			continue
		}

		var tax sdk.Coins
		taxUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !tk.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
				tax = k.ComputeBurnTax(ctx, msg.Amount)
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
					tax = k.ComputeBurnTax(ctx, input.Coins)
				}
			}

		case *marketexported.MsgSwapSend:
			tax = k.ComputeBurnTax(ctx, sdk.NewCoins(msg.OfferCoin))

		case *wasm.MsgInstantiateContract:
		case *wasm.MsgInstantiateContract2:
			tax = k.ComputeBurnTax(ctx, msg.Funds)

		case *wasm.MsgExecuteContract:
			if !tk.HasBurnTaxExemptionContract(ctx, msg.Contract) {
				tax = k.ComputeBurnTax(ctx, msg.Funds)
			}

		case *stakingtypes.MsgDelegate:
		case *stakingtypes.MsgUndelegate:
			tax = k.ComputeBurnTax(ctx, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, msg.Amount.Amount)))

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				tax, taxUluna = k.GetTaxCoins(ctx, tk, ok, messages...)
			}
		}

		if tax != nil {
			taxes = taxes.Add(tax...)
		}

		if taxUluna.IsZero() && tax != nil && !tax.IsZero() {
			taxUluna = k.CoinsToMicroLuna(ctx, ok, tax)
		}

		if !taxUluna.IsZero() {
			taxesUluna = taxesUluna.Add(taxUluna)
		}
	}

	return taxes, taxesUluna
}

func (k Keeper) CoinToMicroLuna(ctx sdk.Context, ok oraclekeeper.Keeper, coin sdk.Coin) sdk.Coin {
	microLuna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	if coin.Denom == core.MicroLunaDenom {
		microLuna = microLuna.Add(coin)
	} else {
		// get the exchange rate
		exchangeRate, err := ok.GetLunaExchangeRate(ctx, coin.Denom)
		if err != nil && !exchangeRate.IsZero() {
			// convert to micro luna
			amount := sdk.NewDecFromInt(coin.Amount).Quo(exchangeRate).TruncateInt()
			microLuna = microLuna.Add(sdk.NewCoin(core.MicroLunaDenom, amount))
		}
	}

	return microLuna
}

func (k Keeper) CoinsToMicroLuna(ctx sdk.Context, ok oraclekeeper.Keeper, coins sdk.Coins) sdk.Coin {
	microLuna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	for _, coin := range coins {
		microLuna = microLuna.Add(k.CoinToMicroLuna(ctx, ok, coin))
	}

	return microLuna
}

func (k Keeper) IsOracleTx(msgs []sdk.Msg) bool {
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

func (k Keeper) CalculateSentTax(ctx sdk.Context, feeTx sdk.FeeTx, stabilityTaxes sdk.Coins, tk expectedkeeper.TreasuryKeeper, ok oraclekeeper.Keeper) (sdk.DecCoins, sdk.Coins, error) {
	gas := feeTx.GetGas()
	fee := feeTx.GetFee()

	_, taxesUluna := k.GetTaxCoins(ctx, tk, ok, feeTx.GetMsgs()...)
	requiredFees, requiredFeesUluna := k.GetFeeCoins(ctx, gas, stabilityTaxes, ok)

	// calculate the ratio of the tax to the gas
	sentFeesUluna := sdk.NewDec(k.CoinsToMicroLuna(ctx, ok, fee).Amount.Int64())
	feeGasUluna := sdk.NewDec(requiredFeesUluna.Amount.Int64())
	feeTaxUluna := sdk.NewDec(taxesUluna.Amount.Int64())

	if feeTaxUluna.IsZero() {
		return nil, nil, nil
	}

	// calculate the assumed multiplier that was used to calculate fees to send (gas * multiplier * gasPrice = sentFees)
	multiplier := sentFeesUluna.Quo(feeGasUluna.Add(feeTaxUluna))

	sentFeesTax := sdk.NewDecCoinsFromCoins(fee...)
	sentFeesTax = sentFeesTax.MulDecTruncate(multiplier)

	coins, _ := sentFeesTax.TruncateDecimal()
	neg := false
	fee, neg = fee.SafeSub(coins...)
	if neg {
		return nil, fee, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", fee, requiredFees)
	}

	return sentFeesTax, fee, nil
}
