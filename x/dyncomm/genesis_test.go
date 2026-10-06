package dyncomm_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/dyncomm"
	dyncommkeeper "github.com/classic-terra/core/v4/x/dyncomm/keeper"
	dyncommtypes "github.com/classic-terra/core/v4/x/dyncomm/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisRoundTrip(t *testing.T) {
	input := dyncommkeeper.CreateTestInput(t)

	operator := dyncommkeeper.ValAddrFrom(0).String()
	minRate := math.LegacyNewDecWithPrec(5, 2)
	targetRate := math.LegacyNewDecWithPrec(3, 2)
	input.DyncommKeeper.SetDynCommissionRate(input.Ctx, operator, minRate)
	input.DyncommKeeper.SetTargetCommissionRate(input.Ctx, operator, targetRate)

	exported := dyncomm.ExportGenesis(input.Ctx, input.DyncommKeeper)
	require.Len(t, exported.ValidatorCommissionRates, 1)

	input2 := dyncommkeeper.CreateTestInput(t)
	dyncomm.InitGenesis(input2.Ctx, input2.DyncommKeeper, exported)

	require.Equal(t, minRate, input2.DyncommKeeper.GetDynCommissionRate(input2.Ctx, operator))
	require.Equal(t, targetRate, input2.DyncommKeeper.GetTargetCommissionRate(input2.Ctx, operator))
}

func TestGenesisRoundTripDefaultsValidatorsWithoutRate(t *testing.T) {
	input := dyncommkeeper.CreateTestInput(t)

	exported := dyncomm.ExportGenesis(input.Ctx, input.DyncommKeeper)
	require.Empty(t, exported.ValidatorCommissionRates)

	input2 := dyncommkeeper.CreateTestInput(t)
	dyncomm.InitGenesis(input2.Ctx, input2.DyncommKeeper, exported)

	require.Equal(t, dyncommtypes.DefaultParams(), input2.DyncommKeeper.GetParams(input2.Ctx))
}
