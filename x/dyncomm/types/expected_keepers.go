package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper is expected keeper for auth module
type StakingKeeper interface {
	MinCommissionRate(ctx sdk.Context) sdk.Dec
	GetLastTotalPower(ctx sdk.Context) math.Int
	PowerReduction(ctx sdk.Context) math.Int
}
