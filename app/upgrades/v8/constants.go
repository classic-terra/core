package v8

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistpyes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const UpgradeName = "v8"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV8UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			consensustypes.ModuleName,
			crisistpyes.ModuleName,
		},
	},
}
