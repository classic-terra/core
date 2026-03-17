package v71

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v7_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7_1UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
