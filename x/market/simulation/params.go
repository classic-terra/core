package simulation

// DONTCOVER

import (
	"fmt"
	"math/rand"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/classic-terra/core/v2/x/market/types"
)

// ParamChanges defines the parameters that can be modified by param change proposals
// on the simulation
func ParamChanges(*rand.Rand) []simtypes.LegacyParamChange {
	return []simtypes.LegacyParamChange{
		simulation.NewSimLegacyParamChange(types.ModuleName, string(types.KeyBasePool),
			func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", GenBasePool(r))
			},
		),
		simulation.NewSimLegacyParamChange(types.ModuleName, string(types.KeyPoolRecoveryPeriod),
			func(r *rand.Rand) string {
				return fmt.Sprintf("\"%d\"", GenPoolRecoveryPeriod(r))
			},
		),
		simulation.NewSimLegacyParamChange(types.ModuleName, string(types.KeyMinStabilitySpread),
			func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", GenMinSpread(r))
			},
		),
	}
}
