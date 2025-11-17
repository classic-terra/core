package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/types"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Keeper of the market store
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramstypes.Subspace

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
	OracleKeeper  types.OracleKeeper
	DistrKeeper   types.DistributionKeeper

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
	distrKeeper types.DistributionKeeper,
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
		DistrKeeper:       distrKeeper,
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
func (k Keeper) GetTerraPoolDelta(ctx sdk.Context) math.LegacyDec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TerraPoolDeltaKey)
	if bz == nil {
		return math.LegacyZeroDec()
	}

	dp := sdk.DecProto{}
	k.cdc.MustUnmarshal(bz, &dp)
	return dp.Dec
}

// SetTerraPoolDelta updates TerraPoolDelta which is gap between the TerraPool and the BasePool
func (k Keeper) SetTerraPoolDelta(ctx sdk.Context, delta math.LegacyDec) {
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

	// Set daily cap baseline to the pool balance after epoch refill
	// This baseline remains constant for the entire epoch (30 days)
	poolBalances := k.BankKeeper.SpendableCoins(ctx, marketAddr)
	for _, coin := range poolBalances {
		k.SetDailyCapBaseline(ctx, coin.Denom, coin.Amount)
	}

	// Initialize daily cap tracking for the new epoch
	k.SetDailyCapResetHeight(ctx, now)
}

// -------- Oracle tally tracking --------

// GetLastOracleTallyTime returns the timestamp of the last oracle tally
func (k Keeper) GetLastOracleTallyTime(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastOracleTallyTimeKey)
	if bz == nil {
		return 0
	}
	return int64(sdk.BigEndianToUint64(bz))
}

// SetLastOracleTallyTime stores the timestamp of the last oracle tally
func (k Keeper) SetLastOracleTallyTime(ctx sdk.Context, timestamp int64) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(uint64(timestamp))
	store.Set(types.LastOracleTallyTimeKey, bz)
}

// -------- TWAP price tracking --------

// PriceSnapshot represents a price observation at a specific height
type PriceSnapshot struct {
	Height int64
	Price  math.LegacyDec
}

// GetTWAPPrices returns the recent price snapshots for a denom
func (k Keeper) GetTWAPPrices(ctx sdk.Context, denom string) []PriceSnapshot {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetTWAPPriceKey(denom))
	if bz == nil {
		return []PriceSnapshot{}
	}

	var snapshots []PriceSnapshot
	// Encoding: each snapshot is height (8 bytes) + price length (4 bytes) + price (variable)
	offset := 0
	for offset < len(bz) {
		if offset+8 > len(bz) {
			break
		}
		height := int64(sdk.BigEndianToUint64(bz[offset : offset+8]))
		offset += 8

		// Read price length (4 bytes)
		if offset+4 > len(bz) {
			break
		}
		priceLen := int(uint32(bz[offset])<<24 | uint32(bz[offset+1])<<16 | uint32(bz[offset+2])<<8 | uint32(bz[offset+3]))
		offset += 4

		// Read price bytes
		if offset+priceLen > len(bz) {
			break
		}
		priceBytes := bz[offset : offset+priceLen]
		offset += priceLen

		var dp sdk.DecProto
		if err := k.cdc.Unmarshal(priceBytes, &dp); err == nil {
			snapshots = append(snapshots, PriceSnapshot{Height: height, Price: dp.Dec})
		}
	}
	return snapshots
}

