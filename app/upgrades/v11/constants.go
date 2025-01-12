//nolint:revive
package v11

import (
	"github.com/classic-terra/core/v3/app/upgrades"
	store "github.com/cosmos/cosmos-sdk/store/types"
)

const UpgradeName = "v11"
const LFGWallet = "terra1gr0xesnseevzt3h4nxr64sh5gk4dwrwgszx3nw"
const LFGWalletTestnet = "terra1gr0xesnseevzt3h4nxr64sh5gk4dwrwgszx3nw"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV11UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
