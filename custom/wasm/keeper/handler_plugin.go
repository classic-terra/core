package keeper

import (
	"reflect"
	"strings"

	treasurykeeper "github.com/classic-terra/core/v2/x/treasury/keeper"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	classictaxkeeper "github.com/classic-terra/core/v2/x/classictax/keeper"
	classictaxtypes "github.com/classic-terra/core/v2/x/classictax/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/authz"
	bankKeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	marketexported "github.com/classic-terra/core/v2/x/market/exported"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
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
	router           MessageRouter
	encoders         msgEncoder
	treasuryKeeper   treasurykeeper.Keeper
	accountKeeper    authkeeper.AccountKeeper
	bankKeeper       bankKeeper.Keeper
	classictaxKeeper classictaxkeeper.Keeper
}

func NewMessageHandler(
	router MessageRouter,
	channelKeeper wasmtypes.ChannelKeeper,
	capabilityKeeper wasmtypes.CapabilityKeeper,
	bankKeeper bankKeeper.Keeper,
	treasuryKeeper treasurykeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	unpacker codectypes.AnyUnpacker,
	portSource wasmtypes.ICS20TransferPortSource,
	ctk classictaxkeeper.Keeper,
	customEncoders ...*wasmkeeper.MessageEncoders,
) wasmkeeper.Messenger {
	encoders := wasmkeeper.DefaultEncoders(unpacker, portSource)
	for _, e := range customEncoders {
		encoders = encoders.Merge(e)
	}
	return wasmkeeper.NewMessageHandlerChain(
		NewSDKMessageHandler(router, encoders, treasuryKeeper, accountKeeper, bankKeeper, ctk),
		wasmkeeper.NewIBCRawPacketHandler(channelKeeper, capabilityKeeper),
		wasmkeeper.NewBurnCoinMessageHandler(bankKeeper),
	)
}

func NewSDKMessageHandler(router MessageRouter, encoders msgEncoder, treasuryKeeper treasurykeeper.Keeper, accountKeeper authkeeper.AccountKeeper, bankKeeper bankKeeper.Keeper, ctk classictaxkeeper.Keeper) SDKMessageHandler {
	return SDKMessageHandler{
		router:           router,
		encoders:         encoders,
		treasuryKeeper:   treasuryKeeper,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,
		classictaxKeeper: ctk,
	}
}

func DeductTaxFromCoin(ctx sdk.Context, ctk classictaxkeeper.Keeper, coin sdk.Coin) (newCoin sdk.Coin, err error) {
	taxes := ctk.ComputeBurnTax(ctx, sdk.NewCoins(coin))

	if !taxes.IsZero() {
		// Deduct tax from coins
		if found, taxCoin := taxes.Find(coin.Denom); found {
			if newCoin, err = coin.SafeSub(taxCoin); err != nil {
				return newCoin, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
			}
		}
	}

	return newCoin, nil
}

func DeductTaxFromCoins(ctx sdk.Context, ctk classictaxkeeper.Keeper, coins sdk.Coins) (newCoins sdk.Coins, err error) {
	taxes := ctk.ComputeBurnTax(ctx, coins)

	var (
		neg bool
	)

	if !taxes.IsZero() {
		// Deduct tax from coins
		if newCoins, neg = coins.SafeSub(taxes...); neg {
			return newCoins, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}
	}

	return newCoins, nil
}

