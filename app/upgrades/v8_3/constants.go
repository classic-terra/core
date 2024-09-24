//nolint:revive
package v8_3

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v8_3"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV83UpgradeHandler,
}
