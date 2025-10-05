package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/classic-terra/core/v3/x/market/types"
)

// Simulation parameter constants
const (
	basePoolKey           = "base_pool"
	poolRecoveryPeriodKey = "pool_recovery_period"
	minStabilitySpreadKey = "min_spread"
)

// GenBasePool randomized MintBasePool
func GenBasePool(r *rand.Rand) math.LegacyDec {
	return math.LegacyNewDec(50000000000000).Add(math.LegacyNewDec(int64(r.Intn(10000000000))))
}

// GenPoolRecoveryPeriod randomized PoolRecoveryPeriod
func GenPoolRecoveryPeriod(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(10000000000))
}

// GenMinSpread randomized MinSpread
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

	marketGenesis := types.NewGenesisState(
		math.LegacyZeroDec(),
		types.Params{
			BasePool:           basePool,
			PoolRecoveryPeriod: poolRecoveryPeriod,
			MinStabilitySpread: minStabilitySpread,
		},
	)

	bz, err := json.MarshalIndent(&marketGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated market parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(marketGenesis)
}
