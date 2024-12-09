//nolint:revive
package v10_2

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v10_2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV102UpgradeHandler,
}
