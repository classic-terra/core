package cli

import (
	"bytes"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/stretchr/testify/require"
)

func TestNormalizeConsAddressArg_AcceptsConsensusAddress(t *testing.T) {
	clientCtx := makeTestClientContext()
	pubKey := ed25519.GenPrivKey().PubKey()
	consAddr := sdk.GetConsAddress(pubKey).String()

	got, err := normalizeConsAddressArg(clientCtx, consAddr)
	require.NoError(t, err)
	require.Equal(t, consAddr, got)
}

func TestNormalizeConsAddressArg_AcceptsConsensusPubKeyJSON(t *testing.T) {
	clientCtx := makeTestClientContext()
	pubKey := ed25519.GenPrivKey().PubKey()

	bz, err := clientCtx.Codec.MarshalInterfaceJSON(pubKey)
	require.NoError(t, err)

	got, err := normalizeConsAddressArg(clientCtx, string(bz))
	require.NoError(t, err)
	require.Equal(t, sdk.GetConsAddress(pubKey).String(), got)
}

func TestPrintProtoIncludesZeroValueSigningFields(t *testing.T) {
	clientCtx := makeTestClientContext()
	var out bytes.Buffer

	clientCtx = clientCtx.WithOutput(&out).WithOutputFormat("json")

	res := &slashingtypes.QuerySigningInfoResponse{
		ValSigningInfo: slashingtypes.ValidatorSigningInfo{
			Address:     sdk.GetConsAddress(ed25519.GenPrivKey().PubKey()).String(),
			StartHeight: 27267533,
			IndexOffset: 2826250,
		},
	}

	err := clientCtx.PrintProto(res)
	require.NoError(t, err)
	require.Contains(t, out.String(), `"tombstoned":false`)
	require.Contains(t, out.String(), `"missed_blocks_counter":"0"`)
}

func makeTestClientContext() client.Context {
	ir := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(ir)

	return client.Context{}.WithCodec(codec.NewProtoCodec(ir))
}
