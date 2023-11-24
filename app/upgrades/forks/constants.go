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

type Acc struct {
	Address       string
	AccountNumber uint64
}

var affectedAccounts = []Acc{}

var AccAddressFixFork = upgrades.Fork{
	UpgradeName:    "v2.3.1",
	UpgradeHeight:  fork.AccAddressFixHeight,
	BeginForkLogic: runForkLogicCorrectAccountSequence,
}
