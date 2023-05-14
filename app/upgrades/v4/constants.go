package v3

import (
	"github.com/classic-terra/core/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v4"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV4UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
