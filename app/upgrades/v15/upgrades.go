package v15

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV15UpgradeHandler adds the Packet Forward Middleware module.
//
// No bespoke state migration is required: because packetforward is a newly
// registered module (absent from the incoming module VersionMap), RunMigrations
// runs its InitGenesis with DefaultGenesis, which sets the default PFM params
// (FeePercentage = 0 — forwarding is free, no community-pool skim yet). The
// FeePercentage can be raised later via the module's MsgUpdateParams.
func CreateV15UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
