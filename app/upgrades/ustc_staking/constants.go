//nolint:revive
package ustc_staking

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
)

const UpgradeName = "v14_5"

// ExistingStoreUpgradeName is the alternative handler name for a chain whose
// app store already contains x/ustcstaking v1. It deliberately has no store
// addition. A network must select exactly one of these upgrade plans after
// checking its persisted module version map.
const ExistingStoreUpgradeName = "v14_5_ustcstaking_existing"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUSTCStakingUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{ustcstakingtypes.StoreKey},
	},
}

// ExistingStoreUpgrade migrates an already-mounted v1 USTC staking store.
// It is an alternative to Upgrade, not a second sequential upgrade.
var ExistingStoreUpgrade = upgrades.Upgrade{
	UpgradeName:          ExistingStoreUpgradeName,
	CreateUpgradeHandler: CreateUSTCStakingUpgradeHandler,
}
