package handlers

import (
	"context"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	taxkeeper "github.com/classic-terra/core/v3/x/tax/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// THIS FILE IS NOT USED CURRENTLY IN THE TAX MODULE
// this is due to the fact that it would cause issues with contracts
// the handling is done in the wasm module / the message handlers a contract executes

type WasmMsgServer struct {
	wasmtypes.UnimplementedMsgServer
	taxKeeper      taxkeeper.Keeper
	bankKeeper     bankkeeper.Keeper
	wasmKeeper     wasmkeeper.Keeper
	treasuryKeeper treasurykeeper.Keeper
	messageServer  wasmtypes.MsgServer
}

func NewWasmMsgServer(wasmKeeper wasmkeeper.Keeper, treasuryKeeper treasurykeeper.Keeper, taxKeeper taxkeeper.Keeper, bankKeeper bankkeeper.Keeper, _ wasmtypes.MsgServer) wasmtypes.MsgServer {
	return &WasmMsgServer{
		taxKeeper:      taxKeeper,
		wasmKeeper:     wasmKeeper,
		bankKeeper:     bankKeeper,
		treasuryKeeper: treasuryKeeper,
	}
}

/*func (s *WasmMsgServer) WithContractValue(ctx sdk.Context, address sdk.AccAddress) sdk.Context {
	balance := s.bankKeeper.GetAllBalances(ctx, address)
	return ctx.WithValue(taxtypes.ContextKeyWasmFunds, balance)
}*/

// ExecuteContract handles MsgExecuteContract with tax deduction
func (s *WasmMsgServer) ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	// This logic might lead to issues if the contract checks the received balance against the expected balance
	// We SKIP this logic for now and handle it in the wasm module
	/*
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithValue(taxtypes.ContextKeyWasmFunds, msg.Funds)
		newCtx = s.WithContractValue(newCtx, sdk.MustAccAddressFromBech32(msg.Contract))
	*/

	return s.messageServer.ExecuteContract(ctx, msg)

	/*
	   sdkCtx := sdk.UnwrapSDKContext(ctx)

	   	if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
	   		return s.messageServer.ExecuteContract(ctx, msg)
	   	}

	   sender := sdk.MustAccAddressFromBech32(msg.Sender)

	   	if !s.treasuryKeeper.HasBurnTaxExemptionContract(sdkCtx, msg.Contract) {
	   		netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
	   		if err != nil {
	   			return nil, err
	   		}
	   		msg.Funds = netFunds
	   	}

	   return s.messageServer.ExecuteContract(ctx, msg)
	*/
}

// InstantiateContract handles MsgInstantiateContract with tax deduction
func (s *WasmMsgServer) InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	// This logic might lead to issues if the contract checks the received balance against the expected balance
	// We SKIP this logic for now and handle it in the wasm module
	/*
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithValue(taxtypes.ContextKeyWasmFunds, msg.Funds).WithValue(taxtypes.ContextKeyWasmFunds, sdk.Coins{})
	*/

	return s.messageServer.InstantiateContract(ctx, msg)

	/*
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
			return s.messageServer.InstantiateContract(ctx, msg)
		}

		sender := sdk.MustAccAddressFromBech32(msg.Sender)

		netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
		if err != nil {
			return nil, err
		}
		msg.Funds = netFunds

		return s.messageServer.InstantiateContract(ctx, msg)*/
}

// InstantiateContract2 handles MsgInstantiateContract2 with tax deduction
func (s *WasmMsgServer) InstantiateContract2(ctx context.Context, msg *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	// This logic might lead to issues if the contract checks the received balance against the expected balance
	// We SKIP this logic for now and handle it in the wasm module

	/*
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithValue(taxtypes.ContextKeyWasmFunds, msg.Funds).WithValue(taxtypes.ContextKeyWasmFunds, sdk.Coins{})
	*/

	return s.messageServer.InstantiateContract2(ctx, msg)

	/*
		if !s.taxKeeper.IsReverseCharge(sdkCtx, true) {
			return s.messageServer.InstantiateContract2(ctx, msg)
		}

		sender := sdk.MustAccAddressFromBech32(msg.Sender)

		netFunds, err := s.taxKeeper.DeductTax(sdkCtx, sender, msg.Funds)
		if err != nil {
			return nil, err
		}
		msg.Funds = netFunds

		return s.messageServer.InstantiateContract2(ctx, msg)*/
}
