package e2e

import (
	"encoding/base64"
	// "path/filepath"

	"github.com/classic-terra/core/v2/tests/e2e/initialization"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	rawTxFile = "tx_raw.json"
)

// buildRawTx build a dummy tx using the TxBuilder and
// return the JSON and encoded tx's
func buildRawTx() ([]byte, string, error) {
	builder := initialization.TxConfig.NewTxBuilder()
	builder.SetGasLimit(gas)
	builder.SetFeeAmount(sdk.NewCoins(standardFees))
	builder.SetMemo("foomemo")
	tx, err := initialization.TxConfig.TxJSONEncoder()(builder.GetTx())
	if err != nil {
		return nil, "", err
	}
	txBytes, err := initialization.TxConfig.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, "", err
	}
	return tx, base64.StdEncoding.EncodeToString(txBytes), err
}
