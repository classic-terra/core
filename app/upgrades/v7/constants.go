package v7

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
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
