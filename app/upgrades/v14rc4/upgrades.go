package v14rc4

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	core "github.com/classic-terra/core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV14RC4UpgradeHandler creates the upgrade handler for v14rc4.
// This upgrade triggers the wasm module migration from consensus version 4 to 5,
// which fixes the ContractInfo protobuf field order swap between wasmd v0.61.4 and v0.61.5.
// The migration only runs on testnet (rebel-2); on mainnet the handler is a no-op.
func CreateV14RC4UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		if sdkCtx.ChainID() != core.RebelChainID {
			fmt.Printf("[v14rc4] skipping wasm migration on chain %s (only runs on %s)\n", sdkCtx.ChainID(), core.RebelChainID)
			sdkCtx.Logger().Info("v14rc4: skipping wasm migration on chain", "chainID", sdkCtx.ChainID())
			return fromVM, nil
		}

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
