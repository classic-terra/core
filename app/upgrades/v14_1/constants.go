package v14_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v14_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV14_1UpgradeHandler,
}
