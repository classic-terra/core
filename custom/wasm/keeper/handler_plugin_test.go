package keeper

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"
)

type testRouter struct {
	handler baseapp.MsgServiceHandler
}

func (r testRouter) Handler(sdk.Msg) baseapp.MsgServiceHandler {
	return r.handler
}

type capturingICS4Wrapper struct {
	seq           uint64
	sourcePort    string
	sourceChannel string
	data          []byte
}

func (m *capturingICS4Wrapper) SendPacket(
	_ sdk.Context,
	sourcePort string,
	sourceChannel string,
	_ clienttypes.Height,
	_ uint64,
	data []byte,
) (uint64, error) {
	m.sourcePort = sourcePort
	m.sourceChannel = sourceChannel
	m.data = data
	return m.seq, nil
}

func (m *capturingICS4Wrapper) WriteAcknowledgement(sdk.Context, ibcexported.PacketI, ibcexported.Acknowledgement) error {
	return nil
}

type capturingChannelKeeperV2 struct {
	clientID string
	sequence uint64
	ack      channeltypesv2.Acknowledgement
}

func (m *capturingChannelKeeperV2) WriteAcknowledgement(
	_ sdk.Context,
	clientID string,
	sequence uint64,
	ack channeltypesv2.Acknowledgement,
) error {
	m.clientID = clientID
	m.sequence = sequence
	m.ack = ack
	return nil
}

func TestHandleSdkMessageRejectsProtoSignerMismatch(t *testing.T) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	contractAddr := sdk.AccAddress("contract-address")
	fromAddr := sdk.AccAddress("external-signer")
	toAddr := sdk.AccAddress("recipient")
	var bankKeeper bankkeeper.Keeper
	handler := NewMessageHandler(
		testRouter{},
		nil,
		nil,
		nil,
		bankKeeper,
		codectestutil.CodecOptions{}.NewCodec(),
		nil,
		&wasmkeeper.MessageEncoders{
			Custom: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				require.Equal(t, contractAddr, sender)
				return []sdk.Msg{banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("uluna", sdkmath.NewInt(1))))}, nil
			},
		},
	)
	_, _, _, err := handler.DispatchMsg(ctx, contractAddr, "", wasmvmtypes.CosmosMsg{Custom: []byte(`{}`)})
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
}

func TestNewMessageHandlerSetsReverseChargeContext(t *testing.T) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	contractAddr := sdk.AccAddress("contract-address")
	toAddr := sdk.AccAddress("recipient")
	toAddrBech32 := toAddr.String()
	called := false
	var bankKeeper bankkeeper.Keeper
	handler := NewMessageHandler(
		testRouter{
			handler: func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
				called = true
				require.Equal(t, true, ctx.Value(taxtypes.ContextKeyTaxReverseCharge))
				require.Equal(t, banktypes.NewMsgSend(contractAddr, toAddr, sdk.NewCoins(sdk.NewCoin("uluna", sdkmath.NewInt(1)))), msg)
				return &sdk.Result{}, nil
			},
		},
		nil,
		nil,
		nil,
		bankKeeper,
		codectestutil.CodecOptions{}.NewCodec(),
		nil,
	)
	_, _, _, err := handler.DispatchMsg(ctx, contractAddr, "", wasmvmtypes.CosmosMsg{Bank: &wasmvmtypes.BankMsg{Send: &wasmvmtypes.SendMsg{
		ToAddress: toAddrBech32,
		Amount:    wasmvmtypes.Array[wasmvmtypes.Coin]{{Denom: "uluna", Amount: "1"}},
	}}})
	require.NoError(t, err)
	require.True(t, called)
}

func TestNewMessageHandlerDispatchesIBCSendPacket(t *testing.T) {
	ics4Wrapper := &capturingICS4Wrapper{seq: 7}
	var bankKeeper bankkeeper.Keeper
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	handler := NewMessageHandler(
		testRouter{},
		nil,
		ics4Wrapper,
		nil,
		bankKeeper,
		codectestutil.CodecOptions{}.NewCodec(),
		nil,
	)

	_, data, msgResponses, err := handler.DispatchMsg(
		ctx,
		sdk.AccAddress("contract-address"),
		"wasm.port",
		wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{SendPacket: &wasmvmtypes.SendPacketMsg{
			ChannelID: "channel-7",
			Data:      []byte("payload"),
		}}},
	)
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.Len(t, msgResponses, 1)
	require.Equal(t, uint64(7), msgResponses[0][0].GetCachedValue().(*wasmtypes.MsgIBCSendResponse).Sequence)
	require.Equal(t, "wasm.port", ics4Wrapper.sourcePort)
	require.Equal(t, "channel-7", ics4Wrapper.sourceChannel)
	require.Equal(t, []byte("payload"), ics4Wrapper.data)
}

func TestNewMessageHandlerDispatchesIBC2WriteAcknowledgement(t *testing.T) {
	channelKeeperV2 := &capturingChannelKeeperV2{}
	var bankKeeper bankkeeper.Keeper
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	handler := NewMessageHandler(
		testRouter{},
		nil,
		nil,
		channelKeeperV2,
		bankKeeper,
		codectestutil.CodecOptions{}.NewCodec(),
		nil,
	)

	_, data, msgResponses, err := handler.DispatchMsg(
		ctx,
		sdk.AccAddress("contract-address"),
		"ibc2.port",
		wasmvmtypes.CosmosMsg{IBC2: &wasmvmtypes.IBC2Msg{WriteAcknowledgement: &wasmvmtypes.IBC2WriteAcknowledgementMsg{
			SourceClient:   "07-tendermint-0",
			PacketSequence: 11,
			Ack:            wasmvmtypes.IBCAcknowledgement{Data: []byte("ack")},
		}}},
	)
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.Len(t, msgResponses, 1)
	require.Equal(t, "07-tendermint-0", channelKeeperV2.clientID)
	require.Equal(t, uint64(11), channelKeeperV2.sequence)
	require.Equal(t, [][]byte{[]byte("ack")}, channelKeeperV2.ack.AppAcknowledgements)
	_, ok := msgResponses[0][0].GetCachedValue().(*wasmtypes.MsgIBCWriteAcknowledgementResponse)
	require.True(t, ok)
}