func DeductTaxFromMessage(ctx sdk.Context, ctk classictaxkeeper.Keeper, msg sdk.Msg) error {
	var (
		err error
	)

	taxableMsgTypes := ctk.GetTaxableMsgTypes(ctx)

	taxable := false
	for _, msgType := range taxableMsgTypes {
		// get the type string (e.g. types.MsgSend)
		// TODO check if this needs to be improved
		tp := strings.TrimLeft(reflect.TypeOf(msg).String(), "*")
		ctk.Logger(ctx).Info("Check taxable", "msg", tp, "msgType", msgType)
		if tp == msgType {
			taxable = true
			ctk.Logger(ctx).Info("Found taxable message type")
			break
		}
	}

	if !taxable {
		return nil
	}

	switch msg := msg.(type) {
	case *banktypes.MsgSend:
		if msg.Amount, err = DeductTaxFromCoins(ctx, ctk, msg.Amount); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}

	case *banktypes.MsgMultiSend:
		for _, input := range msg.Inputs {
			if input.Coins, err = DeductTaxFromCoins(ctx, ctk, input.Coins); err != nil {
				return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
			}
		}

	case *marketexported.MsgSwap:
		if msg.OfferCoin, err = DeductTaxFromCoin(ctx, ctk, msg.OfferCoin); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}
	case *marketexported.MsgSwapSend:
		if msg.OfferCoin, err = DeductTaxFromCoin(ctx, ctk, msg.OfferCoin); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}

	case *wasm.MsgInstantiateContract:
		if msg.Funds, err = DeductTaxFromCoins(ctx, ctk, msg.Funds); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}
	case *wasm.MsgInstantiateContract2:
		if msg.Funds, err = DeductTaxFromCoins(ctx, ctk, msg.Funds); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}

	case *wasm.MsgExecuteContract:
		if msg.Funds, err = DeductTaxFromCoins(ctx, ctk, msg.Funds); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}

	case *stakingtypes.MsgDelegate:
		if msg.Amount, err = DeductTaxFromCoin(ctx, ctk, msg.Amount); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}
	case *stakingtypes.MsgUndelegate:
		if msg.Amount, err = DeductTaxFromCoin(ctx, ctk, msg.Amount); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "insufficient funds to pay tax")
		}
	case *authz.MsgExec:
		messages, err := msg.GetMessages()
		if err != nil {
			DeductTaxFromMessages(ctx, ctk, messages...)
		}
	}

	return nil
}

func DeductTaxFromMessages(ctx sdk.Context, ctk classictaxkeeper.Keeper, msgs ...sdk.Msg) error {
	for _, sdkMsg := range msgs {
		switch msg := sdkMsg.(type) {
		case *authz.MsgExec:
			messages, err := msg.GetMessages()
			if err != nil {
				DeductTaxFromMessages(ctx, ctk, messages...)
			}

		default:
			DeductTaxFromMessage(ctx, ctk, sdkMsg)
		}
	}

	return nil
}

func (h SDKMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
	sdkMsgs, err := h.encoders.Encode(ctx, contractAddr, contractIBCPortID, msg)
	if err != nil {
		return nil, nil, err
	}

	for _, sdkMsg := range sdkMsgs {
		// Charge tax on result msg
		stabilityTaxes := classictaxkeeper.FilterMsgAndComputeStabilityTax(ctx, h.treasuryKeeper, sdkMsg)
		if !stabilityTaxes.IsZero() {
			eventManager := sdk.NewEventManager()
			contractAcc := h.accountKeeper.GetAccount(ctx, contractAddr)
			if err := cosmosante.DeductFees(h.bankKeeper, ctx.WithEventManager(eventManager), contractAcc, stabilityTaxes); err != nil {
				return nil, nil, err
			}

			events = eventManager.Events()
		}

		taxes, _ := h.classictaxKeeper.GetTaxCoins(ctx, sdkMsg)
		// Deduct tax from contract account and subtract from amount that can be sent
		if !taxes.IsZero() {
			h.classictaxKeeper.Logger(ctx).Info("Deduct tax from contract account", "taxes", taxes)

			eventManager := sdk.NewEventManager()
			contractAcc := h.accountKeeper.GetAccount(ctx, contractAddr)
			if err := classictaxkeeper.DeductFees(h.bankKeeper, ctx.WithEventManager(eventManager), contractAcc, taxes, false); err != nil {
				return nil, nil, err
			}

			if err := h.classictaxKeeper.BurnTaxSplit(ctx.WithEventManager(eventManager), taxes); err != nil {
				return nil, nil, err
			}

			taxEvents := sdk.Events{
				sdk.NewEvent(
					sdk.EventTypeTx,
					sdk.NewAttribute(classictaxtypes.AttributeKeyTax, taxes.String()),
					sdk.NewAttribute(sdk.AttributeKeyFeePayer, contractAddr.String()),
				),
			}
			eventManager.EmitEvents(taxEvents)
			events = append(events, eventManager.Events()...)

			DeductTaxFromMessage(ctx, h.classictaxKeeper, sdkMsg)
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
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
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
	return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
}
