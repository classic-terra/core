package app

import (
	"testing"

	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradesIncludeUSTCStakingStoreUpgrade(t *testing.T) {
	found := false
	for _, upgrade := range Upgrades {
		if upgrade.UpgradeName == "v14_5" {
			found = true
			require.Equal(t, []string{ustcstakingtypes.StoreKey}, upgrade.StoreUpgrades.Added)
		}
	}
	require.True(t, found)
}
