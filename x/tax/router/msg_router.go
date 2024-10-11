// customrouter/msg_service_router.go
package router

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	// Import other required modules
)

type TaxMsgServiceRouter struct {
	defaultRouter  *baseapp.MsgServiceRouter
	treasuryKeeper treasurykeeper.Keeper
	bankKeeper     bankkeeper.Keeper
	distrKeeper    distributionkeeper.Keeper
	taxKeeper      taxkeeper.Keeper
}

func NewTaxMsgServiceRouter(
	defaultRouter *baseapp.MsgServiceRouter,
	treasuryKeeper treasurykeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	distrKeeper distributionkeeper.Keeper,
	taxKeeper taxkeeper.Keeper,
) *TaxMsgServiceRouter {
	return &TaxMsgServiceRouter{
		defaultRouter:  defaultRouter,
		treasuryKeeper: treasuryKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
		taxKeeper:      taxKeeper,
	}
}

func (r *TaxMsgServiceRouter) Route(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	switch msg := msg.(type) {
	case *banktypes.MsgSend:
		return r.handleMsgSend(ctx, msg)
	case *banktypes.MsgMultiSend:
		return r.handleMsgMultiSend(ctx, msg)
	case *wasmtypes.MsgExecuteContract:
		return r.handleMsgExecuteContract(ctx, msg)
	case *wasmtypes.MsgInstantiateContract:
		return r.handleMsgInstantiateContract(ctx, msg)
	case *wasmtypes.MsgInstantiateContract2:
		return r.handleMsgInstantiateContract2(ctx, msg)
	case *markettypes.MsgSwapSend:
		return r.handleMsgSwapSend(ctx, msg)
	// Handle other message types as needed
	default:
		// Delegate to the default router
		handler := r.defaultRouter.Handler(msg)
		if handler == nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unrecognized message type")
		}
		return handler(ctx, msg)
	}
}

func (r *TaxMsgServiceRouter) defaultRoute(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	handler := r.defaultRouter.Handler(msg)
	if handler == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unrecognized message type")
	}
	return handler(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgSend(ctx sdk.Context, msg *banktypes.MsgSend) (*sdk.Result, error) {
	if r.treasuryKeeper.HasBurnTaxExemptionAddress(ctx, msg.FromAddress, msg.ToAddress) {
		// Delegate to default handler
		return r.defaultRoute(ctx, msg)
	}

	fromAddr := sdk.MustAccAddressFromBech32(msg.FromAddress)
	netAmount, err := r.taxKeeper.DeductTax(ctx, fromAddr, msg.Amount)
	if err != nil {
		return nil, err
	}
	msg.Amount = netAmount

	// Delegate to default handler with modified msg
	return r.defaultRoute(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgMultiSend(ctx sdk.Context, msg *banktypes.MsgMultiSend) (*sdk.Result, error) {
	tainted := false
	for _, input := range msg.Inputs {
		if r.treasuryKeeper.HasBurnTaxExemptionAddress(ctx, input.Address) {
			tainted = true
			break
		}
	}

	if !tainted {
		for i, input := range msg.Inputs {
			fromAddr := sdk.MustAccAddressFromBech32(input.Address)
			netCoins, err := r.taxKeeper.DeductTax(ctx, fromAddr, input.Coins)
			if err != nil {
				return nil, err
			}
			msg.Inputs[i].Coins = netCoins
		}
	}

	// Delegate to default handler with modified msg
	return r.defaultRoute(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgExecuteContract(ctx sdk.Context, msg *wasmtypes.MsgExecuteContract) (*sdk.Result, error) {
	// Handle MsgExecuteContract
	return r.defaultRoute(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgInstantiateContract(ctx sdk.Context, msg *wasmtypes.MsgInstantiateContract) (*sdk.Result, error) {
	// Handle MsgInstantiateContract
	return r.defaultRoute(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgInstantiateContract2(ctx sdk.Context, msg *wasmtypes.MsgInstantiateContract2) (*sdk.Result, error) {
	// Handle MsgInstantiateContract2
	return r.defaultRoute(ctx, msg)
}

func (r *TaxMsgServiceRouter) handleMsgSwapSend(ctx sdk.Context, msg *markettypes.MsgSwapSend) (*sdk.Result, error) {
	// Handle MsgSwapSend
	return r.defaultRoute(ctx, msg)
}
