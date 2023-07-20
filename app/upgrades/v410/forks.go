package v410

import (
	"github.com/classic-terra/core/v2/app/keepers"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func runContractStargateEnable(ctx sdk.Context, keppers *keepers.AppKeepers, _ *module.Manager) {
	ctx.Logger().Info("runContractStargateEnable")
}
