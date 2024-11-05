package v10

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"

	tax2gastypes "github.com/classic-terra/core/v3/x/tax/types"
)

const UpgradeName = "v10"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV10UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tax2gastypes.ModuleName,
		},
	},
}
