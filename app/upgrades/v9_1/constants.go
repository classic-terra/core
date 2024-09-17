package v91

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v9_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV91UpgradeHandler,
}
