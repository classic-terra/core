package v5

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v3/app/upgrades"

	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

const UpgradeName = "v5"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV5UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			icacontrollertypes.StoreKey,
		},
	},
}
