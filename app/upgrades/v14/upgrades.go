package v14

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// CreateV14UpgradeHandler wires module migrations for v14.
// Add any one-off migration logic here before/after RunMigrations if needed.
func CreateV14UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		keepers.IBCKeeper.ClientKeeper.SetParams(sdk.UnwrapSDKContext(ctx), clienttypes.DefaultParams())

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
