package v91

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"

	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
)

const UpgradeName = "v9_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV91UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tax2gastypes.ModuleName,
		},
	},
}
