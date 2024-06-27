//nolint:revive
package v8_1

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistpyes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const UpgradeName = "v8_1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV81UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			consensustypes.ModuleName,
			crisistpyes.ModuleName,
		},
	},
}
