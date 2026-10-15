package simulation

import (
	"math/rand"

	"cosmossdk.io/math"

	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	sdksim "github.com/cosmos/cosmos-sdk/x/simulation"
)

// ProposalMsgs are submitted and voted on by the SDK governance simulator;
// their signer is the governance module account, never a privileged trader.
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	return []simtypes.WeightedProposalMsg{
		sdksim.NewWeightedProposalMsg("ustcstaking_update_params", 20, func(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
			params := types.DefaultParams()
			params.RewardRatePerDay = math.NewInt(r.Int63n(1_000_001))
			params.Paused = r.Intn(5) == 0
			return &types.MsgUpdateParams{Authority: authority, Params: params}
		}),
		sdksim.NewWeightedProposalMsg("ustcstaking_fund_rewards", 100, func(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
			return &types.MsgFundRewards{Authority: authority, Amount: sdk.NewInt64Coin(types.BondDenom, 1+r.Int63n(1_000_000))}
		}),
		sdksim.NewWeightedProposalMsg("ustcstaking_return_reward_dust", 10, func(_ *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
			return &types.MsgReturnRewardDust{Authority: authority, Amount: sdk.NewInt64Coin(types.BondDenom, 1)}
		}),
	}
}