// AddTWAPPrice adds a new price snapshot and prunes old ones
func (k Keeper) AddTWAPPrice(ctx sdk.Context, denom string, price math.LegacyDec) {
	snapshots := k.GetTWAPPrices(ctx, denom)
	currentHeight := ctx.BlockHeight()
	lookback := int64(k.TwapLookbackWindow(ctx))

	// Add new snapshot
	snapshots = append(snapshots, PriceSnapshot{Height: currentHeight, Price: price})

	// Prune old snapshots (keep only those within lookback window)
	pruned := []PriceSnapshot{}
	for _, snap := range snapshots {
		if currentHeight-snap.Height <= lookback {
			pruned = append(pruned, snap)
		}
	}

	// Encode and store
	store := ctx.KVStore(k.storeKey)
	var bz []byte
	for _, snap := range pruned {
		heightBytes := sdk.Uint64ToBigEndian(uint64(snap.Height))
		priceBytes := k.cdc.MustMarshal(&sdk.DecProto{Dec: snap.Price})
		// Store length + data for variable-length protobuf (4 bytes for length)
		priceLen := uint32(len(priceBytes))
		lengthBytes := []byte{
			byte(priceLen >> 24),
			byte(priceLen >> 16),
			byte(priceLen >> 8),
			byte(priceLen),
		}
		bz = append(bz, heightBytes...)
		bz = append(bz, lengthBytes...)
		bz = append(bz, priceBytes...)
	}

	if len(bz) > 0 {
		store.Set(types.GetTWAPPriceKey(denom), bz)
	} else {
		store.Delete(types.GetTWAPPriceKey(denom))
	}
}

// ComputeTWAP calculates the time-weighted average price from snapshots
func (k Keeper) ComputeTWAP(ctx sdk.Context, denom string) (math.LegacyDec, error) {
	snapshots := k.GetTWAPPrices(ctx, denom)
	if len(snapshots) == 0 {
		return math.LegacyZeroDec(), fmt.Errorf("no TWAP data for %s", denom)
	}

	// Simple average for now (could be improved to true time-weighted)
	sum := math.LegacyZeroDec()
	for _, snap := range snapshots {
		sum = sum.Add(snap.Price)
	}
	return sum.QuoInt64(int64(len(snapshots))), nil
}

// -------- Daily cap tracking --------

func (k Keeper) GetDailyCapResetHeight(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DailyCapResetHeightKey)
	if bz == nil {
		return 0
	}
	return int64(sdk.BigEndianToUint64(bz))
}

func (k Keeper) SetDailyCapResetHeight(ctx sdk.Context, h int64) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(uint64(h))
	store.Set(types.DailyCapResetHeightKey, bz)
}

// GetDailyCapBaseline returns the baseline balance for a denom set at epoch change
func (k Keeper) GetDailyCapBaseline(ctx sdk.Context, denom string) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetDailyCapBaselineKey(denom))
	if bz == nil {
		return math.ZeroInt()
	}

	var amount sdk.IntProto
	k.cdc.MustUnmarshal(bz, &amount)
	return amount.Int
}

// SetDailyCapBaseline stores the baseline balance for a denom (set at epoch change)
func (k Keeper) SetDailyCapBaseline(ctx sdk.Context, denom string, amount math.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: amount})
	store.Set(types.GetDailyCapBaselineKey(denom), bz)
}

// GetDailyCapUsage returns the amount drained today for a denom
func (k Keeper) GetDailyCapUsage(ctx sdk.Context, denom string) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetDailyCapUsageKey(denom))
	if bz == nil {
		return math.ZeroInt()
	}

	var amount sdk.IntProto
	k.cdc.MustUnmarshal(bz, &amount)
	return amount.Int
}

// SetDailyCapUsage stores the amount drained today for a denom
func (k Keeper) SetDailyCapUsage(ctx sdk.Context, denom string, amount math.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: amount})
	store.Set(types.GetDailyCapUsageKey(denom), bz)
}

// ResetDailyCapIfNeeded resets daily usage counters if a day has passed
func (k Keeper) ResetDailyCapIfNeeded(ctx sdk.Context) {
	lastReset := k.GetDailyCapResetHeight(ctx)
	currentHeight := ctx.BlockHeight()

	// Reset every 14,400 blocks (1 day at 3s/block)
	if lastReset == 0 || currentHeight-lastReset >= int64(core.BlocksPerDay) {
		// Clear all daily usage counters
		// Note: We iterate through all denoms that have baselines
		store := ctx.KVStore(k.storeKey)
		iterator := storetypes.KVStorePrefixIterator(store, types.DailyCapUsageKey)
		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			store.Delete(iterator.Key())
		}

		k.SetDailyCapResetHeight(ctx, currentHeight)
	}
}

