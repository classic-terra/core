//nolint:revive
package v14_4

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v14_4"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV144UpgradeHandler,
	// Add new stores introduced since the last upgrade here. If there are
	// no new stores for this upgrade, leave this empty.
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
