package v6

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v2/x/taxexemption/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v7"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			taxexemptiontypes.StoreKey,
		},
	},
}
