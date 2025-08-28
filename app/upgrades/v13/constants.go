//nolint:revive
package v13

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v13"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV13UpgradeHandler,
}

const (
	WasmMigrationMarker = "v13_wasm_migrated"
)
