package v15

import (
	store "cosmossdk.io/store/types"

	"github.com/classic-terra/core/v4/app/upgrades"

	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
)

const UpgradeName = "v15"

// Upgrade wires the Packet Forward Middleware (PFM) into Terra Classic.
// It only adds the PFM module store; PFM's default genesis (FeePercentage = 0)
// is initialized automatically by RunMigrations because the module is new.
var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV15UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{packetforwardtypes.StoreKey},
		Deleted: []string{},
	},
}
