package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// Simulation parameter constants
const (
	basePoolKey             = "base_pool"
	poolRecoveryPeriodKey   = "pool_recovery_period"
	minStabilitySpreadKey   = "min_spread"
	swapFeeBurnRateKey      = "swap_fee_burn_rate"
	swapFeeCommunityRateKey = "swap_fee_community_rate"
	maxOracleAgeSecondsKey  = "max_oracle_age_seconds"
	twapLookbackWindowKey   = "twap_lookback_window"
	maxTWAPDeviationKey     = "max_twap_deviation"
	dailyCapFactorKey       = "daily_cap_factor"
)

// GenBasePool randomized MintBasePool
func GenBasePool(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDec(50000000000000).Add(math.LegacyNewDec(int64(r.Intn(10000000000))))
}

// GenPoolRecoveryPeriod randomized PoolRecoveryPeriod
func GenPoolRecoveryPeriod(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(10000000000))
}

// GenEpochLengthBlocks randomized EpochLengthBlocks
func GenEpochLengthBlocks(r *rand.Rand) uint64 {
	// between 7 and 60 days worth of blocks
	days := 7 + r.Intn(54)
	return uint64(days) * core.BlocksPerDay
}

// GenSwapFeeBurnRate randomized SwapFeeBurnRate in [0, 0.5].
// Kept below 0.5 so that burn + community rates can never exceed 1 together
// (validated by Params.Validate).
func GenSwapFeeBurnRate(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDecWithPrec(int64(r.Intn(501)), 3) // [0, 0.500]
}

// GenSwapFeeCommunityRate randomized SwapFeeCommunityRate in [0, 0.5].
func GenSwapFeeCommunityRate(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDecWithPrec(int64(r.Intn(501)), 3) // [0, 0.500]
}

// GenMaxOracleAgeSeconds randomized MaxOracleAgeSeconds in [30, 300] seconds.
func GenMaxOracleAgeSeconds(r *rand.Rand) uint64 {
	return uint64(30 + r.Intn(271))
}

// GenTWAPLookbackWindow randomized TWAPLookbackWindow in [10, 200] blocks.
func GenTWAPLookbackWindow(r *rand.Rand) uint64 {
	return uint64(10 + r.Intn(191))
}

// GenMaxTWAPDeviation randomized MaxTWAPDeviation in [0, 0.5].
func GenMaxTWAPDeviation(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDecWithPrec(int64(r.Intn(501)), 3) // [0, 0.500]
}

// GenDailyCapFactor randomized DailyCapFactor in [0, 0.5].
func GenDailyCapFactor(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDecWithPrec(int64(r.Intn(501)), 3) // [0, 0.500]
}

func GenMinSpread(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDecWithPrec(1, 2).Add(math.LegacyNewDecWithPrec(int64(r.Intn(100)), 3))
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var basePool math.LegacyDec
	simState.AppParams.GetOrGenerate(
		basePoolKey, &basePool, simState.Rand,
		func(r *rand.Rand) { basePool = GenBasePool(r) },
	)

	var poolRecoveryPeriod uint64
	simState.AppParams.GetOrGenerate(
		poolRecoveryPeriodKey, &poolRecoveryPeriod, simState.Rand,
		func(r *rand.Rand) { poolRecoveryPeriod = GenPoolRecoveryPeriod(r) },
	)

	var minStabilitySpread math.LegacyDec
	simState.AppParams.GetOrGenerate(
		minStabilitySpreadKey, &minStabilitySpread, simState.Rand,
		func(r *rand.Rand) { minStabilitySpread = GenMinSpread(r) },
	)

	var epochLengthBlocks uint64
	simState.AppParams.GetOrGenerate(
		string(types.KeyEpochLengthBlocks), &epochLengthBlocks, simState.Rand,
		func(r *rand.Rand) { epochLengthBlocks = GenEpochLengthBlocks(r) },
	)

	var swapFeeBurnRate math.LegacyDec
	simState.AppParams.GetOrGenerate(
		swapFeeBurnRateKey, &swapFeeBurnRate, simState.Rand,
		func(r *rand.Rand) { swapFeeBurnRate = GenSwapFeeBurnRate(r) },
	)

	var swapFeeCommunityRate math.LegacyDec
	simState.AppParams.GetOrGenerate(
		swapFeeCommunityRateKey, &swapFeeCommunityRate, simState.Rand,
		func(r *rand.Rand) { swapFeeCommunityRate = GenSwapFeeCommunityRate(r) },
	)

	var maxOracleAgeSeconds uint64
	simState.AppParams.GetOrGenerate(
		maxOracleAgeSecondsKey, &maxOracleAgeSeconds, simState.Rand,
		func(r *rand.Rand) { maxOracleAgeSeconds = GenMaxOracleAgeSeconds(r) },
	)

	var twapLookbackWindow uint64
	simState.AppParams.GetOrGenerate(
		twapLookbackWindowKey, &twapLookbackWindow, simState.Rand,
		func(r *rand.Rand) { twapLookbackWindow = GenTWAPLookbackWindow(r) },
	)

	var maxTWAPDeviation math.LegacyDec
	simState.AppParams.GetOrGenerate(
		maxTWAPDeviationKey, &maxTWAPDeviation, simState.Rand,
		func(r *rand.Rand) { maxTWAPDeviation = GenMaxTWAPDeviation(r) },
	)

	var dailyCapFactor math.LegacyDec
	simState.AppParams.GetOrGenerate(
		dailyCapFactorKey, &dailyCapFactor, simState.Rand,
		func(r *rand.Rand) { dailyCapFactor = GenDailyCapFactor(r) },
	)

	marketGenesis := types.NewGenesisState(
		math.LegacyZeroDec(),
		types.Params{
			BasePool:             basePool,
			PoolRecoveryPeriod:   poolRecoveryPeriod,
			MinStabilitySpread:   minStabilitySpread,
			EpochLengthBlocks:    epochLengthBlocks,
			SwapFeeBurnRate:      swapFeeBurnRate,
			SwapFeeCommunityRate: swapFeeCommunityRate,
			MaxOracleAgeSeconds:  maxOracleAgeSeconds,
			TwapLookbackWindow:   twapLookbackWindow,
			MaxTwapDeviation:     maxTWAPDeviation,
			DailyCapFactor:       dailyCapFactor,
		},
	)

	bz, err := json.MarshalIndent(&marketGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated market parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(marketGenesis)
}
