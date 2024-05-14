package v71

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v7_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7_1UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
