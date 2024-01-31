package v6

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	classictaxtypes "github.com/classic-terra/core/v2/x/classictax/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v7"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV6UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			classictaxtypes.StoreKey,
		},
	},
}
