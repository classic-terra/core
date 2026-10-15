package app

import (
	"testing"

	ustcstakingupgrade "github.com/classic-terra/core/v4/app/upgrades/ustc_staking"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradesIncludeUSTCStakingStoreUpgrade(t *testing.T) {
	found := false
	for _, upgrade := range Upgrades {
		if upgrade.UpgradeName == ustcstakingupgrade.UpgradeName {
			found = true
			require.Equal(t, []string{ustcstakingtypes.StoreKey}, upgrade.StoreUpgrades.Added)
		}
	}
	require.True(t, found)
}
