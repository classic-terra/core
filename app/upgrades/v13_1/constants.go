//nolint:revive
package v13_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v13_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV131UpgradeHandler,
}
