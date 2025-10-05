package v3

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "cosmossdk.io/store/types"
)

const UpgradeName = "v3"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV3UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
