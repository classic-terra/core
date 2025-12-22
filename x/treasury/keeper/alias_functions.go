package keeper

import (
	"github.com/classic-terra/core/v4/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// GetTreasuryModuleAccount returns treasury ModuleAccount
func (k Keeper) GetTreasuryModuleAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// GetBurnModuleAccount returns burn ModuleAccount
func (k Keeper) GetBurnModuleAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, types.BurnModuleName)
}
