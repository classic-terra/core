package v7

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
	forwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v6/router/types"
	ibc_hooks_types "github.com/terra-money/core/v2/x/ibc-hooks/types"
)

const UpgradeName = "v7"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			ibc_hooks_types.StoreKey,
			forwardtypes.StoreKey,
		},
	},
}
