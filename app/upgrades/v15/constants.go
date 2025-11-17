package v15

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v15"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV15UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{}, // no store upgrades
}
