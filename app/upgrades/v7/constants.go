package v7

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	ibc_hooks_types "github.com/classic-terra/core/v2/x/ibc-hooks/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v7"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			ibc_hooks_types.StoreKey,
		},
	},
}
