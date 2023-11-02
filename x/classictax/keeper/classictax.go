package keeper

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm"
	core "github.com/classic-terra/core/v2/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	marketexported "github.com/classic-terra/core/v2/x/market/exported"
	oracleexported "github.com/classic-terra/core/v2/x/oracle/exported"
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

// this function uses on-chain parameters for gas prices to calculate
// a gas amount that equals the tax amount
func (k Keeper) CalculateTaxGas(ctx sdk.Context, taxes sdk.Coins, gasPrices sdk.DecCoins) (uint64, error) {
	taxGas := uint64(0)

	// we are using uluna as a measuring point for the gas of taxes
	for _, tax := range taxes {
		// ensure that gasPrice isn't zero
		gasPrice := gasPrices.AmountOf(tax.Denom)
		if gasPrice.IsZero() {
			return 0, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "gas price for %s is zero", tax.Denom)
		}

		// calculate tax gas
		taxForGas := sdk.NewDecFromBigInt(tax.Amount.BigInt()).Quo(gasPrice)
		taxGasAmount := taxForGas.TruncateInt64()

		if taxGasAmount > 0 {
			taxGas += uint64(taxGasAmount)
		}
	}

	return taxGas, nil
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
			// IBC denom are excluded from tax, due to a passed governance proposal
			continue
		}

		taxDue := sdk.NewDecFromInt(coin.Amount).Mul(taxRate).Ceil().RoundInt()

		// If tax due is greater than the tax cap, cap!
		if taxDue.Equal(sdk.ZeroInt()) {
			continue
		}

		taxes = taxes.Add(sdk.NewCoin(coin.Denom, taxDue))
	}

	return taxes
}

// this function calculates the required fees for a transaction
// based on a provided gas value and the on-chain gas price parameters
func (k Keeper) GetFeeCoins(ctx sdk.Context, gas uint64, stabilityTaxes sdk.Coins) (sdk.Coins, sdk.Coin) {
	requiredGasFees := sdk.Coins{}
	requiredGasFeesUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	gasPrices := k.GetGasPrices(ctx)
	k.Logger(ctx).Info("GasPrices", "GasPrices", gasPrices)
	if !gasPrices.IsZero() {
		requiredGasFees = make(sdk.Coins, 0, len(gasPrices))

		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(gasPrice * gasLimit).
		glDec := sdk.NewDec(int64(gas))
		for _, gp := range gasPrices {
			fee := gp.Amount.Mul(glDec)
			coin := sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			requiredGasFees = append(requiredGasFees, coin)
			if gp.Denom == core.MicroLunaDenom {
				requiredGasFeesUluna = sdk.NewCoin(core.MicroLunaDenom, fee.Ceil().RoundInt())
				break
			} else {
				// to get an optional uluna amount for paying the whole tax in uluna,
				// we take the highest equivalent of the gas prices
				inUluna := k.CoinToMicroLuna(ctx, sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt()))
				if requiredGasFeesUluna.IsLT(inUluna) {
					requiredGasFeesUluna = inUluna
				}
			}
		}
	}

	stabilityTaxes = stabilityTaxes.Sort()
	requiredFees := requiredGasFees.Sort()

	if !stabilityTaxes.IsZero() {
		requiredFees = requiredFees.Add(stabilityTaxes...)
	}

	return requiredFees, requiredGasFeesUluna
}

// this function calculates the coins necessary for paying the tax
// tax can either be paid as previously in the corresponding denom
// or totally in uluna
func (k Keeper) GetTaxCoins(ctx sdk.Context, msgs ...sdk.Msg) (sdk.Coins, sdk.Coin) {
	// define empty coins list
	taxes := sdk.NewCoins()
	taxesUluna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	// read taxable message types from params
	taxableMsgTypes := k.GetTaxableMsgTypes(ctx)

	// read taxable message types from params
	for _, msg := range msgs {
		taxable := false
		for _, msgType := range taxableMsgTypes {
			// get the type string (e.g. types.MsgSend)
			// TODO check if this needs to be improved
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

		// calculate the tax needed for this message
		// TODO check for further message types that might be able to be taxed
		// as the taxable msg types can be changed by governance
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !k.treasuryKeeper.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
				tax = k.ComputeBurnTax(ctx, msg.Amount)
			}

		case *banktypes.MsgMultiSend:
			tainted := 0

			for _, input := range msg.Inputs {
				if k.treasuryKeeper.HasBurnTaxExemptionAddress(ctx, input.Address) {
					tainted++
				}
			}

			for _, output := range msg.Outputs {
				if k.treasuryKeeper.HasBurnTaxExemptionAddress(ctx, output.Address) {
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
			if !k.treasuryKeeper.HasBurnTaxExemptionContract(ctx, msg.Contract) {
				tax = k.ComputeBurnTax(ctx, msg.Funds)
			}

		case *stakingtypes.MsgDelegate:
		case *stakingtypes.MsgUndelegate:
			tax = k.ComputeBurnTax(ctx, sdk.NewCoins(sdk.NewCoin(core.MicroLunaDenom, msg.Amount.Amount)))

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				tax, taxUluna = k.GetTaxCoins(ctx, messages...)
			}
		}

		if tax != nil {
			taxes = taxes.Add(tax...)
		}

		if taxUluna.IsZero() && tax != nil && !tax.IsZero() {
			// if the tax is not already in uluna, convert it from oracle exchange rates
			taxUluna = k.CoinsToMicroLuna(ctx, tax)
		}

		if !taxUluna.IsZero() {
			taxesUluna = taxesUluna.Add(taxUluna)
		}
	}

	return taxes, taxesUluna
}

