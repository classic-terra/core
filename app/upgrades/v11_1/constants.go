//nolint:revive
package v11_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v11_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV111UpgradeHandler,
}
