package v11_2

import (
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v11_2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV112UpgradeHandler,
}
