package v81

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	tax2gasTypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v8_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV8_1UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tax2gasTypes.StoreKey,
		},
	},
}
