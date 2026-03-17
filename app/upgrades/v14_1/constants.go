//nolint:revive
package v14_1

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const UpgradeName = "v14_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV141UpgradeHandler,
	// Add new stores introduced since the last upgrade here. If there are
	// no new stores for this upgrade, leave this empty.
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{},
		Deleted: []string{
			crisistypes.ModuleName,
		},
		Renamed: []store.StoreRename{},
	},
}
