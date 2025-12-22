package v6

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	dyncommtypes "github.com/classic-terra/core/v4/x/dyncomm/types"
)

const UpgradeName = "v6"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV6UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			dyncommtypes.StoreKey,
		},
	},
}
