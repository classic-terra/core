package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/tax2gas/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return Keeper{cdc: cdc, storeKey: storeKey, authority: authority}
}

// InitGenesis initializes the tax2gas module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the tax2gas module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the x/tax2gas module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) IsEnabled(ctx sdk.Context) bool {
	return k.GetParams(ctx).Enabled
}

func (k Keeper) GetGasPrices(ctx sdk.Context) sdk.DecCoins {
	return k.GetParams(ctx).GasPrices.Sort()
}

// GetBypassMinFeeMsgTypes gets the tax2gas module's BypassMinFeeMsgTypes.
func (k Keeper) GetBypassMinFeeMsgTypes(ctx sdk.Context) []string {
	return k.GetParams(ctx).BypassMinFeeMsgTypes
}

// GetBypassMinFeeMsgTypes gets the tax2gas module's BypassMinFeeMsgTypes.
func (k Keeper) GetMaxTotalBypassMinFeeMsgGasUsage(ctx sdk.Context) uint64 {
	return k.GetParams(ctx).MaxTotalBypassMinFeeMsgGasUsage
}
