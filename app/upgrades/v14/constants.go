package v14

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v14"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV14UpgradeHandler,
	// Add new stores introduced since the last upgrade here. If there are
	// no new stores for this upgrade, leave this empty.
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
