package v61

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "cosmossdk.io/store/types"
)

const UpgradeName = "v6_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV6_1UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
