//nolint:revive
package v8_4

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v8_4"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV84UpgradeHandler,
}
