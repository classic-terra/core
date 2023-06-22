package v1

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	core "github.com/classic-terra/core/v2/types"
)

var (

	// UpgradeName defines the set of on-chain upgrade name for the classic terra v1 upgrade.
	UpgradeNames = []string{"v1.0.0", "v1.0.5", "v1.1.0"}
	// UpgradeHeight defines the set of block heights at which the classic terra v1 upgrade is
	// triggered.
	UpgradeHeights = []int64{core.SwapDisableForkHeight, core.SwapEnableForkHeight, core.VersionMapEnableHeight}
	// ForkLogicFuncs defines the set of functions that are run at the beginning
	ForkLogicFuncs = []upgrades.ForkLogicFunc{runForkLogic1_0_0, runForkLogic1_0_5, runForkLogic1_1_0}
)

var Forks = []upgrades.Fork{}

func init() {
	for i := range UpgradeNames {
		Forks = append(Forks, upgrades.Fork{
			UpgradeName:    UpgradeNames[i],
			UpgradeHeight:  UpgradeHeights[i],
			BeginForkLogic: ForkLogicFuncs[i],
		})
	}
}
