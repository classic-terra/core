package utils

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	"github.com/cosmos/cosmos-sdk/x/authz"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	marketexported "github.com/classic-terra/core/v3/x/market/exported"
	tax2gasexported "github.com/classic-terra/core/v3/x/tax2gas/exported"
	tax2gasutils "github.com/classic-terra/core/v3/x/tax2gas/utils"
)

type (
	// EstimateFeeResp defines a tx fee estimation response
	EstimateFeeResp struct {
		Fee legacytx.StdFee `json:"fee" yaml:"fee"`
	}
)

type ComputeReqParams struct {
	Memo          string
	ChainID       string
	AccountNumber uint64
	Sequence      uint64
	GasPrices     sdk.DecCoins
	Gas           string
	GasAdjustment string

	Msgs []sdk.Msg
}

// ComputeFeesWithCmd returns fee amount with cli options.
func ComputeFeesWithCmd(
	clientCtx client.Context, flagSet *pflag.FlagSet, msgs ...sdk.Msg,
) (*legacytx.StdFee, error) {
	txf, err := tx.NewFactoryCLI(clientCtx, flagSet)
	if err != nil {
		return nil, err
	}

	gas := txf.Gas()
	if txf.SimulateAndExecute() {
		txf, err := prepareFactory(clientCtx, txf)
		if err != nil {
			return nil, err
		}

		_, adj, err := tx.CalculateGas(clientCtx, txf, msgs...)
		if err != nil {
			return nil, err
		}

		gas = adj
	}

	// Computes taxes of the msgs
	taxes, err := FilterMsgAndComputeTax(clientCtx, msgs...)
	if err != nil {
		return nil, err
	}

	fees := txf.Fees().Add(taxes...)
	gasPrices := txf.GasPrices()

	if !gasPrices.IsZero() {
		glDec := sdk.NewDec(int64(gas))
		adjustment := sdk.NewDecWithPrec(int64(txf.GasAdjustment())*100, 2)

		if adjustment.LT(sdk.OneDec()) {
			adjustment = sdk.OneDec()
		}

		// Derive the fees based on the provided gas prices, where
		// fee = ceil(gasPrice * gasLimit).
		gasFees := make(sdk.Coins, len(gasPrices))
		for i, gp := range gasPrices {
			fee := gp.Amount.Mul(glDec).Mul(adjustment)
			gasFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}

		fees = fees.Add(gasFees.Sort()...)
	}

	return &legacytx.StdFee{
		Amount: fees,
		Gas:    gas,
	}, nil
}

// FilterMsgAndComputeTax computes the stability tax on MsgSend and MsgMultiSend.
func FilterMsgAndComputeTax(clientCtx client.Context, msgs ...sdk.Msg) (taxes sdk.Coins, err error) {
	burnTaxRate, err := queryTaxRate(clientCtx)
	if err != nil {
		return nil, err
	}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			tax := tax2gasutils.ComputeTax(burnTaxRate, msg.Amount)
			taxes = taxes.Add(tax...)

		case *banktypes.MsgMultiSend:
			for _, input := range msg.Inputs {
				tax := tax2gasutils.ComputeTax(burnTaxRate, input.Coins)
				taxes = taxes.Add(tax...)
			}

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				panic(err)
			}

			tax, err := FilterMsgAndComputeTax(clientCtx, messages...)
			if err != nil {
				return nil, err
			}

			taxes = taxes.Add(tax...)

		case *marketexported.MsgSwapSend:
			tax := tax2gasutils.ComputeTax(burnTaxRate, sdk.NewCoins(msg.OfferCoin))
			taxes = taxes.Add(tax...)

		case *wasmtypes.MsgInstantiateContract:
			tax := tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)
			taxes = taxes.Add(tax...)

		case *wasmtypes.MsgInstantiateContract2:
			tax := tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)
			taxes = taxes.Add(tax...)

		case *wasmtypes.MsgExecuteContract:
			tax := tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)
			taxes = taxes.Add(tax...)
		}
	}

	return taxes, nil
}

func queryTaxRate(clientCtx client.Context) (sdk.Dec, error) {
	queryClient := tax2gasexported.NewQueryClient(clientCtx)

	res, err := queryClient.BurnTaxRate(context.Background(), &tax2gasexported.QueryBurnTaxRateRequest{})
	if err != nil {
		return sdk.ZeroDec(), err
	}
	return res.BurnTaxRate, err
}

// prepareFactory ensures the account defined by ctx.GetFromAddress() exists and
// if the account number and/or the account sequence number are zero (not set),
// they will be queried for and set on the provided Factory. A new Factory with
// the updated fields will be returned.
func prepareFactory(clientCtx client.Context, txf tx.Factory) (tx.Factory, error) {
	from := clientCtx.GetFromAddress()

	if err := txf.AccountRetriever().EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := txf.AccountRetriever().GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}
