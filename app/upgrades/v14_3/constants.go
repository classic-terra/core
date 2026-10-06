//nolint:revive
package v14_3

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v14_3"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV143UpgradeHandler,
	// Add new stores introduced since the last upgrade here. If there are
	// no new stores for this upgrade, leave this empty.
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
