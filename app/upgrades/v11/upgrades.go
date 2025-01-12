//nolint:revive
package v11

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func justBurnItAlready(ctx sdk.Context, bank bankeeper.Keeper, targetAddr sdk.AccAddress) {
	ustc := bank.GetBalance(ctx, targetAddr, "uusd")
	if ustc.IsZero() {
		return
	}
	bank.SendCoinsFromAccountToModule(ctx, targetAddr, banktypes.ModuleName, sdk.NewCoins(ustc))
	bank.BurnCoins(ctx, banktypes.ModuleName, sdk.NewCoins(ustc))
}

func CreateV11UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		var targetAddr sdk.AccAddress
		if ctx.ChainID() == "rebel-2" {
			targetAddr = sdk.MustAccAddressFromBech32(LFGWalletTestnet)
		} else if ctx.ChainID() == "columbus-5" {
			targetAddr = sdk.MustAccAddressFromBech32(LFGWallet)
		} else {
			return mm.RunMigrations(ctx, cfg, fromVM)
		}
		justBurnItAlready(ctx, keepers.BankKeeper, targetAddr)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
