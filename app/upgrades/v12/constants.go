package v12

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v4/x/taxexemption/types"
)

const UpgradeName = "v12"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV12UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			taxexemptiontypes.StoreKey,
		},
	},
}
