package v4_1

import (
	"github.com/classic-terra/core/v2/app/upgrades"
)

const UpgradeName = "v4.1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV4_1UpgradeHandler,
}
