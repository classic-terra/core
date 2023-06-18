package v2

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/classic-terra/core/v2/app/upgrades"
)

const UpgradeName = "v2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV2UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
