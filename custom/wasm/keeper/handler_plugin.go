package keeper

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	taxtypes "github.com/classic-terra/core/v4/x/tax/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankKeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

func NewMessageHandler(
	router wasmkeeper.MessageRouter,
	wasmKeeper wasmtypes.IBCContractKeeper,
	ics4Wrapper wasmtypes.ICS4Wrapper,
	channelKeeperV2 wasmtypes.ChannelKeeperV2,
	bankKeeper bankKeeper.Keeper,
	cdc codec.Codec,
	portSource wasmtypes.ICS20TransferPortSource,
	customEncoders ...*wasmkeeper.MessageEncoders,
) wasmkeeper.Messenger {
	encoders := wasmkeeper.DefaultEncoders(cdc, portSource)
	for _, e := range customEncoders {
		encoders = encoders.Merge(e)
	}
	sdkHandler := wasmkeeper.NewSDKMessageHandler(cdc, router, encoders)
	wrappedSDKHandler := wasmkeeper.MessageHandlerFunc(func(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		contractIBCPortID string,
		msg wasmvmtypes.CosmosMsg,
	) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
		ctx = ctx.WithValue(taxtypes.ContextKeyTaxReverseCharge, true)
		return sdkHandler.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	})
	return wasmkeeper.NewMessageHandlerChain(
		wrappedSDKHandler,
		wasmkeeper.NewIBCRawPacketHandler(ics4Wrapper, wasmKeeper),
		wasmkeeper.NewIBC2RawPacketHandler(channelKeeperV2),
		wasmkeeper.NewBurnCoinMessageHandler(bankKeeper),
	)
}
