//nolint:revive
package ustc_staking

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
)

const UpgradeName = "v14_5"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUSTCStakingUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{ustcstakingtypes.StoreKey},
	},
}
