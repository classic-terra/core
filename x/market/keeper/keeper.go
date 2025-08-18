package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/types"
)

// Keeper of the market store
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramstypes.Subspace

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
	OracleKeeper  types.OracleKeeper

	// allowedSwapDenoms contains denoms that are allowed to be swapped with uluna
	// This is kept in-memory (not a chain param) so tests can differ from live defaults.
	allowedSwapDenoms map[string]bool
}

// NewKeeper constructs a new keeper for oracle
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramstore paramstypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
) Keeper {
	// ensure market module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// set KeyTable if it has not already been set
	if !paramstore.HasKeyTable() {
		paramstore = paramstore.WithKeyTable(types.ParamKeyTable())
	}

	// default allowed swap denoms: only USD for production, but we might need others for tests
	allowed := map[string]bool{
		core.MicroUSDDenom: true,
	}

	return Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		paramSpace:        paramstore,
		AccountKeeper:     accountKeeper,
		BankKeeper:        bankKeeper,
		OracleKeeper:      oracleKeeper,
		allowedSwapDenoms: allowed,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetAllowedSwapDenoms sets which denoms are allowed to be swapped with uluna.
// Note: This is intended for configuration/tests; it is not persisted.
func (k *Keeper) SetAllowedSwapDenoms(denoms []string) {
	m := make(map[string]bool, len(denoms))
	for _, d := range denoms {
		m[d] = true
	}
	k.allowedSwapDenoms = m
}

func (k Keeper) isAllowedSwapDenom(denom string) bool {
	return k.allowedSwapDenoms[denom]
}

// GetTerraPoolDelta returns the gap between the TerraPool and the TerraBasePool
func (k Keeper) GetTerraPoolDelta(ctx sdk.Context) sdk.Dec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TerraPoolDeltaKey)
	if bz == nil {
		return sdk.ZeroDec()
	}

	dp := sdk.DecProto{}
	k.cdc.MustUnmarshal(bz, &dp)
	return dp.Dec
}

// SetTerraPoolDelta updates TerraPoolDelta which is gap between the TerraPool and the BasePool
func (k Keeper) SetTerraPoolDelta(ctx sdk.Context, delta sdk.Dec) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&sdk.DecProto{Dec: delta})
	store.Set(types.TerraPoolDeltaKey, bz)
}

// ReplenishPools replenishes each pool(Terra,Luna) to BasePool
func (k Keeper) ReplenishPools(ctx sdk.Context) {
	poolDelta := k.GetTerraPoolDelta(ctx)

	poolRecoveryPeriod := int64(k.PoolRecoveryPeriod(ctx))
	poolRegressionAmt := poolDelta.QuoInt64(poolRecoveryPeriod)

	// Replenish pools towards each base pool
	// regressionAmt cannot make delta zero
	poolDelta = poolDelta.Sub(poolRegressionAmt)

	k.SetTerraPoolDelta(ctx, poolDelta)
}

// -------- Epoch processing (burn + refill) --------

func (k Keeper) getLastEpochHeight(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EpochLastHeightKey)
	if bz == nil {
		return 0
	}
	return int64(sdk.BigEndianToUint64(bz))
}

func (k Keeper) setLastEpochHeight(ctx sdk.Context, h int64) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(uint64(h))
	store.Set(types.EpochLastHeightKey, bz)
}

// ProcessEpochIfDue burns leftover pool balances and refills from the accumulator module account
// when an epoch boundary is reached.
func (k Keeper) ProcessEpochIfDue(ctx sdk.Context) {
	last := k.getLastEpochHeight(ctx)
	now := ctx.BlockHeight()
	epochLen := k.EpochLengthBlocks(ctx)
	if last != 0 && uint64(now-last) < epochLen {
		return
	}

	marketAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	accumAddr := k.AccountKeeper.GetModuleAddress(types.AccumulatorModuleName)

	// Burn all balances held by the market module account
	balances := k.BankKeeper.SpendableCoins(ctx, marketAddr)
	if !balances.Empty() {
		if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, balances); err != nil {
			// log and continue; do not panic to avoid halting the chain
			k.Logger(ctx).Error("market epoch burn failed", "err", err)
		}
		// Emit burn event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventEpochBurn,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				sdk.NewAttribute(types.AttributeKeyFromModule, types.ModuleName),
				sdk.NewAttribute(types.AttributeKeyAmount, balances.String()),
				sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", now)),
			),
		)
	}

	// Move all funds from accumulator to market module account
	accumBalances := k.BankKeeper.SpendableCoins(ctx, accumAddr)
	if !accumBalances.Empty() {
		if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.AccumulatorModuleName, types.ModuleName, accumBalances); err != nil {
			k.Logger(ctx).Error("market epoch refill failed", "err", err)
		}
		// Emit refill event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventEpochRefill,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				sdk.NewAttribute(types.AttributeKeyFromModule, types.AccumulatorModuleName),
				sdk.NewAttribute(types.AttributeKeyToModule, types.ModuleName),
				sdk.NewAttribute(types.AttributeKeyAmount, accumBalances.String()),
				sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", now)),
			),
		)
	}

	k.setLastEpochHeight(ctx, now)
}
