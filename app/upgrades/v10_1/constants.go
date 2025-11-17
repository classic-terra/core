package v10_1

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v3/app/upgrades"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax/types"
)

const UpgradeName = "v10_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV101UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tax2gastypes.ModuleName,
		},
	},
}
