package types

import (
	context "context"

	"cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AccountKeeper is expected keeper for auth module
type StakingKeeper interface {
	MinCommissionRate(ctx context.Context) (math.LegacyDec, error)
	GetLastTotalPower(ctx context.Context) (math.Int, error)
	PowerReduction(ctx context.Context) math.Int
	IterateValidators(context.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	SetValidator(ctx context.Context, validator stakingtypes.Validator) error
}
