package forks

import (
	"fmt"

	"github.com/classic-terra/core/v2/app/keepers"
	core "github.com/classic-terra/core/v2/types"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

func runForkLogicSwapDisable(ctx sdk.Context, keppers *keepers.AppKeepers, _ *module.Manager) {
	if ctx.ChainID() == core.ColumbusChainID {
		// Make min spread to 100% to disable swap
		params := keppers.MarketKeeper.GetParams(ctx)
		params.MinStabilitySpread = sdk.OneDec()
		keppers.MarketKeeper.SetParams(ctx, params)

		// Disable IBC Channels
		channelIDs := []string{
			"channel-1",  // Osmosis
			"channel-49", // Crescent
			"channel-20", // Juno
		}
		for _, channelID := range channelIDs {
			channel, found := keppers.IBCKeeper.ChannelKeeper.GetChannel(ctx, ibctransfertypes.PortID, channelID)
			if !found {
				panic(fmt.Sprintf("%s not found", channelID))
			}

			channel.State = ibcchanneltypes.CLOSED
			keppers.IBCKeeper.ChannelKeeper.SetChannel(ctx, ibctransfertypes.PortID, channelID, channel)
		}
	}
}

func runForkLogicIbcEnable(ctx sdk.Context, keppers *keepers.AppKeepers, _ *module.Manager) {
	if ctx.ChainID() == core.ColumbusChainID {
		// Enable IBC Channels
		channelIDs := []string{
			"channel-1",  // Osmosis
			"channel-49", // Crescent
			"channel-20", // Juno
		}
		for _, channelID := range channelIDs {
			channel, found := keppers.IBCKeeper.ChannelKeeper.GetChannel(ctx, ibctransfertypes.PortID, channelID)
			if !found {
				panic(fmt.Sprintf("%s not found", channelID))
			}

			channel.State = ibcchanneltypes.OPEN
			keppers.IBCKeeper.ChannelKeeper.SetChannel(ctx, ibctransfertypes.PortID, channelID, channel)
		}
	}
}

func runForkLogicVersionMapEnable(ctx sdk.Context, keppers *keepers.AppKeepers, mm *module.Manager) {
	// trigger SetModuleVersionMap in upgrade keeper at the VersionMapEnableHeight
	if ctx.ChainID() == core.ColumbusChainID {
		keppers.UpgradeKeeper.SetModuleVersionMap(ctx, mm.GetVersionMap())
	}
}

func runForkLogicBlacklist800M(ctx sdk.Context, keepers *keepers.AppKeepers, mm *module.Manager) {

	var freeze treasurytypes.FreezeList

	if ctx.ChainID() == core.ColumbusChainID {

		addr, err := sdk.AccAddressFromBech32("terra1qyw695vaxj7jl6s4u564c6xkfe59kercg0h88w")
		if err != nil {
			ctx.Logger().Error("Could not unmarshal blacklist address - ignoring.")
			return
		}
		freeze.Add(addr.String())

		keepers.TreasuryKeeper.SetFreezeAddrs(ctx, freeze)

	}

}

func runForkLogicBlacklist800MRebel(ctx sdk.Context, keepers *keepers.AppKeepers, mm *module.Manager) {

	var freeze treasurytypes.FreezeList

	if ctx.ChainID() == core.RebelChainID {

		addr, err := sdk.AccAddressFromBech32("terra10zn3xx8nhvtdynux5tzjer23q2qpg0tz7xamut")
		if err != nil {
			ctx.Logger().Error("Could not unmarshal blacklist address - ignoring.")
			return
		}
		freeze.Add(addr.String())

		addr, err = sdk.AccAddressFromBech32("terra1njlydj87f05jmzdt9wmam0z28dlrc97qr6twqn")
		if err != nil {
			ctx.Logger().Error("Could not unmarshal blacklist address - ignoring.")
			return
		}
		freeze.Add(addr.String())

		keepers.TreasuryKeeper.SetFreezeAddrs(ctx, freeze)

	}
}
