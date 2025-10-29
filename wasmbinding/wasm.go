package wasmbinding

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	customwasm "github.com/classic-terra/core/v3/custom/wasm"
	marketkeeper "github.com/classic-terra/core/v3/x/market/keeper"
	oraclekeeper "github.com/classic-terra/core/v3/x/oracle/keeper"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
)

func RegisterCustomPlugins(
	marketKeeper *marketkeeper.Keeper,
	oracleKeeper *oraclekeeper.Keeper,
	treasuryKeeper *treasurykeeper.Keeper,
) []wasmkeeper.Option {
	wasmQueryPlugin := NewQueryPlugin(
		marketKeeper,
		oracleKeeper,
		treasuryKeeper,
	)

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})
	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(marketKeeper),
	)

	return []wasmkeeper.Option{
		queryPluginOpt,
		messengerDecoratorOpt,
	}
}

func RegisterStargateQueries(queryRouter baseapp.GRPCQueryRouter, codec codec.Codec) []wasmkeeper.Option {
	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: StargateQuerier(queryRouter, codec),
	})

	return []wasmkeeper.Option{
		queryPluginOpt,
	}
}

// RegisterLegacyQueryHandler wraps the wasm query handler with legacy store support for historical queries
func RegisterLegacyQueryHandler(storeKey storetypes.StoreKey) wasmkeeper.Option {
	return wasmkeeper.WithQueryHandlerDecorator(func(next wasmkeeper.WasmVMQueryHandler) wasmkeeper.WasmVMQueryHandler {
		return customwasm.NewLegacyQueryHandler(next, storeKey)
	})
}
