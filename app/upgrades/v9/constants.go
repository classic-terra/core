package v9

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v3/x/taxexemption/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v9"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV9UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			taxexemptiontypes.StoreKey,
		},
	},
}
