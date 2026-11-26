//nolint:revive
package v14_6

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV146UpgradeHandler wires module migrations for v14_6 and opens wasm
// code upload to everybody.
//
// The binary links the patched wasmvm (3.0.9-tcsec.2), which rejects modules
// that could stall the singlepass compiler on the checked StoreCode path. With
// that in place, the governance restriction on code upload is no longer
// needed.
func CreateV146UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versionMap, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}

		if err := AllowCodeUploadForEverybody(sdk.UnwrapSDKContext(ctx), k); err != nil {
			return nil, err
		}

		return versionMap, nil
	}
}

// AllowCodeUploadForEverybody sets the wasm code_upload_access parameter to
// Everybody. All other wasm params are left unchanged.
func AllowCodeUploadForEverybody(ctx sdk.Context, k *keepers.AppKeepers) error {
	params := k.WasmKeeper.GetParams(ctx)
	previous := params.CodeUploadAccess
	params.CodeUploadAccess = wasmtypes.AllowEverybody

	if err := k.WasmKeeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("v14_6: set wasm code upload access: %w", err)
	}

	ctx.Logger().Info("v14_6: set wasm code upload access to everybody",
		"previous", previous.Permission.String(), "previous_addresses", previous.Addresses)
	return nil
}
