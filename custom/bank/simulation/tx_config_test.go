package simulation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestSimulationTxConfigBankSigners(t *testing.T) {
	config := makeTxConfig()
	from := sdk.AccAddress(make([]byte, 20))
	to := sdk.AccAddress([]byte("another-account-addr"))
	builder := config.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(from, to, sdk.NewCoins())))
	signers, err := builder.GetTx().GetSigners()
	require.NoError(t, err)
	require.Equal(t, [][]byte{from.Bytes()}, signers)
}
