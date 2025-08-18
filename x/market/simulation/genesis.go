package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/types"
)

// Simulation parameter constants
const (
	basePoolKey           = "base_pool"
	poolRecoveryPeriodKey = "pool_recovery_period"
	minStabilitySpreadKey = "min_spread"
)

// GenBasePool randomized MintBasePool
func GenBasePool(r *rand.Rand) sdk.Dec {
	return sdk.NewDec(50000000000000).Add(sdk.NewDec(int64(r.Intn(10000000000))))
}

// GenPoolRecoveryPeriod randomized PoolRecoveryPeriod
func GenPoolRecoveryPeriod(r *rand.Rand) uint64 {
	return uint64(100 + r.Intn(10000000000))
}

// GenMinSpread randomized MinSpread
func GenMinSpread(r *rand.Rand) sdk.Dec {
	return sdk.NewDecWithPrec(int64(r.Intn(3)), 2)
}

// GenEpochLengthBlocks randomized EpochLengthBlocks
func GenEpochLengthBlocks(r *rand.Rand) uint64 {
	// between 7 and 60 days worth of blocks
	days := 7 + r.Intn(54)
	return uint64(days) * uint64(core.BlocksPerDay)
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var basePool sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, string(types.KeyBasePool), &basePool, nil,
		func(r *rand.Rand) { basePool = GenBasePool(r) },
	)

	var poolRecoveryPeriod uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, poolRecoveryPeriodKey, &poolRecoveryPeriod, simState.Rand,
		func(r *rand.Rand) { poolRecoveryPeriod = GenPoolRecoveryPeriod(r) },
	)

	var minStabilitySpread sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, string(types.KeyMinStabilitySpread), &minStabilitySpread, nil,
		func(r *rand.Rand) { minStabilitySpread = GenMinSpread(r) },
	)

	var epochLengthBlocks uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, string(types.KeyEpochLengthBlocks), &epochLengthBlocks, nil,
		func(r *rand.Rand) { epochLengthBlocks = GenEpochLengthBlocks(r) },
	)

	params := types.Params{
		BasePool:           basePool,
		PoolRecoveryPeriod: poolRecoveryPeriod,
		MinStabilitySpread: minStabilitySpread,
		EpochLengthBlocks:  epochLengthBlocks,
	}

	marketGenesis := types.NewGenesisState(
		sdk.ZeroDec(),
		params,
	)

	bz, err := json.MarshalIndent(&marketGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated market parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(marketGenesis)
}
