package v11_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v11"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV111UpgradeHandler,
}
