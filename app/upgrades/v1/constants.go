package v1

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	"github.com/classic-terra/core/v2/types"
)

var DisableSwapFork = upgrades.Fork{
	UpgradeName:    "v0.5.20",
	UpgradeHeight:  types.SwapDisableHeight,
	BeginForkLogic: runForkLogicSwapDisable,
}

var IbcEnableFork = upgrades.Fork{
	UpgradeName:    "v1.0.4",
	UpgradeHeight:  types.IbcEnableHeight,
	BeginForkLogic: runForkLogicIbcEnable,
}

var VersionMapEnableFork = upgrades.Fork{
	UpgradeName:    "v1.0.5",
	UpgradeHeight:  types.VersionMapEnableHeight,
	BeginForkLogic: runForkLogicVersionMapEnable,
}
