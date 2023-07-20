package v410

import (
	"github.com/classic-terra/core/v2/app/upgrades"
	"github.com/classic-terra/core/v2/types/fork"
)

var EnableWasmStargateQueryFork = upgrades.Fork{
	UpgradeName:    "v4.1.0",
	UpgradeHeight:  fork.WasmStargateQueryEnableHeight,
	BeginForkLogic: runContractStargateEnable,
}
