package utils

import (
	"context"
	"fmt"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
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
	txf, err := clienttx.NewFactoryCLI(clientCtx, flagSet)
	if err != nil {
		return nil, err
	}

	gasStr, _ := flagSet.GetString(flags.FlagGas)
	switch gasStr {
	case flags.GasFlagAuto:
		txf = txf.WithGas(0)
		txf = txf.WithSimulateAndExecute(true)
	case "":
		txf = txf.WithGas(0)
		txf = txf.WithSimulateAndExecute(true)
	default:
		return nil, fmt.Errorf("current version can not support specific gas as it will cause exceed block max gas, please use --fees flag")
	}

	gas := txf.Gas()
	if txf.SimulateAndExecute() {
		txf, err := prepareFactory(clientCtx, txf)
		if err != nil {
			return nil, err
		}

		_, adj, err := CalculateGas(clientCtx, txf, msgs...)
		if err != nil {
			return nil, err
		}

		gas = adj
	}

	// As the tax is already converted to gas when simulating,
	// we don't need to calculate tax anymore
	fees := txf.Fees()
	gasPrices := txf.GasPrices()

	if !gasPrices.IsZero() {
		glDec := sdk.NewDec(int64(gas))
		adjustment := sdk.NewDecWithPrec(int64(txf.GasAdjustment()*100), 2)

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

// CalculateGas simulates the execution of a transaction and returns the
// simulation response obtained by the query and the adjusted gas amount.
func CalculateGas(
	clientCtx gogogrpc.ClientConn, txf clienttx.Factory, msgs ...sdk.Msg,
) (*tx.SimulateResponse, uint64, error) {
	txBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return nil, 0, err
	}

	txSvcClient := tx.NewServiceClient(clientCtx)
	simSpecialRes, err := txSvcClient.SimulateSpecial(context.Background(), &tx.SimulateSpecialRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, 0, err
	}
	simRes, err := txSvcClient.Simulate(context.Background(), &tx.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, 0, err
	}

	taxGas := simRes.GasInfo.GasUsed - simSpecialRes.GasInfo.GasUsed
	actualGas := uint64(txf.GasAdjustment() * float64(simSpecialRes.GasInfo.GasUsed))
	simRes.GasInfo = simSpecialRes.GasInfo
	return simRes, actualGas + taxGas, nil
}

// prepareFactory ensures the account defined by ctx.GetFromAddress() exists and
// if the account number and/or the account sequence number are zero (not set),
// they will be queried for and set on the provided Factory. A new Factory with
// the updated fields will be returned.
func prepareFactory(clientCtx client.Context, txf clienttx.Factory) (clienttx.Factory, error) {
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
