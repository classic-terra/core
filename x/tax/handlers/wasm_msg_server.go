package handlers

import (
	"context"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type WasmMsgServer struct {
	wasmtypes.UnimplementedMsgServer
	taxKeeper      taxkeeper.Keeper
	wasmKeeper     wasmkeeper.Keeper
	treasuryKeeper treasurykeeper.Keeper
	messageServer  wasmtypes.MsgServer
}

func NewWasmMsgServer(wasmKeeper wasmkeeper.Keeper, treasuryKeeper treasurykeeper.Keeper, taxKeeper taxkeeper.Keeper, messageServer wasmtypes.MsgServer) wasmtypes.MsgServer {
	return &WasmMsgServer{
		taxKeeper:      taxKeeper,
		wasmKeeper:     wasmKeeper,
		treasuryKeeper: treasuryKeeper,
	}
}

// ExecuteContract handles MsgExecuteContract with tax deduction
func (s *WasmMsgServer) ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	if !s.treasuryKeeper.HasBurnTaxExemptionContract(sdkCtx, msg.Contract) {
		netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
		if err != nil {
			return nil, err
		}
		msg.Funds = netFunds
	}

	return s.messageServer.ExecuteContract(ctx, msg)
}

// InstantiateContract handles MsgInstantiateContract with tax deduction
func (s *WasmMsgServer) InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
	if err != nil {
		return nil, err
	}
	msg.Funds = netFunds

	return s.messageServer.InstantiateContract(ctx, msg)
}

// InstantiateContract2 handles MsgInstantiateContract2 with tax deduction
func (s *WasmMsgServer) InstantiateContract2(ctx context.Context, msg *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
	if err != nil {
		return nil, err
	}
	msg.Funds = netFunds

	return s.messageServer.InstantiateContract2(ctx, msg)
}
