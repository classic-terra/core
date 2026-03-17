package v8_1

import (
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v8_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV81UpgradeHandler,
}
