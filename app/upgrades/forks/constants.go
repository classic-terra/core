package forks

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	"github.com/classic-terra/core/v2/types/fork"
)

var DisableSwapFork = upgrades.Fork{
	UpgradeName:    "v0.5.20",
	UpgradeHeight:  fork.SwapDisableHeight,
	BeginForkLogic: runForkLogicSwapDisable,
}

var IbcEnableFork = upgrades.Fork{
	UpgradeName:    "v0.5.23",
	UpgradeHeight:  fork.IbcEnableHeight,
	BeginForkLogic: runForkLogicIbcEnable,
}

var VersionMapEnableFork = upgrades.Fork{
	UpgradeName:    "v1.0.5",
	UpgradeHeight:  fork.VersionMapEnableHeight,
	BeginForkLogic: runForkLogicVersionMapEnable,
}

var Freeze800MFork = upgrades.Fork{
	UpgradeName:    "v2.2.3",
	UpgradeHeight:  fork.Freeze800MHeight,
	BeginForkLogic: runForkLogicBlacklist800M,
}

var Freeze800MForkRebel = upgrades.Fork{
	UpgradeName:    "v2.2.3",
	UpgradeHeight:  fork.Freeze800MHeightRebel,
	BeginForkLogic: runForkLogicBlacklist800MRebel,
}
