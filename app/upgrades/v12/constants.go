package v12

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v3/x/taxexemption/types"
	store "cosmossdk.io/store/types"
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
