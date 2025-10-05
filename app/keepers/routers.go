package keepers

import (
	icacontroller "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	transfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/classic-terra/core/v3/x/treasury"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v10"
)

func (appKeepers *AppKeepers) newGovRouter() govv1beta1.Router {
	govRouter := govv1beta1.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(appKeepers.ParamsKeeper)).
		// TODO: check this // AddRoute(upgradetypes.RouterKey, upgradekeeper.NewSoftwareUpgradeProposalHandler(appKeepers.UpgradeKeeper)).
		// TODO: check this // AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(appKeepers.IBCKeeper.ClientKeeper)).
		AddRoute(treasurytypes.RouterKey, treasury.NewProposalHandler(appKeepers.TreasuryKeeper))

	return govRouter
}

func (appKeepers *AppKeepers) newIBCRouter() *porttypes.Router {
	// Create Transfer Stack
	var transferStack porttypes.IBCModule
	var transferHookStack porttypes.IBCModule

	transferStack = transfer.NewIBCModule(appKeepers.TransferKeeper)
	transferHookStack = ibchooks.NewIBCMiddleware(transferStack, appKeepers.IBCHooksWrapper)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> channel.SendPacket
	var icaControllerStack porttypes.IBCModule
	// integration point for custom authentication modules
	// see https://medium.com/the-interchain-foundation/ibc-go-v6-changes-to-interchain-accounts-and-how-it-impacts-your-chain-806c185300d7
	var noAuthzModule porttypes.IBCModule
	_ = noAuthzModule // not used in v10 single-arg middleware
	icaControllerStack = icacontroller.NewIBCMiddleware(appKeepers.ICAControllerKeeper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	var icaHostStack porttypes.IBCModule
	icaHostStack = icahost.NewIBCModule(appKeepers.ICAHostKeeper)

	// Create wasm ibc Stack
	var wasmStack porttypes.IBCModule
	wasmStack = wasm.NewIBCHandler(
		appKeepers.WasmKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.TransferKeeper,
		appKeepers.IBCHooksWrapper,
	)

	ibcRouter := porttypes.NewRouter()
	ibcRouter.
		AddRoute(ibctransfertypes.ModuleName, transferHookStack).
		AddRoute(wasmtypes.ModuleName, wasmStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack)

	return ibcRouter
}