// convert a coin to uluna by using the oracle exchange rates
func (k Keeper) CoinToMicroLuna(ctx sdk.Context, coin sdk.Coin) sdk.Coin {
	microLuna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	if coin.Denom == core.MicroLunaDenom {
		microLuna = microLuna.Add(coin)
	} else {
		// get the exchange rate
		exchangeRate, err := k.oracleKeeper.GetLunaExchangeRate(ctx, coin.Denom)
		if err != nil && !exchangeRate.IsZero() {
			// convert to micro luna
			amount := sdk.NewDecFromInt(coin.Amount).Quo(exchangeRate).TruncateInt()
			microLuna = microLuna.Add(sdk.NewCoin(core.MicroLunaDenom, amount))
		}
	}

	return microLuna
}

func (k Keeper) CoinsToMicroLuna(ctx sdk.Context, coins sdk.Coins) sdk.Coin {
	microLuna := sdk.NewCoin(core.MicroLunaDenom, sdk.ZeroInt())

	for _, coin := range coins {
		microLuna = microLuna.Add(k.CoinToMicroLuna(ctx, coin))
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

// this function calculates the tax that was sent with the transaction (separates it from the gas fees)
// this should only be used in post handler due to the fact that it checks consumed gas instead of sent gas
func (k Keeper) CalculateSentTax(ctx sdk.Context, feeTx sdk.FeeTx, stabilityTaxes sdk.Coins) (sdk.DecCoins, sdk.Coins, uint64, error) {
	gas := feeTx.GetGas()
	gasConsumed := ctx.GasMeter().GasConsumed()
	fee := feeTx.GetFee()

	taxes, taxesUluna := k.GetTaxCoins(ctx, feeTx.GetMsgs()...)
	requiredFees, requiredFeesUluna := k.GetFeeCoins(ctx, gasConsumed, stabilityTaxes)

	// get the tax equivalent in gas
	taxGas, err := k.CalculateTaxGas(ctx, taxes, k.GetGasPrices(ctx))
	if err != nil {
		return nil, nil, gas, err
	}

	// calculate the ratio of the tax to the gas
	sentFeesUluna := sdk.NewDec(k.CoinsToMicroLuna(ctx, fee).Amount.Int64())
	feeGasUluna := sdk.NewDec(requiredFeesUluna.Amount.Int64())
	feeTaxUluna := sdk.NewDec(taxesUluna.Amount.Int64())

	k.Logger(ctx).Info("CalculateSentTax", "sentFeesUluna", sentFeesUluna, "feeGasUluna", feeGasUluna, "feeTaxUluna", feeTaxUluna, "requiredFeesUluna", requiredFeesUluna, "requiredFees", requiredFees, "taxesUluna", taxesUluna, "taxes", taxes, "checktx", ctx.IsCheckTx())

	if feeTaxUluna.IsZero() {
		return nil, nil, gas, nil
	}

	// calculate the assumed multiplier that was used to calculate fees to send (gas * multiplier * gasPrice = sentFees)
	multiplier := sdk.NewDec(int64(gas)).Quo(sdk.NewDec(int64(gasConsumed)).Add(sdk.NewDec(int64(taxGas))))

	sentFeesTax := sdk.NewDecCoinsFromCoins(taxes...)
	sentTaxGas := sdk.NewDec(int64(taxGas))

	// this is the gas amount without the tax gas
	reducedGas := sdk.NewDec(int64(gas)).Sub(sentTaxGas)

	k.Logger(ctx).Info("CalculateSentTax", "assumed_multiplier", multiplier, "gas", gas, "assumed_gas_estimate", reducedGas, "taxgas", taxGas)

	// at this point we calculate the potion of the sent fee coins that is tax
	// this is done to only deduct the full sent gas fees, but not excessive tax from the user's account
	sentFeesTax = sentFeesTax.MulDecTruncate(multiplier)
	coins, _ := sentFeesTax.TruncateDecimal()

	reducedFee, neg := fee.SafeSub(coins...)
	if neg {
		// this should never happen, but we check it anyway to be on the safe side
		return nil, nil, gas, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %q required: %q - TODO 1", fee, requiredFees)
	}

	// return the full fees sent as tax, the sent fees reduced by that amount and the gas without taxgas
	return sentFeesTax, reducedFee, reducedGas.TruncateInt().Uint64(), nil
}
