package v7

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
	forwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v6/router/types"
)

const UpgradeName = "v7_testnet"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7TestnetUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Deleted: []string{
			forwardtypes.StoreKey,
		},
	},
}
