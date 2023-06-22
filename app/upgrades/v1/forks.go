package v1

import (
	"fmt"

	"github.com/classic-terra/core/v2/app/keepers"
	core "github.com/classic-terra/core/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
)

func RunForkLogic_1_0_0(ctx sdk.Context, keppers *keepers.AppKeepers, _ *module.Manager) {
	if ctx.ChainID() == core.ColumbusChainID && ctx.BlockHeight() == core.SwapDisableForkHeight { // Make min spread to one to disable swap
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

func RunForkLogic_1_0_5(ctx sdk.Context, keppers *keepers.AppKeepers, _ *module.Manager) {
	if ctx.ChainID() == core.ColumbusChainID && ctx.BlockHeight() == core.SwapEnableForkHeight { // Re-enable IBCs
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

func RunForkLogic_1_1_0(ctx sdk.Context, keppers *keepers.AppKeepers, mm *module.Manager) {
	// trigger SetModuleVersionMap in upgrade keeper at the VersionMapEnableHeight
	if ctx.ChainID() == core.ColumbusChainID && ctx.BlockHeight() == core.VersionMapEnableHeight {
		keppers.UpgradeKeeper.SetModuleVersionMap(ctx, mm.GetVersionMap())
	}
}
