//nolint:revive
package v8_2

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v8_2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV82UpgradeHandler,
}
