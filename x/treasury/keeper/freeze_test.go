package keeper

import (
	"fmt"
	"testing"

	"github.com/classic-terra/core/v2/x/treasury/types"
	"github.com/stretchr/testify/require"
)

func TestFreezeListSetGet(t *testing.T) {
	input := CreateTestInput(t)

	// Should be empty after initialization
	freeze := input.TreasuryKeeper.GetFreezeAddrs(input.Ctx)
	require.Equal(t, 0, len(freeze.Frozen))

	// Setting
	freeze2 := types.NewFreezeList()
	freeze2.Add("test2")
	freeze2.Add("test3")
	input.TreasuryKeeper.SetFreezeAddrs(input.Ctx, freeze2)
	fmt.Printf("freeze2: %v\n", freeze2)

	// Should contain correct data after receive
	freeze3 := input.TreasuryKeeper.GetFreezeAddrs(input.Ctx)
	fmt.Printf("freeze3: %v\n", freeze3)
	require.Equal(t, 2, len(freeze3.Frozen))
	require.Equal(t, "test2", freeze3.Frozen[0])
	require.Equal(t, "test3", freeze3.Frozen[1])

}
