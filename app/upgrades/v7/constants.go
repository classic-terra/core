package v7

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/types"
)

const UpgradeName = "v7"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV7UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			ibchookstypes.StoreKey,
		},
	},
}
