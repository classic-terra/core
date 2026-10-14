//nolint:revive
package ustc_staking

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func CreateUSTCStakingUpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	appKeepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx, write := sdk.UnwrapSDKContext(ctx).CacheContext()
		if existingVersion, alreadyExists := fromVM[ustcstakingtypes.ModuleName]; alreadyExists {
			if existingVersion != 1 {
				return nil, fmt.Errorf("unsupported existing USTC staking version %d", existingVersion)
			}
		} else {
			if err := normalizePoolAccounts(sdkCtx, appKeepers); err != nil {
				return nil, err
			}
		}
		versionMap, err := mm.RunMigrations(sdkCtx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		for _, moduleName := range []string{ustcstakingtypes.PrincipalPoolName, ustcstakingtypes.RewardPoolName} {
			_, permissions := appKeepers.AccountKeeper.GetModuleAddressAndPermissions(moduleName)
			account, ok := appKeepers.AccountKeeper.GetAccount(sdkCtx, authtypes.NewModuleAddress(moduleName)).(sdk.ModuleAccountI)
			if !ok {
				return nil, fmt.Errorf("%s account is not a module account", moduleName)
			}
			if account.GetName() != moduleName || len(account.GetPermissions()) != 0 || len(permissions) != 0 {
				return nil, fmt.Errorf("%s module account has unexpected permissions %v", moduleName, permissions)
			}
		}
		if err := appKeepers.UstcStakingKeeper.ValidateState(sdkCtx); err != nil {
			return nil, err
		}
		write()
		return versionMap, nil
	}
}
