//nolint:revive
package v8_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v8_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV81UpgradeHandler,
}