// AfterOracleTally is called by the oracle module after a tally completes
// This updates the last tally timestamp and TWAP prices for all denoms
func (k Keeper) AfterOracleTally(ctx sdk.Context) {
	currentTime := ctx.BlockTime().Unix()
	k.SetLastOracleTallyTime(ctx, currentTime)

	// Update TWAP for USTC price (MetaUSDDenom - USTC price in USD)
	ustcPrice, err := k.OracleKeeper.GetLunaExchangeRate(ctx, oracletypes.MetaUSDDenom)
	if err == nil && ustcPrice.IsPositive() {
		k.AddTWAPPrice(ctx, oracletypes.MetaUSDDenom, ustcPrice)
	}

	// Update TWAP for all denoms including LUNC (LUNC price in each currency)
	// These are GetLunaExchangeRate for each denom (e.g., ukrw returns LUNC price in KRW)
	denoms := []string{
		core.MicroUSDDenom, // LUNC price in USD
		core.MicroKRWDenom, core.MicroSDRDenom, core.MicroCNYDenom,
		core.MicroJPYDenom, core.MicroEURDenom, core.MicroGBPDenom,
		core.MicroMNTDenom,
	}

	for _, denom := range denoms {
		price, err := k.OracleKeeper.GetLunaExchangeRate(ctx, denom)
		if err == nil && price.IsPositive() {
			k.AddTWAPPrice(ctx, denom, price)
		}
	}
}

// CheckAndUpdateDailyCapForSwap checks if a proposed swap would exceed daily cap limits and updates usage
// Each day allows draining up to DailyCapFactor × baseline (e.g. 10% of 1M = 100k per day)
// When you swap A→B, you drain B and add A. Adding A back reduces B's drainage counter.
func (k Keeper) CheckAndUpdateDailyCapForSwap(ctx sdk.Context, offerCoin sdk.Coin, askCoin sdk.Coin) error {
	k.ResetDailyCapIfNeeded(ctx)

	dailyCapFactor := k.DailyCapFactor(ctx)

	// Check the ask denom (what's being drained from the pool)
	askBaseline := k.GetDailyCapBaseline(ctx, askCoin.Denom)
	if askBaseline.IsZero() {
		// No baseline yet - allow swap (first epoch or denom not in pool at last epoch)
		return nil
	}

	// Calculate daily cap for this denom
	dailyCap := dailyCapFactor.MulInt(askBaseline).TruncateInt()

	// Get current daily usage (how much has been drained today)
	currentUsage := k.GetDailyCapUsage(ctx, askCoin.Denom)

	// When we offer the same denom that was previously drained, we reduce its usage
	// Example: Day 1 drain 80k LUNC, Day 1 add back 40k LUNC → net drainage = 40k
	if offerCoin.Denom == askCoin.Denom {
		// This shouldn't happen in normal swaps (can't swap LUNC for LUNC)
		return nil
	}

	// Check if the offer denom was previously drained - if so, this swap adds it back
	offerBaseline := k.GetDailyCapBaseline(ctx, offerCoin.Denom)
	if !offerBaseline.IsZero() {
		// Reduce the ask denom usage by the amount we're adding back via offer
		// This is the key insight: if we drained LUNC and now offer LUNC, we're undoing the drainage
		offerUsage := k.GetDailyCapUsage(ctx, offerCoin.Denom)
		if offerUsage.IsPositive() {
			// We're adding back a denom that was previously drained
			reduction := offerCoin.Amount
			if reduction.GT(offerUsage) {
				reduction = offerUsage
			}
			k.SetDailyCapUsage(ctx, offerCoin.Denom, offerUsage.Sub(reduction))
		}
	}

	// Calculate new usage after draining askCoin
	newUsage := currentUsage.Add(askCoin.Amount)

	// Check if new usage would exceed daily cap
	if newUsage.GT(dailyCap) {
		return types.ErrDailyCapExceeded
	}

	// Update usage for ask denom (only if check passed)
	k.SetDailyCapUsage(ctx, askCoin.Denom, newUsage)

	return nil
}
