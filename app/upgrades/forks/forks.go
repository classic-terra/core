package forks

import (
	"fmt"

	"github.com/classic-terra/core/v2/app/keepers"
	core "github.com/classic-terra/core/v2/types"
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

func runForkLogicCorrectAccountSequence(ctx sdk.Context, keppers *keepers.AppKeepers, mm *module.Manager) {
	if ctx.ChainID() == core.ColumbusChainID {
		// Correct account sequence
		for _, acc := range affectedAccounts {
			account := keppers.AccountKeeper.GetAccount(ctx, sdk.MustAccAddressFromBech32(acc.Address))
			// check if local account sequence is equal to targeted account sequence
			if account.GetSequence() == acc.Sequence {
				ctx.Logger().Info(fmt.Sprintf("Account %s sequence is already correct", acc.Address))
				continue
			}

			account.SetSequence(acc.Sequence)
			keppers.AccountKeeper.SetAccount(ctx, account)
			ctx.Logger().Info(fmt.Sprintf("Account %s sequence is corrected to %d", acc.Address, acc.Sequence))
		}
	}
}
