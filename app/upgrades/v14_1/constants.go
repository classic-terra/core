//nolint:revive
package v14_1

import (
	"github.com/classic-terra/core/v4/app/upgrades"
)

const UpgradeName = "v14_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV141UpgradeHandler,
}
