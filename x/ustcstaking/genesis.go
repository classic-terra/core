package ustcstaking

import (
	"fmt"

	"github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesisState() *types.GenesisState {
	return types.DefaultGenesisState()
}

func ValidateGenesis(data *types.GenesisState) error {
	return types.ValidateGenesis(data)
}

func InitGenesis(ctx sdk.Context, k keeper.Keeper, data *types.GenesisState) {
	if data == nil {
		panic(fmt.Sprintf("invalid %s genesis: %s", types.ModuleName, types.ErrInvalidGenesis.Wrap("genesis is nil")))
	}
	if err := ValidateGenesis(data); err != nil {
		panic(fmt.Sprintf("invalid %s genesis: %s", types.ModuleName, err))
	}
	normalized := *data
	state := data.RewardState
	if state.LastRewardTime == nil {
		now := ctx.BlockTime()
		state.LastRewardTime = &now
	} else if state.LastRewardTime.After(ctx.BlockTime()) {
		panic(fmt.Sprintf("invalid %s genesis: %s", types.ModuleName, types.ErrInvalidRewardState.Wrap("last reward time is in the future")))
	}
	normalized.RewardState = state
	if err := ValidateGenesis(&normalized); err != nil {
		panic(fmt.Sprintf("invalid %s genesis: %s", types.ModuleName, err))
	}
	k.SetParams(ctx, normalized.Params)
	k.SetRewardState(ctx, normalized.RewardState)
	for _, position := range normalized.Positions {
		if err := k.SetPosition(ctx, position); err != nil {
			panic(fmt.Sprintf("invalid %s position: %s", types.ModuleName, err))
		}
	}
	k.SetNextPositionID(ctx, normalized.NextPositionId)
	if err := k.RebuildDerivedState(ctx); err != nil {
		panic(fmt.Sprintf("invalid %s derived state: %s", types.ModuleName, err))
	}
	// Bank-backed validation runs after all genesis modules initialize, or after
	// upgrade migrations, so incidental balances cannot panic during activation.
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	positions := make([]types.Position, 0)
	k.IteratePositions(ctx, func(position types.Position) bool {
		positions = append(positions, position)
		return false
	})
	state := types.NewGenesisState(k.GetParams(ctx), positions, k.GetRewardState(ctx), k.GetNextPositionID(ctx))
	if err := ValidateGenesis(state); err != nil {
		panic(fmt.Sprintf("invalid exported %s genesis: %s", types.ModuleName, err))
	}
	return state
}
