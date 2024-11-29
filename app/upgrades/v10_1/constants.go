//nolint:revive
package v10_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v10_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV101UpgradeHandler,
}
