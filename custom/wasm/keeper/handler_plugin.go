package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	bankKeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	marketexported "github.com/classic-terra/core/v3/x/market/exported"
	tax2gaskeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	tax2gasutils "github.com/classic-terra/core/v3/x/tax2gas/utils"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
)

// msgEncoder is an extension point to customize encodings
type msgEncoder interface {
	// Encode converts wasmvm message to n cosmos message types
	Encode(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Msg, error)
}

// MessageRouter ADR 031 request type routing
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}

// SDKMessageHandler can handles messages that can be encoded into sdk.Message types and routed.
type SDKMessageHandler struct {
	router         MessageRouter
	encoders       msgEncoder
	treasuryKeeper treasurykeeper.Keeper
	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     bankKeeper.Keeper
	tax2gaskeeper  tax2gaskeeper.Keeper
}

func NewMessageHandler(
	router MessageRouter,
	ics4Wrapper wasmtypes.ICS4Wrapper,
	channelKeeper wasmtypes.ChannelKeeper,
	capabilityKeeper wasmtypes.CapabilityKeeper,
	bankKeeper bankKeeper.Keeper,
	treasuryKeeper treasurykeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	tax2gaskeeper tax2gaskeeper.Keeper,
	unpacker codectypes.AnyUnpacker,
	portSource wasmtypes.ICS20TransferPortSource,
	customEncoders ...*wasmkeeper.MessageEncoders,
) wasmkeeper.Messenger {
	encoders := wasmkeeper.DefaultEncoders(unpacker, portSource)
	for _, e := range customEncoders {
		encoders = encoders.Merge(e)
	}
	return wasmkeeper.NewMessageHandlerChain(
		NewSDKMessageHandler(router, encoders, treasuryKeeper, accountKeeper, bankKeeper, tax2gaskeeper),
		wasmkeeper.NewIBCRawPacketHandler(ics4Wrapper, channelKeeper, capabilityKeeper),
		wasmkeeper.NewBurnCoinMessageHandler(bankKeeper),
	)
}

func NewSDKMessageHandler(router MessageRouter, encoders msgEncoder, treasuryKeeper treasurykeeper.Keeper, accountKeeper authkeeper.AccountKeeper, bankKeeper bankKeeper.Keeper, tax2gaskeeper tax2gaskeeper.Keeper) SDKMessageHandler {
	return SDKMessageHandler{
		router:         router,
		encoders:       encoders,
		treasuryKeeper: treasuryKeeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		tax2gaskeeper:  tax2gaskeeper,
	}
}

func (h SDKMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
	sdkMsgs, err := h.encoders.Encode(ctx, contractAddr, contractIBCPortID, msg)
	if err != nil {
		return nil, nil, err
	}

	for _, sdkMsg := range sdkMsgs {
		if h.tax2gaskeeper.IsEnabled(ctx) {
			burnTaxRate := h.tax2gaskeeper.GetBurnTaxRate(ctx)
			taxes := FilterMsgAndComputeTax(ctx, h.treasuryKeeper, burnTaxRate, sdkMsg)
			if !taxes.IsZero() {
				eventManager := sdk.NewEventManager()

				taxGas, err := tax2gasutils.ComputeGas(h.tax2gaskeeper.GetGasPrices(ctx), taxes)
				if err != nil {
					return nil, nil, err
				}
				ctx.TaxGasMeter().ConsumeGas(taxGas, "tax gas")

				events = eventManager.Events()
			}
		}

		res, err := h.handleSdkMessage(ctx, contractAddr, sdkMsg)
		if err != nil {
			return nil, nil, err
		}
		// append data
		data = append(data, res.Data)
		// append events
		sdkEvents := make([]sdk.Event, len(res.Events))
		for i := range res.Events {
			sdkEvents[i] = sdk.Event(res.Events[i])
		}
		events = append(events, sdkEvents...)
	}
	return events, data, nil
}

func (h SDKMessageHandler) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) (*sdk.Result, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	// make sure this account can send it
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
			return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
		}
	}

	// find the handler and execute it
	if handler := h.router.Handler(msg); handler != nil {
		// ADR 031 request type routing
		msgResult, err := handler(ctx, msg)
		return msgResult, err
	}
	// legacy sdk.Msg routing
	// Assuming that the app developer has migrated all their Msgs to
	// proto messages and has registered all `Msg services`, then this
	// path should never be called, because all those Msgs should be
	// registered within the `msgServiceRouter` already.
	return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
}

// FilterMsgAndComputeTax computes the stability tax on messages.
func FilterMsgAndComputeTax(ctx sdk.Context, tk tax2gastypes.TreasuryKeeper, burnTaxRate sdk.Dec, msgs ...sdk.Msg) sdk.Coins {
	taxes := sdk.Coins{}

	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *banktypes.MsgSend:
			if !tk.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
				taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, msg.Amount)...)
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
					taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, input.Coins)...)
				}
			}

		case *marketexported.MsgSwapSend:
			taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, sdk.NewCoins(msg.OfferCoin))...)

		case *wasmtypes.MsgInstantiateContract:
			taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)...)

		case *wasmtypes.MsgInstantiateContract2:
			taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)...)

		case *wasmtypes.MsgExecuteContract:
			if !tk.HasBurnTaxExemptionContract(ctx, msg.Contract) {
				taxes = taxes.Add(tax2gasutils.ComputeTax(burnTaxRate, msg.Funds)...)
			}

		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err == nil {
				taxes = taxes.Add(FilterMsgAndComputeTax(ctx, tk, burnTaxRate, messages...)...)
			}
		}
	}

	return taxes
}
